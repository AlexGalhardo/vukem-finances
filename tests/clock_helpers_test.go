package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}

	return d.Add(12 * time.Hour)
}

// clockApp is a saas app whose clock the test moves.
func clockApp(t *testing.T, start string) (*vukemtest.App, *vukemtest.FakeClock) {
	t.Helper()

	clock := vukemtest.NewFakeClock(day(start))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})

	return app, clock
}

func mustDB(t *testing.T, app *vukemtest.App) *database.DB {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	return db
}
