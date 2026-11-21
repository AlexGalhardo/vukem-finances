package ledger

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
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
