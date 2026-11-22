// Package bootstrap wires the application together. There is no hidden
// registration: everything the app uses is listed here.
package bootstrap

import (
	"reflect"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/listeners"
	"github.com/AlexGalhardo/vukem-finances/app/offline"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/config"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
	"github.com/AlexGalhardo/vukem-finances/database/seeders"
	"github.com/AlexGalhardo/vukem-finances/resources"
	"github.com/AlexGalhardo/vukem-finances/routes"
)

// New builds the application; cmd/app and the tests call it.
func New(opts ...vukem.Option) (*vukem.App, error) {
	// The one place the offline rules of the personal edition are applied (D-027).
	gate := offline.NewGate()

	base := []vukem.Option{
		vukem.WithName("vukem-finances"),
		// Sessions are rows of the app database, not encrypted cookies: logging out ends that
		// session for every copy of its cookie, and "sign out of the other devices" has rows
		// to delete (QA-001). The framework creates the table and prunes it hourly. An
		// explicit SESSION_DRIVER still wins; no network is involved in either edition.
		vukem.WithEnvDefaults(map[string]string{"SESSION_DRIVER": "database"}),
		vukem.WithProviders(append([]vukem.Provider{gate.Provider()}, Providers()...)...),
		vukem.WithViewFuncs(gate.ViewFuncs()),
		vukem.WithMigrations(migrations.All()),
		vukem.WithSeeders(seeders.All()...),
		vukem.WithAppSeeders(seeders.AllApp()...),
		vukem.WithBench(Bench()...),
		vukem.WithListeners(listeners.All()...),
		vukem.WithJobsFunc(jobs.All),
		vukem.WithSchedule(routes.Schedule),
		vukem.WithRoutesAfterBoot(checkTriggers),
		vukem.WithViews(resources.Views()),
		vukem.WithVite(resources.Public()),
		vukem.WithReactEmails(resources.Emails()),
		vukem.WithInertia(inertia.Config{Pages: pages.All(), SharedType: reflect.TypeFor[pages.Shared]()}),
	}

	// First, before any route reads the edition: a typo in APP_EDITION must
	// stop every command and the server, not boot the wrong edition.
	base = append(base, vukem.WithRoutes(checkEdition), vukem.WithRoutes(checkRetention))

	for _, fn := range routes.All() {
		base = append(base, vukem.WithRoutes(fn))
	}

	app, err := vukem.New(append(base, opts...)...)
	if err != nil {
		return nil, err
	}

	app.AddCommand(Commands()...)

	return app, nil
}

// checkRetention stops the boot on a bad TOMBSTONE_RETENTION_DAYS, instead of
// failing the nightly job a day later.
func checkRetention(_ *router.Router, app *vukem.App) error {
	_, err := config.TombstoneRetention(app.Source)

	return err
}

func checkEdition(_ *router.Router, app *vukem.App) error {
	_, err := config.LoadEdition(app.Source)

	return err
}
