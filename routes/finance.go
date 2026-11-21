package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/events"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/admin"
	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/controllers"
	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Finance registers the personal finance app: the dashboard, transactions,
// accounts, categories, budgets, CSV import, workspace settings, billing
// and the Stripe and Resend webhooks.
func Finance(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	ws, err := workspaces.New(app)
	if err != nil {
		return err
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return err
	}

	// The personal edition has no billing: no Stripe settings are read, and
	// the billing pages and the Stripe webhook are not registered (404).
	var b *billing.Billing

	if ws.Edition.SaaS() {
		if b, err = billing.New(app, ws); err != nil {
			return err
		}

		// The webhooks run before the session and CSRF middleware: Stripe and
		// Resend authenticate with a signature, not with cookies.
		r.UseServer(b.Webhook())
		events.Listen(app.Events, b.HandleWebhook)
	}

	resend, err := appmail.ResendWebhook(app.Source.Get("RESEND_WEBHOOK_SECRET"), ws.DB, app.Log)
	if err != nil {
		return err
	}

	r.UseServer(resend)

	h := ledger.New(ws, fin)
	admins := admin.NewAdmins(app.Source.Get("ADMIN_EMAILS"))
	required := auth.Required(acc.Web)
	verified := auth.Verified(acc.Web)

	// Invitations: any verified user, no workspace yet. In the personal edition
	// the link is also how a person with no account registers: the token is a
	// path segment, never a query string, so it stays out of access logs.
	if ws.Edition.Personal() {
		limiters, err := app.RateLimiters()
		if err != nil {
			return err
		}

		ih := controllers.NewInvite(acc, ws)
		signupIP := ratelimit.Middleware(limiters("register", 10, 10*time.Minute), ratelimit.ByIP)

		r.Get("/invites/{token}", ih.Entry).Name("invites.show").SecretParams("token")
		r.Get("/invites/{token}/register", ih.ShowRegister).Name("invites.register.show").
			Use(auth.Guest(acc.Web)).SecretParams("token")
		r.Post("/invites/{token}/register", ih.Register).Name("invites.register").
			Use(auth.Guest(acc.Web), signupIP).SecretParams("token")
	} else {
		r.Get("/invites/{token}", h.ShowInvite).Name("invites.show").Use(required, verified).SecretParams("token")
	}

	r.Post("/invites/{token}", h.AcceptInvite).Name("invites.accept").Use(required, verified).SecretParams("token")

	// Everything else lives in a workspace: the tenant middleware picks the
	// current one from the session and answers 404 for one the user is not in.
	r.Group("", func(g *router.Group) {
		g.Use(required, verified, ws.Middleware(), shareWorkspace(ws, acc, admins))

		g.Get("/dashboard", h.Dashboard).Name("dashboard")
		g.Get("/transactions", h.Transactions).Name("transactions.index")
		g.Post("/transactions", h.StoreTransaction).Name("transactions.store")
		g.Patch("/transactions/{transaction}", h.UpdateTransaction).Name("transactions.update")
		g.Delete("/transactions/{transaction}", h.DestroyTransaction).Name("transactions.destroy")
		g.Get("/accounts", h.Accounts).Name("accounts.index")
		g.Post("/accounts", h.StoreAccount).Name("accounts.store")
		g.Delete("/accounts/{account}", h.DestroyAccount).Name("accounts.destroy")
		g.Get("/categories", h.Categories).Name("categories.index")
		g.Post("/categories", h.StoreCategory).Name("categories.store")
		g.Get("/budgets", h.Budgets).Name("budgets.index")
		g.Post("/budgets", h.StoreBudget).Name("budgets.store")
		g.Delete("/budgets/{budget}", h.DestroyBudget).Name("budgets.destroy")
		g.Get("/import", h.Import).Name("import.index")
		g.Post("/import", h.UploadImport).Name("import.upload")
		g.Post("/import/{import}", h.ConfirmImport).Name("import.confirm")
		g.Get("/workspace", h.Workspace).Name("workspace.show")
		g.Put("/workspace", h.UpdateWorkspace).Name("workspace.update")
		g.Post("/workspaces", h.CreateWorkspace).Name("workspaces.store")
		g.Post("/workspaces/{workspace}/switch", h.SwitchWorkspace).Name("workspaces.switch")
		g.Post("/workspace/invites", h.Invite).Name("workspace.invite")
		g.Delete("/workspace/members/{user}", h.RemoveMember).Name("workspace.members.destroy")

		if b != nil {
			bh := ledger.NewBilling(h, b)
			g.Get("/billing", bh.Show).Name("billing.show")
			g.Post("/billing/checkout/{plan}", bh.Checkout).Name("billing.checkout")
			g.Post("/billing/portal", bh.Portal).Name("billing.portal")
		}
	})

	if ws.Edition.SaaS() {
		// The business overview of the admins (ADMIN_EMAILS): across workspaces.
		r.Get("/admin/revenue", ledger.Revenue(h)).Name("admin.revenue").Use(required, verified, onlyAdmins(admins))
	}

	return nil
}

func onlyAdmins(admins admin.Admins) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, err := auth.UserFrom[models.User](r.Context(), "web")
			if err != nil || !admins.Has(&u) {
				http.NotFound(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shareWorkspace shares the current workspace, the user's other ones, their
// role and the plan with every page (usePage().props.workspace).
func shareWorkspace(
	ws *workspaces.Service, acc *accounts.Accounts, admins admin.Admins,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			inertia.Share(ctx, "workspace", func(context.Context) (any, error) {
				c := web.NewCtx(w, r, web.ServicesFrom(ctx))

				u, err := acc.Web.User(c)
				if err != nil {
					return nil, nil //nolint:nilerr // signed out
				}

				cur, err := ws.Current(ctx)
				if err != nil {
					return nil, err
				}

				mine, err := ws.Mine(ctx, u)
				if err != nil {
					return nil, err
				}

				lim, err := ws.Limits(ctx)
				if err != nil {
					return nil, err
				}

				all := make([]pages.WorkspaceView, len(mine))
				for i, m := range mine {
					all[i] = pages.WorkspaceView{ID: m.ID, Name: m.Name, Currency: m.Currency}
				}

				return pages.SharedWorkspace{
					Current: pages.WorkspaceView{ID: cur.ID, Name: cur.Name, Currency: cur.Currency},
					All:     all, Role: workspaces.Role(ctx), Plan: lim.Plan, Admin: admins.Has(&u),
				}, nil
			})

			next.ServeHTTP(w, r)
		})
	}
}
