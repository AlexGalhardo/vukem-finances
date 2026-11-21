package controllers

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Home renders the Home page component (resources/js/pages/Home).
func Home(c *web.Ctx) error { return pages.Home.Render(c, struct{}{}) }

// PersonalHome is "/" in the personal edition, which has no landing page: it
// sends the visitor to the dashboard, to the login, or to the setup while the
// installation has no user yet.
func PersonalHome(acc *accounts.Accounts, ws *workspaces.Service) func(*web.Ctx) error {
	return func(c *web.Ctx) error {
		if _, err := acc.Web.User(c); err == nil {
			return c.Redirect("/dashboard")
		}

		open, err := ws.SetupOpen(c.Context())
		if err != nil {
			return err
		}

		if open {
			return c.Redirect("/setup")
		}

		return c.Redirect("/login")
	}
}
