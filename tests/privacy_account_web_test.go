package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func accountDeletionForm(confirmation, password string) url.Values {
	return url.Values{"confirmation": {confirmation}, "password": {password}}
}

func TestAccountDeletionOverTheWeb(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))
	token := rawToken(t, app, ana.User, ana.Workspace.ID, allScopes()...)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	pending := func() bool {
		u, err := orm.QueryOf[models.User](db).Find(t.Context(), ana.User.ID)
		if err != nil {
			t.Fatal(err)
		}

		return u.PendingDeletion()
	}

	// A wrong password, then a wrong confirmation: nothing happens.
	ana.Client.PostForm("/account/deletion", accountDeletionForm("ana@example.com", "wrong")).AssertSessionHasErrors("password")
	ana.Client.PostForm("/account/deletion", accountDeletionForm("someone@else.com", knownPassword)).AssertSessionHasErrors("confirmation")

	if pending() {
		t.Fatal("a failed confirmation scheduled the deletion")
	}

	apiClient(t, app, token).Get("/api/v1/accounts").AssertOK()

	ana.Client.PostForm("/account/deletion", accountDeletionForm("ANA@example.com", knownPassword)).AssertRedirect("/login")

	if !pending() {
		t.Fatal("the account is not pending")
	}

	// Every credential stopped at once: the API token and the browser session.
	apiClient(t, app, token).Get("/api/v1/accounts").AssertUnauthorized()
	ana.Client.Get("/dashboard").AssertRedirect("/login")

	// Signing in during the grace period leads to the offer to cancel, and nothing else opens.
	login := app.Client(t).WithHeader("Sec-Fetch-Site", "same-origin")
	login.PostForm("/login", url.Values{"email": {"ana@example.com"}, "password": {knownPassword}}).AssertRedirect("/dashboard")
	login.Get("/dashboard").AssertRedirect("/account/deletion")
	login.Get("/transactions").AssertRedirect("/account/deletion")

	login.Get("/account/deletion").AssertOK().AssertInertia("Auth/AccountDeletion")

	// Looking at the page cancels nothing.
	if !pending() {
		t.Fatal("signing in cancelled the deletion silently")
	}

	login.PostForm("/account/deletion/cancel", url.Values{}).AssertRedirect("/dashboard")

	if pending() {
		t.Fatal("the cancel did not bring the account back")
	}

	login.Get("/dashboard").AssertOK()
}

func TestAccountDeletionIsRefusedForTheOnlyOwnerOfASharedWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := withPassword(t, app, signUp(t, app, "Ana", "ana@example.com"))
	bob := signUp(t, app, "Bob", "bob@example.com")
	join(t, app, ana, bob.User, models.RoleMember)

	ana.Client.PostForm("/account/deletion", accountDeletionForm("ana@example.com", knownPassword)).AssertStatus(303)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Find(t.Context(), ana.User.ID)
	if err != nil || u.PendingDeletion() {
		t.Fatalf("a refused deletion changed the account: %+v %v", u, err)
	}

	ana.Client.Get("/dashboard").AssertOK()
}
