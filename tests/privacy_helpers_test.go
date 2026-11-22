package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

const knownPassword = "correct horse battery staple"

// withPassword gives w.User a password the hasher accepts and returns the user
// as the browser should see it (the session loads the user again by id).
func withPassword(t *testing.T, app *vukemtest.App, w who) who {
	t.Helper()

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	h, err := acc.Hasher.Make(knownPassword)
	if err != nil {
		t.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := orm.QueryOf[models.User](db).Where("id", w.User.ID).Update(t.Context(), orm.Set{"password": h}); err != nil {
		t.Fatal(err)
	}

	w.User.Password = h
	w.Client = browser(t, app, w.User, w.Workspace.ID)

	return w
}

func passwordForm(pw string) url.Values { return url.Values{"password": {pw}} }
