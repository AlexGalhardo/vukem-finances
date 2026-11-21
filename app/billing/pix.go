package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// PixExpirySeconds is how long a PIX charge can be paid (30 minutes).
const PixExpirySeconds = 1800

var (
	errBadMonths = errs.New(http.StatusUnprocessableEntity, "Choose one of the offered periods.")
	errPixOff    = errs.New(http.StatusServiceUnavailable, "PIX payments are not configured.")
)

// StartPix creates a PIX charge for months of Pro. The amount is the server's
// price times the months; the request names only the months, and only one of
// PIX_MONTHS_OPTIONS is accepted. The caller has checked billing.manage.
func (b *Billing) StartPix(ctx context.Context, ws models.Workspace, actor int64, months int) (models.PixPayment, error) {
	if b.Abacate == nil {
		return models.PixPayment{}, errPixOff
	}

	cents, ok := b.Config.Pix.Amount(months)
	if !ok {
		return models.PixPayment{}, errBadMonths
	}

	ref := models.NewUID()

	ch, err := b.Abacate.CreatePix(ctx, CreatePixRequest{
		AmountCents: cents, Description: "Pro plan, " + strconv.Itoa(months) + " month(s)",
		ExpiresInSeconds: PixExpirySeconds, Metadata: map[string]string{"ref": ref},
	})
	if err != nil {
		return models.PixPayment{}, err
	}

	expires := ch.ExpiresAt.UTC()
	if expires.IsZero() {
		expires = b.Now().UTC().Add(PixExpirySeconds * time.Second)
	}

	p := models.PixPayment{
		PublicRef: ref, ProviderRef: ch.ID, Status: models.PixPending, AmountCents: cents, Currency: "BRL",
		Months: months, BRCode: ch.BRCode, QRPNGBase64: ch.BRCodeImage, ExpiresAt: expires, CreatedBy: actor,
	}

	err = b.DB.Tx(ctx, func(tx *database.Tx) error {
		tctx := workspaces.WithTenant(ctx, ws.ID)
		if err := orm.Create(tctx, tx, &p); err != nil {
			return err
		}

		return record(ctx, tx, ws.ID, SourcePix, "pix_created", ref, fmt.Sprintf("PIX charge for %d month(s)", months))
	})

	return p, err
}

// PixPayment returns the payment with public reference ref of the workspace of
// ctx; another workspace's reference is a 404.
func (b *Billing) PixPayment(ctx context.Context, ref string) (models.PixPayment, error) {
	p, err := workspaces.Scoped[models.PixPayment](ctx, b.DB).Where("public_ref", ref).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return p, errs.ErrNotFound
	}

	return p, err
}

// RefreshPix re-reads a pending or expired charge from the provider and applies
// what it says; it is what a poll does. A paid, refunded or disputed payment
// is returned as it is: the reconciliation job watches those.
func (b *Billing) RefreshPix(ctx context.Context, p models.PixPayment) (models.PixPayment, error) {
	if p.Status != models.PixPending && p.Status != models.PixExpired {
		return p, nil
	}

	if err := b.applyPixState(ctx, p); err != nil {
		return p, err
	}

	return b.pixByID(ctx, p.WorkspaceID, p.ID)
}

func (b *Billing) pixByID(ctx context.Context, workspaceID, id int64) (models.PixPayment, error) {
	return workspaces.Scoped[models.PixPayment](workspaces.WithTenant(ctx, workspaceID), b.DB).Where("id", id).First(ctx)
}

// HandlePixEvent is what a webhook delivery causes: it looks the charge up by
// the provider reference and re-reads it from the provider. The body of the
// delivery is never trusted for the outcome, only for which charge to check.
// An unknown charge is ignored.
func (b *Billing) HandlePixEvent(ctx context.Context, providerRef string) error {
	p, err := orm.QueryOf[models.PixPayment](b.DB).Where("provider_ref", providerRef).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	return b.applyPixState(ctx, p)
}

// applyPixState reads the charge at the provider and moves the local payment to
// match. Every move is a conditional UPDATE on the current status, so a repeat
// finds nothing to change.
func (b *Billing) applyPixState(ctx context.Context, p models.PixPayment) error {
	if b.Abacate == nil {
		return errPixOff
	}

	ch, err := b.Abacate.CheckPix(ctx, p.ProviderRef)
	if errors.Is(err, ErrCharge) {
		return nil
	}

	if err != nil {
		return err
	}

	switch ch.Status {
	case ProviderPaid:
		switch p.Status {
		case models.PixPending, models.PixExpired:
			return b.activatePix(ctx, p)
		case models.PixDisputed:
			return b.restorePix(ctx, p)
		}
	case ProviderExpired, ProviderCancelled:
		_, err := b.movePix(ctx, p, models.PixExpired, models.PixPending)

		return err
	case ProviderUnderDispute:
		return b.disputePix(ctx, p)
	case ProviderRefunded:
		return b.RevokePixPayment(ctx, p, models.RevokedRefund, SourcePix)
	}

	return nil
}

