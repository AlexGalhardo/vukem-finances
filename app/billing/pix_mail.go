package billing

import (
	"context"
	"strconv"

	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// notifyPixPaid emails the owners a receipt that says until when Pro lasts.
// A failed email is logged by workspace only and never fails the activation.
func (b *Billing) notifyPixPaid(ctx context.Context, p models.PixPayment) {
	tctx := workspaces.WithTenant(ctx, p.WorkspaceID)

	ent, err := workspaces.Scoped[models.PixEntitlement](tctx, b.DB).First(tctx)
	if err != nil {
		b.Log.WarnContext(ctx, "billing: entitlement not found for a receipt", "workspace", p.WorkspaceID)
		return
	}

	ws, err := orm.QueryOf[models.Workspace](b.DB).Find(ctx, p.WorkspaceID)
	if err != nil {
		b.Log.WarnContext(ctx, "billing: workspace not found for a receipt", "workspace", p.WorkspaceID)
		return
	}

	owners, err := b.WS.OwnerEmails(ctx, p.WorkspaceID)
	if err != nil {
		b.Log.WarnContext(ctx, "billing: owners not found", "workspace", p.WorkspaceID)
		return
	}

	for _, to := range owners {
		var m mail.Mailable = appmail.PixReceipt{
			To: to, Workspace: ws.Name, Amount: finance.FormatAmount(p.AmountCents, "BRL"),
			Months: strconv.Itoa(p.Months), ValidUntil: ent.ValidUntil.Format("January 2, 2006"), URL: b.URL + "/billing",
		}
		if err := b.WS.Notify(ctx, m); err != nil {
			b.Log.WarnContext(ctx, "billing: email not sent", "workspace", p.WorkspaceID)
		}
	}
}
