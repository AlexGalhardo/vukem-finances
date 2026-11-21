package routes

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
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
	"github.com/AlexGalhardo/vukem-finances/config"
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

	// RATE_LIMIT_STORE=cache shares the limits between servers.
	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	user := ratelimit.ByUser("web")
	reads := ratelimit.Middleware(limiters("workspace.read", config.ReadLimit, config.ReadWindow), onMethods(user, true))
	writes := ratelimit.Middleware(limiters("workspace.write", config.WriteLimit, config.WriteWindow), onMethods(user, false))
	imports := ratelimit.Middleware(limiters("workspace.import", config.ImportLimit, config.ImportWindow), byWorkspace)
	invites := ratelimit.Middleware(limiters("workspace.invite", config.InviteLimit, config.InviteWindow), byWorkspace)

	// Everything else lives in a workspace: the tenant middleware picks the
	// current one from the session and answers 404 for one the user is not in.
	// The limits come after it: they need the signed-in user and the workspace.
	r.Group("", func(g *router.Group) {
		g.Use(required, verified, ws.Middleware(), ws.Require(workspaces.AbilityRead), reads, writes,
			shareWorkspace(ws, acc, admins))

		financeDashboard(g, h)
		financeTransactions(g, h)
		financeAccounts(g, h)
		financeCategories(g, h)
		financeBudgets(g, h)
		financeImports(g, h, imports)
		financeWorkspace(g, h)

		// One implicit workspace in the personal edition: nothing to create or switch to.
		if ws.Edition.SaaS() {
			g.Post("/workspaces", h.CreateWorkspace).Name("workspaces.store")
			g.Post("/workspaces/{workspace}/switch", h.SwitchWorkspace).Name("workspaces.switch")
		}

		g.Post("/workspace/invites", h.Invite).Name("workspace.invite").Use(invites)
		g.Delete("/workspace/members/{user}", h.RemoveMember).Name("workspace.members.destroy")
		g.Patch("/workspace/members/{user}", h.ChangeRole).Name("workspace.members.role")
		g.Post("/workspace/owner", h.TransferOwnership).Name("workspace.transfer")
		g.Post("/workspace/leave", h.Leave).Name("workspace.leave")

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
	} else {
		// Without this, the panel's generic /admin/{slug} answers 200 with an empty shell.
		r.Get("/admin/revenue", func(*web.Ctx) error { return errs.ErrNotFound }).Name("admin.revenue")
	}

	return nil
}

// onMethods applies key to reads (GET, HEAD) when reads is true, else to
// every other method; the other requests pass this limiter.
func onMethods(key ratelimit.KeyFunc, reads bool) ratelimit.KeyFunc {
	return func(r *http.Request) string {
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) != reads {
			return ""
		}

		return key(r)
	}
}

// byWorkspace keys a limit by the workspace of the request, so the members of
// one workspace share it; the tenant middleware has already put it there.
func byWorkspace(r *http.Request) string {
	id, ok := workspaces.TenantID(r.Context())
	if !ok {
		return ratelimit.ByIP(r)
	}

	return "workspace:" + strconv.FormatInt(id, 10)
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
					all[i] = pages.WorkspaceView{ID: m.ID, Name: m.Name, Currency: m.Currency, Timezone: m.Timezone}
				}

				return pages.SharedWorkspace{
					Current: pages.WorkspaceView{ID: cur.ID, Name: cur.Name, Currency: cur.Currency, Timezone: cur.Timezone},
					All:     all, Role: workspaces.Role(ctx), Plan: lim.Plan, Admin: admins.Has(&u),
					Edition: ws.Edition.String(),
				}, nil
			})

			next.ServeHTTP(w, r)
		})
	}
}