// movePix sets the status of the payment when it is one of from, and reports
// whether a row changed.
func (b *Billing) movePix(ctx context.Context, p models.PixPayment, to string, from ...string) (bool, error) {
	args := make([]any, len(from))
	for i, s := range from {
		args[i] = s
	}

	n, err := orm.QueryOf[models.PixPayment](b.DB).Where("id", p.ID).Where("workspace_id", p.WorkspaceID).
		WhereIn("status", args...).Update(ctx, orm.Set{"status": to})

	return n == 1, err
}

// activatePix marks the payment paid and extends the entitlement, in one
// transaction. The conditional UPDATE is the idempotency key: only the delivery
// that changes the row (RowsAffected == 1) extends the entitlement; every
// other one returns success with no effect. The UPDATE comes first so the
// transaction holds the write lock before it reads the entitlement.
func (b *Billing) activatePix(ctx context.Context, p models.PixPayment) error {
	now := b.Now().UTC()
	activated := false

	err := b.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.PixPayment](tx).Where("id", p.ID).Where("workspace_id", p.WorkspaceID).
			WhereIn("status", models.PixPending, models.PixExpired).
			Update(ctx, orm.Set{"status": models.PixPaid, "paid_at": now})
		if err != nil || n != 1 {
			return err
		}

		activated = true

		if err := b.grantMonths(ctx, tx, p, now); err != nil {
			return err
		}

		return record(ctx, tx, p.WorkspaceID, SourcePix, "pix_paid", p.PublicRef,
			fmt.Sprintf("PIX paid, %d month(s)", p.Months))
	})
	if err != nil || !activated {
		return err
	}

	b.notifyPixPaid(ctx, p)

	return nil
}

// restorePix undoes what the dispute withheld after it was won, on the original
// anchor: time that passed during the dispute is not given again, so repeated
// disputes cannot extend access. A payment whose months were already used when
// disputed gets nothing back.
func (b *Billing) restorePix(ctx context.Context, p models.PixPayment) error {
	return b.DB.Tx(ctx, func(tx *database.Tx) error {
		query := func() orm.Query[models.PixPayment] {
			return orm.QueryOf[models.PixPayment](tx).Where("id", p.ID).Where("workspace_id", p.WorkspaceID)
		}

		n, err := query().Where("status", models.PixDisputed).Update(ctx, orm.Set{"status": models.PixPaid})
		if err != nil || n != 1 {
			return err
		}

		row, err := query().First(ctx)
		if err != nil {
			return err
		}

		summary := "PIX dispute won: no months to restore"

		if row.MonthsWithheld > 0 {
			restored, err := restoreMonths(ctx, tx, row)
			if err != nil {
				return err
			}

			if restored {
				summary = "PIX dispute won: access restored"
			}

			if _, err := query().Update(ctx, orm.Set{"months_withheld": 0}); err != nil {
				return err
			}
		}

		return record(ctx, tx, p.WorkspaceID, SourcePix, "pix_dispute_won", p.PublicRef, summary)
	})
}

