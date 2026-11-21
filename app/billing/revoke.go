package billing

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/AlexGalhardo/vukem/contrib/stripe"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// charge is the part of a Stripe charge the refund and dispute rules read.
type charge struct {
	ID             string `json:"id"`
	Amount         int64  `json:"amount"`
	AmountRefunded int64  `json:"amount_refunded"`
	Invoice        string `json:"invoice"`
}

// dispute is the part of a Stripe dispute the rules read.
type dispute struct {
	ID     string `json:"id"`
	Charge string `json:"charge"`
	Status string `json:"status"`
}

// get reads one Stripe object; the id is escaped into the path.
func (b *Billing) get(ctx context.Context, path, id string, out any) error {
	if b.Stripe == nil {
		return ErrNotConfigured
	}

	return sanitizeStripe(b.Stripe.Call(ctx, http.MethodGet, path+url.PathEscape(id), nil, out))
}

// subscriptionOfCharge returns the id of the subscription a charge paid, or ""
// for a charge that is not the payment of a subscription invoice.
func (b *Billing) subscriptionOfCharge(ctx context.Context, ch charge) (string, error) {
	if ch.Invoice == "" {
		return "", nil
	}

	var inv invoice
	if err := b.get(ctx, "/v1/invoices/", ch.Invoice, &inv); err != nil {
		return "", err
	}

	return inv.subscription(), nil
}

// handleRefund reads the charge (the event is only a pointer to it, so a late
// or repeated event gives the same answer) and revokes access on a full refund
// of a subscription payment. A partial refund keeps access, and so does the
// refund of an invoice whose period has ended: the current period was paid by
// another invoice. An invoice that does not say its period counts as current.
func (b *Billing) handleRefund(ctx context.Context, chargeID string) error {
	var ch charge
	if err := b.get(ctx, "/v1/charges/", chargeID, &ch); err != nil {
		return err
	}

	subID, err := b.subscriptionOfCharge(ctx, ch)
	if err != nil || subID == "" {
		return err
	}

	if ch.Amount <= 0 || ch.AmountRefunded < ch.Amount {
		return b.noteRefund(ctx, subID, ch, "partial_refund", "Partial refund: access kept")
	}

	var inv invoice
	if err := b.get(ctx, "/v1/invoices/", ch.Invoice, &inv); err != nil {
		return err
	}

	if end := inv.serviceEnd(); end > 0 && end <= b.Now().Unix() {
		return b.noteRefund(ctx, subID, ch, "past_period_refund", "Refund of an ended period: access kept")
	}

	return b.revokeSubscription(ctx, subID, models.RevokedRefund, ch.ID)
}

// noteRefund writes one billing event per refunded amount.
func (b *Billing) noteRefund(ctx context.Context, subID string, ch charge, kind, summary string) error {
	row, ok, err := b.subscriptionRow(ctx, subID)
	if err != nil || !ok {
		return err
	}

	ref := ch.ID + ":" + itoa(ch.AmountRefunded)

	seen, err := orm.QueryOf[models.BillingEvent](b.DB).Where("workspace_id", row.WorkspaceID).
		Where("kind", kind).Where("ref", ref).Exists(ctx)
	if err != nil || seen {
		return err
	}

	return record(ctx, b.DB, row.WorkspaceID, SourceStripe, kind, ref, summary)
}

// handleDispute applies the current state of a dispute. Created and closed
// events do the same thing, so their order does not matter.
func (b *Billing) handleDispute(ctx context.Context, disputeID string) error {
	var d dispute
	if err := b.get(ctx, "/v1/disputes/", disputeID, &d); err != nil {
		return err
	}

	var ch charge
	if err := b.get(ctx, "/v1/charges/", d.Charge, &ch); err != nil {
		return err
	}

	subID, err := b.subscriptionOfCharge(ctx, ch)
	if err != nil || subID == "" {
		return err
	}

	switch d.Status {
	case "won", "warning_closed", "prevented":
		return b.reinstate(ctx, subID, ch.ID)
	case "lost":
		_, err := orm.QueryOf[models.WorkspaceSubscription](b.DB).
			Where("stripe_subscription_id", subID).Where("revoked_reason", models.RevokedDispute).
			Where("revoked_ref", ch.ID).
			Update(ctx, orm.Set{"revoked_reason": models.RevokedDisputeLost})
		if err != nil {
			return err
		}

		return b.revokeSubscription(ctx, subID, models.RevokedDisputeLost, ch.ID)
	default:
		return b.revokeSubscription(ctx, subID, models.RevokedDispute, ch.ID)
	}
}

// subscriptionRow returns the local row of a Stripe subscription, syncing it
// first when an event for it arrives before any other one did.
func (b *Billing) subscriptionRow(ctx context.Context, subID string) (models.WorkspaceSubscription, bool, error) {
	find := func() (models.WorkspaceSubscription, error) {
		return orm.QueryOf[models.WorkspaceSubscription](b.DB).Where("stripe_subscription_id", subID).First(ctx)
	}

	row, err := find()
	if errors.Is(err, orm.ErrNotFound) {
		if err := b.SyncSubscription(ctx, subID); err != nil {
			return row, false, err
		}

		row, err = find()
	}

	if errors.Is(err, orm.ErrNotFound) {
		return row, false, nil
	}

	return row, err == nil, err
}

