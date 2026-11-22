package tests

import (
	"strings"
	"testing"
)

func TestPersonalExportNeedsThePassword(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))

	// No confirmation: the download goes back to the privacy page.
	ana.Client.Get("/account/export.json").AssertRedirect("/account/privacy")

	// A wrong password confirms nothing.
	ana.Client.PostForm("/account/export", passwordForm("wrong")).AssertStatus(303).AssertSessionHasErrors("password")
	ana.Client.Get("/account/export.json").AssertRedirect("/account/privacy")

	ana.Client.PostForm("/account/export", passwordForm(knownPassword)).AssertRedirect("/account/privacy?export=ready")

	res := ana.Client.Get("/account/export.json").AssertOK().
		AssertHeaderContains("Content-Disposition", "attachment").AssertHeader("Cache-Control", "no-store")

	body := res.Body()
	if !strings.Contains(body, "ana@example.com") || strings.Contains(body, ana.User.Password) {
		t.Errorf("the export must hold the profile and never the password hash: %s", body)
	}

	// The confirmation is spent by the download.
	ana.Client.Get("/account/export.json").AssertRedirect("/account/privacy")
}
