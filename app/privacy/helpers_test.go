package privacy_test

import (
	"maps"
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func newApp(t *testing.T, env ...map[string]string) *vukemtest.App {
	t.Helper()

	opts := vukemtest.Options{Env: map[string]string{"APP_EDITION": "saas"}}
	for _, e := range env {
		maps.Copy(opts.Env, e)
	}

	return vukemtest.New(t, bootstrap.New, opts)
}

func newUser(t *testing.T, app *vukemtest.App, name, email string) models.User {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: name, Email: email, Password: noLogin}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	return u
}

func workspaceSvc(t *testing.T, app *vukemtest.App) *workspaces.Service {
	t.Helper()

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	return ws
}

// noLogin is a stored value no hasher accepts, for users who never sign in.
const noLogin = "!"

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func newService(t *testing.T, app *vukemtest.App) *privacy.Service {
	t.Helper()

	svc, err := privacy.New(app.App, workspaceSvc(t, app))
	if err != nil {
		t.Fatal(err)
	}

	return svc
}
