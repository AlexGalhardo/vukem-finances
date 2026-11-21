package ai_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// testKey is a made-up key: no test needs, or may use, a real one.
const testKey = "sk-placeholder-0123456789abcdefghij-WXYZ"

type site struct {
	app *vukemtest.App
	ws  models.Workspace
	// ctx acts for ws.
	ctx  context.Context
	user models.User
}

func newApp(t *testing.T, env map[string]string) *vukemtest.App {
	t.Helper()

	if env == nil {
		env = map[string]string{}
	}

	if env["APP_EDITION"] == "" {
		env["APP_EDITION"] = "saas"
	}

	return vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: env})
}

// newSite makes a user with a workspace in app.
func newSite(t *testing.T, app *vukemtest.App, email string) site {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: "Ana", Email: email, Password: strings.Repeat("h", 20)}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	svc, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	ws, err := svc.Create(t.Context(), u, "Ana household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	return site{app: app, ws: ws, ctx: workspaces.WithTenant(t.Context(), ws.ID), user: u}
}
