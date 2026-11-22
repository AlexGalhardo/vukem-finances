package controllers

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Root is "/" in both editions. The app has no landing page: a signed-in visitor goes to the
// dashboard, a guest to the login, and in the personal edition to the setup while the
// installation has no user yet. It never leaves this server, so the personal edition
// works offline.
func Root(acc *accounts.Accounts, ws *workspaces.Service, edition config.Edition) func(*web.Ctx) error {
	return func(c *web.Ctx) error {
		if _, err := acc.Web.User(c); err == nil {
			return c.Redirect("/dashboard")
		}

		if edition.Personal() {
			open, err := ws.SetupOpen(c.Context())
			if err != nil {
				return err
			}

			if open {
				return c.Redirect("/setup")
			}
		}

		return c.Redirect("/login")
	}
}
