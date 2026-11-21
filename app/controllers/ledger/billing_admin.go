package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
)

// AdminPixRefund marks a PIX payment as refunded (support action, behind the
// admin gate): the payment is revoked like a provider refund and an audit row
// names the admin. Doing it twice changes nothing.
func AdminPixRefund(b *billing.Billing) func(c *web.Ctx) error {
	return func(c *web.Ctx) error {
		u, err := auth.User[models.User](c, "web")
		if err != nil {
			return err
		}

		if err := b.AdminMarkRefunded(c.Context(), c.Param("ref"), u.ID); err != nil {
			return back(c, "/admin/revenue", err)
		}

		return c.Redirect("/admin/revenue")
	}
}

// pixRevenue adds the PIX side to the revenue overview: the monthly equivalent
// of every entitlement that is valid and not revoked (the price of its latest
// paid payment divided by its months, in whole cents), and the latest payments
// with the refund action. It reads across workspaces on purpose, behind the
// admin gate.
func pixRevenue(ctx context.Context, h *Handlers, props *pages.AdminRevenueProps) error {
	now := time.Now().UTC()

	ents, err := orm.QueryOf[models.PixEntitlement](h.WS.DB).WhereNull("revoked_at").
		Where("valid_until", ">", now).Get(ctx)
	if err != nil {
		return err
	}

	for _, e := range ents {
		last, err := orm.QueryOf[models.PixPayment](h.WS.DB).Where("workspace_id", e.WorkspaceID).
			Where("status", models.PixPaid).OrderBy("id", orm.Desc).First(ctx)
		if errors.Is(err, orm.ErrNotFound) {
			continue
		}

		if err != nil {
			return err
		}

		props.PixActive++
		props.PixMRRCents += last.AmountCents / int64(max(last.Months, 1))
	}

	recent, err := orm.QueryOf[models.PixPayment](h.WS.DB).OrderBy("id", orm.Desc).Limit(20).Get(ctx)
	if err != nil {
		return err
	}

	props.RecentPix = make([]pages.AdminPixPayment, 0, len(recent))

	for _, p := range recent {
		props.RecentPix = append(props.RecentPix, pages.AdminPixPayment{
			Ref: p.PublicRef, Workspace: p.WorkspaceID, Status: p.Status, AmountCents: p.AmountCents, Months: p.Months,
			CanRefund: p.Status == models.PixPaid || p.Status == models.PixDisputed,
		})
	}

	return nil
}
