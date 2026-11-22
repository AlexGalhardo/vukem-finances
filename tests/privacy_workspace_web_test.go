package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func deletionForm(name, password string) url.Values {
	return url.Values{"name": {name}, "password": {password}}
}

func TestWorkspaceDeletionOverTheWeb(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))

	bob := withPassword(t, app, signUp(t, app, "Bob", "bob@example.com"))
	join(t, app, ana, bob.User, models.RoleAdmin)

	adminBrowser := browser(t, app, bob.User, ana.Workspace.ID)

	// An admin is not the owner: 403 before the password is even looked at.
	adminBrowser.PostForm("/workspace/deletion", deletionForm(ana.Workspace.Name, knownPassword)).AssertForbidden()

	// The owner with the wrong password, then with the wrong name.
	ana.Client.PostForm("/workspace/deletion", deletionForm(ana.Workspace.Name, "wrong")).AssertSessionHasErrors("password")
	ana.Client.PostForm("/workspace/deletion", deletionForm("not the name", knownPassword)).AssertStatus(303)

	if workspaceRow(t, app, ana).PendingDeletion() {
		t.Fatal("a failed confirmation scheduled the deletion")
	}

	ana.Client.PostForm("/workspace/deletion", deletionForm(ana.Workspace.Name, knownPassword)).
		AssertRedirect("/account/privacy")

	if !workspaceRow(t, app, ana).PendingDeletion() {
		t.Fatal("the workspace is not pending")
	}

	adminBrowser.PostForm("/workspace/deletion/cancel", url.Values{}).AssertForbidden()
	ana.Client.PostForm("/workspace/deletion/cancel", url.Values{}).AssertRedirect("/account/privacy")

	if workspaceRow(t, app, ana).PendingDeletion() {
		t.Fatal("the owner could not cancel")
	}
}
