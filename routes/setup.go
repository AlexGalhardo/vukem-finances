package routes

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/controllers"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Setup registers /setup, the form that creates the first owner of a
// personal installation. It answers 404 once a user exists, and needs the
// setup secret (SETUP_TOKEN, or the code printed in the boot log).
func Setup(r *router.Router, app *vukem.App) error {
	edition, err := config.LoadEdition(app.Source)
	if err != nil {
		return err
	}

	if !edition.Personal() {
		return nil
	}

	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	ws, err := workspaces.New(app)
	if err != nil {
		return err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	secret := accounts.NewSetupSecret(app.Source.Get("SETUP_TOKEN"))
	announceSetup(app, ws, secret)

	h := controllers.NewSetup(acc, ws, secret)
	signupIP := ratelimit.Middleware(limiters("register", 10, 10*time.Minute), ratelimit.ByIP)
	// Five tries a minute per address: the code cannot be guessed online.
	setupIP := ratelimit.Middleware(limiters("setup", 5, time.Minute), ratelimit.ByIP)

	r.Get("/setup", h.Show).Name("setup.show")
	r.Post("/setup", h.Store).Name("setup.store").Use(signupIP, setupIP)

	return nil
}

// announceSetup prints the generated setup code once, in the boot log of the
// server, while there is still no user to set up.
func announceSetup(app *vukem.App, ws *workspaces.Service, secret *accounts.SetupSecret) {
	if app.Kind != vukem.KindWeb {
		return
	}

	// An unreadable table (not migrated yet) counts as "no user yet".
	if open, err := ws.SetupOpen(context.Background()); err == nil && !open {
		return
	}

	switch {
	case secret.Code != "":
		app.Log.Info("setup code: " + secret.Code)

		if app.Config.IsProduction() {
			app.Log.Warn("SETUP_TOKEN is not set: /setup needs the one-time code above, printed again at every restart")
		}
	default:
		app.Log.Info("setup: /setup needs SETUP_TOKEN")
	}
}