// restoreMonths puts the withheld months back in the period they came from. It
// reports false when a newer period has started: the old term is not added to it.
func restoreMonths(ctx context.Context, tx *database.Tx, row models.PixPayment) (bool, error) {
	tctx := workspaces.WithTenant(ctx, row.WorkspaceID)

	ent, err := workspaces.Scoped[models.PixEntitlement](tctx, tx).First(tctx)
	if errors.Is(err, orm.ErrNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if ent.Period != row.EntitlementPeriod {
		return false, nil
	}

	months := ent.PaidMonths + row.MonthsWithheld

	_, err = workspaces.Scoped[models.PixEntitlement](tctx, tx).Where("id", ent.ID).Update(tctx, orm.Set{
		"paid_months": months, "valid_until": timeutil.AddMonths(ent.AnchorAt, months),
		"revoked_at": nil, "revoked_reason": "",
	})

	return err == nil, err
}

// grantMonths adds the months of p and records the period they count in.
func (b *Billing) grantMonths(ctx context.Context, tx *database.Tx, p models.PixPayment, now time.Time) error {
	period, err := b.changeMonths(ctx, tx, p.WorkspaceID, p.Months, now, "")
	if err != nil {
		return err
	}

	_, err = orm.QueryOf[models.PixPayment](tx).Where("id", p.ID).Where("workspace_id", p.WorkspaceID).
		Update(ctx, orm.Set{"entitlement_period": period})

	return err
}

// countsInCurrentPeriod reports whether the months of payment p belong to the
// entitlement's latest period, even if it has lapsed, so paid_months stays the
// sum of that period's paid payments. granting says if that period gives access
// at now.
func countsInCurrentPeriod(
	ctx context.Context, tx *database.Tx, p models.PixPayment, now time.Time,
) (counts, granting bool, err error) {
	tctx := workspaces.WithTenant(ctx, p.WorkspaceID)

	row, err := workspaces.Scoped[models.PixPayment](tctx, tx).Where("id", p.ID).First(tctx)
	if err != nil {
		return false, false, err
	}

	ent, err := workspaces.Scoped[models.PixEntitlement](tctx, tx).First(tctx)
	if errors.Is(err, orm.ErrNotFound) {
		return false, false, nil
	}

	if err != nil {
		return false, false, err
	}

	return row.EntitlementPeriod == ent.Period, ent.Granting(now), nil
}

// disputePix marks a paid payment disputed and takes its months away.
func (b *Billing) disputePix(ctx context.Context, p models.PixPayment) error {
	return b.takeBack(ctx, p, models.PixDisputed, models.RevokedDispute, SourcePix, models.PixPaid)
}

// RevokePixPayment marks the payment refunded and subtracts its months from the
// entitlement, so access ends only when no paid period is left. A payment
// that was disputed already lost its months; one that never became paid has
// none to lose. source is SourcePix (the provider said so) or SourceAdmin.
// Calling it again changes nothing.
func (b *Billing) RevokePixPayment(ctx context.Context, p models.PixPayment, reason, source string) error {
	if err := b.takeBack(ctx, p, models.PixRefunded, reason, source, models.PixPaid); err != nil {
		return err
	}

	// A disputed payment lost its months at the dispute; a payment that was
	// never marked paid here has none to subtract.
	_, err := b.movePix(ctx, p, models.PixRefunded, models.PixDisputed, models.PixPending, models.PixExpired)

	return err
}

// takeBack moves a payment from the status "from" to "to" and subtracts its
// months when they belong to the current period; only the call that changes
// the row subtracts.
func (b *Billing) takeBack(ctx context.Context, p models.PixPayment, to, reason, source, from string) error {
	now := b.Now().UTC()
	subtracted, wasGranting := false, false

	err := b.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.PixPayment](tx).Where("id", p.ID).Where("workspace_id", p.WorkspaceID).
			Where("status", from).Update(ctx, orm.Set{"status": to})
		if err != nil || n != 1 {
			return err
		}

		if subtracted, wasGranting, err = countsInCurrentPeriod(ctx, tx, p, now); err != nil {
			return err
		}

		if subtracted {
			if _, err := b.changeMonths(ctx, tx, p.WorkspaceID, -p.Months, now, reason); err != nil {
				return err
			}

			if to == models.PixDisputed {
				if _, err := orm.QueryOf[models.PixPayment](tx).Where("id", p.ID).Where("workspace_id", p.WorkspaceID).
					Update(ctx, orm.Set{"months_withheld": p.Months}); err != nil {
					return err
				}
			}
		}

		return record(ctx, tx, p.WorkspaceID, source, "pix_"+to, p.PublicRef, "PIX payment "+to+": "+reason)
	})
	if err != nil || !subtracted || !wasGranting {
		return err // access that had already ended needs no "ended" mail
	}

	if ok, err := b.WS.Entitled(ctx, p.WorkspaceID, now); err == nil && !ok {
		b.notifyRevoked(ctx, p.WorkspaceID, reason)
	}

	return nil
}

// changeMonths adds months to the entitlement of the workspace (negative to
// subtract) and recomputes its end from the anchor, so stacked payments never
// drift: valid_until = AddMonths(anchor_at, paid_months).
//
// Adding while the entitlement is still valid grows paid_months; adding after
// it ended or was revoked starts a new period from now. Subtracting to zero
// revokes with reason. It returns the period the entitlement is in.
func (b *Billing) changeMonths(
	ctx context.Context, tx *database.Tx, workspaceID int64, months int, now time.Time, reason string,
) (int, error) {
	tctx := workspaces.WithTenant(ctx, workspaceID)

	ent, err := workspaces.Scoped[models.PixEntitlement](tctx, tx).First(tctx)
	if errors.Is(err, orm.ErrNotFound) {
		if months <= 0 {
			return 0, nil
		}

		return 1, orm.Create(tctx, tx, &models.PixEntitlement{
			AnchorAt: now, PaidMonths: months, ValidUntil: timeutil.AddMonths(now, months), Period: 1,
		})
	}

	if err != nil {
		return 0, err
	}

	set := orm.Set{}

	switch {
	case months > 0 && ent.Granting(now):
		ent.PaidMonths += months
	case months > 0:
		ent.AnchorAt, ent.PaidMonths = now, months
		ent.Period++
		set["anchor_at"], set["period"] = ent.AnchorAt, ent.Period
		set["revoked_at"], set["revoked_reason"] = nil, ""
	default:
		ent.PaidMonths = max(ent.PaidMonths+months, 0)
		if ent.PaidMonths == 0 {
			set["revoked_at"], set["revoked_reason"] = now, reason
		}
	}

	set["paid_months"] = ent.PaidMonths
	set["valid_until"] = timeutil.AddMonths(ent.AnchorAt, ent.PaidMonths)

	_, err = workspaces.Scoped[models.PixEntitlement](tctx, tx).Where("id", ent.ID).Update(tctx, set)

	return ent.Period, err
}
