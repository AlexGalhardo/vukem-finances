package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/config"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// checkTriggers stops the boot when a synced table lost its change-log
// triggers (a table rebuild drops them). Tests run the same check in
// app/changes. It waits for the migration that installs them, so `migrate`
// itself can boot a database that is still behind.
func checkTriggers(_ *router.Router, app *vukem.App) error {
	if app.Source.Env() == config.Test {
		return nil
	}

	ctx := context.Background()

	db, err := app.DB()
	if err != nil {
		return err
	}

	var one int

	err = db.SQL().QueryRowContext(ctx, "SELECT 1 FROM migrations WHERE id = ?", migrations.ChangeTriggersID).Scan(&one)
	if err != nil {
		return nil //nolint:nilerr // no migrations table or migration yet: nothing to verify
	}

	return changes.VerifyTriggers(ctx, db)
}
