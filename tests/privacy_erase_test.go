package tests

import (
	"net/url"
	"testing"
)

func eraseForm(word, password string) url.Values {
	return url.Values{"confirmation": {word}, "password": {password}}
}

func TestPersonalEraseOverTheWeb(t *testing.T) {
	t.Parallel()

	app := newApp(t, map[string]string{"APP_EDITION": "personal"})
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))

	vera := viewerOf(t, app, ana)
	veraBrowser := browser(t, app, vera.User, ana.Workspace.ID)

	// A viewer cannot erase the installation.
	veraBrowser.PostForm("/account/erase", eraseForm("ERASE", knownPassword)).AssertForbidden()

	// The owner needs the password and the exact word.
	ana.Client.PostForm("/account/erase", eraseForm("ERASE", "wrong")).AssertSessionHasErrors("password")
	ana.Client.PostForm("/account/erase", eraseForm("erase", knownPassword)).AssertSessionHasErrors("confirmation")

	ana.Client.PostForm("/account/erase", eraseForm("ERASE", knownPassword)).AssertRedirect("/account/privacy?erased=1")

	ana.Client.Inertia("").Get("/account/privacy").AssertOK().AssertInertia("Privacy").
		PropEquals("canErase", true).PropEquals("personal", true).PropEquals("erased", false)
}

func TestSaasHasNoEraseRoute(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))

	ana.Client.PostForm("/account/erase", eraseForm("ERASE", knownPassword)).AssertNotFound()
	ana.Client.Inertia("").Get("/account/privacy").AssertOK().AssertInertia("Privacy").PropEquals("canErase", false)
}
