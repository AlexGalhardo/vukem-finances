package ledger

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// BillingHandlers serve the billing page, Stripe Checkout and the Billing
// Portal of the current workspace. Only its owner can start or manage a
// subscription; every member sees the plan.
type BillingHandlers struct {
	h *Handlers
	b *billing.Billing
}

// NewBilling returns the billing handlers.
func NewBilling(h *Handlers, b *billing.Billing) *BillingHandlers {
	return &BillingHandlers{h: h, b: b}
}

// Show renders the plan, the limits and the subscription.
func (bh *BillingHandlers) Show(c *web.Ctx) error {
	ws, err := bh.h.WS.Current(c.Context())
	if err != nil {
		return err
	}

	lim, err := bh.h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	props := pages.BillingProps{
		Limits: lim, TrialDays: billing.TrialDays, CanManage: bh.h.can(c, workspaces.AbilityManageBilling) == nil,
		Configured: bh.b.Stripe != nil, Currencies: bh.b.Config.Currencies, Checkout: c.Query("checkout") == "success",
		Pix: bh.pixOffer(),
	}

	if props.PixAccess, err = bh.pixAccess(c); err != nil {
		return err
	}

	if sub, ok, err := bh.b.Subscription(c.Context(), ws.ID); err != nil {
		return err
	} else if ok {
		props.Subscription = &pages.BillingStatus{
			Status: sub.Status, Valid: sub.Valid(), CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		}
		if sub.AccessRevokedAt.Valid {
			props.Subscription.RevokedReason = sub.RevokedReason
		}
		if sub.CurrentPeriodEnd.Valid {
			props.Subscription.CurrentPeriodEnd = sub.CurrentPeriodEnd.V.Format("2006-01-02")
		}
	}

	return pages.Billing.Render(c, props)
}

// pixOffer lists the months on sale with the server's totals; nil without PIX.
func (bh *BillingHandlers) pixOffer() *pages.PixOffer {
	if bh.b.Abacate == nil {
		return nil
	}

	offer := &pages.PixOffer{Currency: "BRL", Options: []pages.PixOption{}}

	for _, months := range bh.b.Config.Pix.Months {
		if cents, ok := bh.b.Config.Pix.Amount(months); ok {
			offer.Options = append(offer.Options, pages.PixOption{Months: months, TotalCents: cents})
		}
	}

	return offer
}

// pixAccess is the PIX period of the current workspace, or nil.
func (bh *BillingHandlers) pixAccess(c *web.Ctx) (*pages.PixAccess, error) {
	ent, err := workspaces.Scoped[models.PixEntitlement](c.Context(), bh.h.WS.DB).First(c.Context())
	if errors.Is(err, orm.ErrNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	view := &pages.PixAccess{
		Active: ent.Granting(bh.b.Now().UTC()), ValidUntil: ent.ValidUntil.UTC().Format("2006-01-02"),
	}
	if ent.RevokedAt.Valid {
		view.RevokedReason = ent.RevokedReason
	}

	return view, nil
}

// PixStart creates a PIX charge for the "months" form field and answers the
// charge as JSON for the dialog: the QR image, the copy-and-paste code and the
// timer. The amount is the server's price for those months, never a request value.
func (bh *BillingHandlers) PixStart(c *web.Ctx) error {
	if err := bh.h.can(c, workspaces.AbilityManageBilling); err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	ws, err := bh.h.WS.Current(c.Context())
	if err != nil {
		return err
	}

	months, err := strconv.Atoi(c.FormValue("months"))
	if err != nil {
		return errs.New(http.StatusUnprocessableEntity, "Choose one of the offered periods.")
	}

	p, err := bh.b.StartPix(c.Context(), ws, u.ID, months)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, pixView(p, true, ""))
}

// PixShow answers the state of a charge of the current workspace for the
// dialog's poll: it re-reads a pending charge from the provider. Another
// workspace's reference is a 404.
func (bh *BillingHandlers) PixShow(c *web.Ctx) error {
	if err := bh.h.can(c, workspaces.AbilityManageBilling); err != nil {
		return err
	}

	p, err := bh.b.PixPayment(c.Context(), c.Param("ref"))
	if err != nil {
		return err
	}

	if p, err = bh.b.RefreshPix(c.Context(), p); err != nil {
		return err
	}

	lim, err := bh.h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, pixView(p, false, lim.Plan))
}

func pixView(p models.PixPayment, withCode bool, plan string) pages.PixView {
	v := pages.PixView{
		Ref: p.PublicRef, Status: p.Status, Months: p.Months, AmountCents: p.AmountCents,
		ExpiresAt: p.ExpiresAt.UTC().Format(time.RFC3339), Plan: plan,
	}
	if withCode {
		v.BRCode, v.QR = p.BRCode, p.QRPNGBase64
	}

	return v
}

// Checkout sends the owner to Stripe Checkout for the plan in the URL, in
// the optional "currency" form field.
func (bh *BillingHandlers) Checkout(c *web.Ctx) error {
	if err := bh.h.can(c, workspaces.AbilityManageBilling); err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	ws, err := bh.h.WS.Current(c.Context())
	if err != nil {
		return err
	}

	target, err := bh.b.Checkout(c.Context(), ws, u.Email, c.Param("plan"), strings.ToLower(c.FormValue("currency")))
	if err != nil {
		return back(c, "/billing", err)
	}

	return away(c, target)
}

// Portal sends the owner to the Stripe Billing Portal.
func (bh *BillingHandlers) Portal(c *web.Ctx) error {
	if err := bh.h.can(c, workspaces.AbilityManageBilling); err != nil {
		return err
	}

	ws, err := bh.h.WS.Current(c.Context())
	if err != nil {
		return err
	}

	target, err := bh.b.Portal(c.Context(), ws)
	if err != nil {
		return back(c, "/billing", err)
	}

	return away(c, target)
}

// away leaves the app for a Stripe page: a full page visit for Inertia,
// the URL for JSON clients, a 303 redirect for forms.
func away(c *web.Ctx, target string) error {
	switch {
	case inertia.IsInertia(c.Request()):
		return inertia.Location(c, target)
	case c.WantsJSON():
		return c.JSON(http.StatusOK, map[string]string{"url": target})
	}

	return c.RedirectStatus(http.StatusSeeOther, target)
}

// Revenue is the admin overview of the business: MRR and subscriptions.
// It reads across workspaces on purpose, behind the admin panel's gate.
func Revenue(h *Handlers) func(c *web.Ctx) error {
	return func(c *web.Ctx) error {
		ctx := c.Context()
		props := pages.AdminRevenueProps{}

		subs, err := orm.QueryOf[models.WorkspaceSubscription](h.WS.DB).Get(ctx)
		if err != nil {
			return err
		}

		for i := range subs {
			switch subs[i].Status {
			case "active":
				props.Active++
				props.MRRCents += subs[i].UnitAmount
			case "trialing":
				props.Trialing++
			case "past_due":
				props.PastDue++
			}
		}

		if props.Workspaces, err = orm.QueryOf[models.Workspace](h.WS.DB).Count(ctx); err != nil {
			return err
		}

		if props.Users, err = orm.QueryOf[models.User](h.WS.DB).Count(ctx); err != nil {
			return err
		}

		return pages.AdminRevenue.Render(c, props)
	}
}