// revokeSubscription takes the Pro access of a subscription away, once: the
// conditional UPDATE changes a row only while it is not revoked, and only that
// change writes the billing event and emails the owners. Revoked access is
// then cancelled at Stripe so the card is not billed again; a failed cancel
// answers an error so Stripe delivers the event again, and the retry only
// repeats the cancel.
func (b *Billing) revokeSubscription(ctx context.Context, subID, reason, ref string) error {
	row, ok, err := b.subscriptionRow(ctx, subID)
	if err != nil || !ok {
		return err
	}

	var changed int64

	err = b.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.WorkspaceSubscription](tx).
			Where("stripe_subscription_id", subID).WhereNull("access_revoked_at").
			Update(ctx, orm.Set{
				"access_revoked_at": b.Now().UTC(), "revoked_reason": reason, "revoked_ref": ref,
			})
		if err != nil || n == 0 {
			return err
		}

		changed = n

		return record(ctx, tx, row.WorkspaceID, SourceStripe, "access_revoked", subID, "Access revoked: "+reason)
	})
	if err != nil {
		return err
	}

	if changed > 0 {
		b.notifyRevoked(ctx, row.WorkspaceID, reason)
	}

	if reason == models.RevokedDispute {
		return nil // an open dispute may still be won: cancel only on a refund or a lost dispute
	}

	return b.cancelSubscription(ctx, subID)
}

// cancelSubscription cancels subID at Stripe at once, with no proration and no
// final invoice, unless the local row already says it is canceled.
func (b *Billing) cancelSubscription(ctx context.Context, subID string) error {
	row, err := orm.QueryOf[models.WorkspaceSubscription](b.DB).Where("stripe_subscription_id", subID).First(ctx)
	if err != nil || row.Status == "canceled" {
		return err
	}

	err = b.Stripe.Call(ctx, http.MethodDelete, "/v1/subscriptions/"+url.PathEscape(subID),
		stripe.Params{"prorate": false, "invoice_now": false}, nil)
	if err != nil && !errors.Is(err, stripe.ErrNotFound) {
		return sanitizeStripe(err)
	}

	_, err = orm.QueryOf[models.WorkspaceSubscription](b.DB).Where("stripe_subscription_id", subID).
		Update(ctx, orm.Set{"status": "canceled"})

	return err
}

// reinstate clears a revocation after a won dispute, only when it was this
// dispute's (reason dispute, same charge), so a won dispute never undoes a
// refund. The subscription's status does not matter here: Valid still requires
// a paying one, and a past_due subscription gets access back when it recovers.
func (b *Billing) reinstate(ctx context.Context, subID, chargeID string) error {
	row, ok, err := b.subscriptionRow(ctx, subID)
	if err != nil || !ok {
		return err
	}

	if err := b.SyncSubscription(ctx, subID); err != nil {
		return err
	}

	return b.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.WorkspaceSubscription](tx).
			Where("stripe_subscription_id", subID).Where("revoked_reason", models.RevokedDispute).
			Where("revoked_ref", chargeID).
			Update(ctx, orm.Set{"access_revoked_at": nil, "revoked_reason": "", "revoked_ref": ""})
		if err != nil || n == 0 {
			return err
		}

		return record(ctx, tx, row.WorkspaceID, SourceStripe, "access_reinstated", subID, "Dispute won: access restored")
	})
}

// revokedText is the sentence the email gives for each reason.
var revokedText = map[string]string{
	models.RevokedRefund:      "the payment was refunded",
	models.RevokedDispute:     "the payment is under dispute with the card issuer",
	models.RevokedDisputeLost: "the card issuer ruled against the payment in a dispute",
	models.RevokedAdmin:       "a support administrator ended it",
}

// notifyRevoked tells the owners of the workspace why Pro ended.
func (b *Billing) notifyRevoked(ctx context.Context, workspaceID int64, reason string) {
	ws, err := orm.QueryOf[models.Workspace](b.DB).Find(ctx, workspaceID)
	if err != nil {
		b.Log.WarnContext(ctx, "billing: workspace not found for a revocation mail", "workspace", workspaceID)
		return
	}

	owners, err := b.WS.OwnerEmails(ctx, workspaceID)
	if err != nil {
		b.Log.WarnContext(ctx, "billing: owners not found", "workspace", workspaceID)
		return
	}

	for _, to := range owners {
		var m mail.Mailable = appmail.AccessRevoked{
			To: to, Workspace: ws.Name, Reason: revokedText[reason], URL: b.URL + "/billing",
		}
		if err := b.WS.Notify(ctx, m); err != nil {
			b.Log.WarnContext(ctx, "billing: email not sent", "workspace", workspaceID)
		}
	}
}
