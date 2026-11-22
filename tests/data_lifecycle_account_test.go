package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// TestDataLifecycle_AccountDeletionLeavesNoTraceOfThePerson follows an account:
// the person is a member of someone else's workspace too, received an
// invitation, signed in, paired a phone and asked for a password reset. After
// the grace period and the purge their address and name are in no table; what
// they gave to the shared workspace stays, anonymously.
func TestDataLifecycle_AccountDeletionLeavesNoTraceOfThePerson(t *testing.T) {
	t.Parallel()

	s := lifecycleApp(t)
	ana := s.SignUp(t, "Ana Lifecycle", "ana-lifecycle@example.com")
	bob := s.SignUp(t, "Bob Keeper", "bob-keeper@example.com")
	s.AddMember(t, bob.Workspace.ID, ana, models.RoleMember)

	w := withPassword(t, s.App, asWho(ana))
	b := withPassword(t, s.App, asWho(bob))

	seedWorkspaceData(t, s.App, w, "lca")

	// What she gave to Bob's workspace is shared data and stays.
	seedWorkspaceData(t, s.App, asWho(bob), "lcb")

	shared := apiClient(t, s.App, rawToken(t, s.App, ana.User, bob.Workspace.ID, allScopes()...))
	sharedAccount := uidOf(t, shared.Get("/api/v1/accounts").AssertOK(), "data.0.uid")
	shared.PostJSON("/api/v1/transactions", map[string]any{
		"account_uid": sharedAccount, "kind": "expense", "amount_cents": 700, "date": "2026-11-21", "description": "lcs-lunch",
	}).AssertCreated()

	// Bob invited her address, and she asked for a reset and connected an app.
	b.Client.PostForm("/workspace/invites", url.Values{"email": {"ana-lifecycle@example.com"}, "role": {"viewer"}})
	s.App.Client(t).WithHeader("Sec-Fetch-Site", "same-origin").PostForm("/forgot-password", url.Values{"email": {"ana-lifecycle@example.com"}})
	s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

	login := s.App.Client(t).WithHeader("Sec-Fetch-Site", "same-origin")
	login.PostForm("/login", url.Values{"email": {"ana-lifecycle@example.com"}, "password": {knownPassword}}).AssertRedirect("/dashboard")

	w.Client.PostForm("/account/deletion", accountDeletionForm("ana-lifecycle@example.com", knownPassword)).AssertRedirect("/login")

	// A session opened during the grace period is a row too.
	login.PostForm("/login", url.Values{"email": {"ana-lifecycle@example.com"}, "password": {knownPassword}}).AssertRedirect("/dashboard")
	login.Get("/account/deletion").AssertOK()

	endSubscriptions(t, s.App)
	dueNow(t, s.App, "workspaces SET purge_at = ? WHERE id = ?", w.Workspace.ID) // the workspace that went with her
	dueNow(t, s.App, "users SET purge_at = ? WHERE id = ?", w.User.ID)
	runPurge(t, s.App)

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM users WHERE id = ?", w.User.ID); n != 0 {
		t.Fatal("the user row is still there")
	}

	for _, needle := range []string{"ana-lifecycle@example.com", "Ana Lifecycle", "lca-"} {
		if where := rowsHolding(t, s.App, needle); len(where) > 0 {
			t.Errorf("%q is still in %v", needle, where)
		}
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM sync_devices WHERE user_id = ?", w.User.ID); n != 0 {
		t.Errorf("%d sync devices of the person remain", n)
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM transactions WHERE workspace_id = ? AND description = ?", bob.Workspace.ID, "lcs-lunch"); n != 1 {
		t.Errorf("the shared transaction was lost (%d rows)", n)
	}
}

// TestDataLifecycle_NoSessionOutlivesThePurge: signing in while the account
// waits for deletion opens a session; the purge must remove it with the person.
func TestDataLifecycle_NoSessionOutlivesThePurge(t *testing.T) {
	t.Parallel()

	s := lifecycleApp(t)
	ana := s.SignUp(t, "Ana Lifecycle", "ana-lifecycle@example.com")
	w := withPassword(t, s.App, asWho(ana))

	w.Client.PostForm("/account/deletion", accountDeletionForm("ana-lifecycle@example.com", knownPassword)).AssertRedirect("/login")

	login := s.App.Client(t).WithHeader("Sec-Fetch-Site", "same-origin")
	login.PostForm("/login", url.Values{"email": {"ana-lifecycle@example.com"}, "password": {knownPassword}}).AssertRedirect("/dashboard")
	login.Get("/account/deletion").AssertOK()

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM sessions WHERE user_id = ?", "web:"+idString(w.User.ID)); n == 0 {
		t.Fatal("the sign-in during the grace period opened no session row")
	}

	endSubscriptions(t, s.App)
	dueNow(t, s.App, "workspaces SET purge_at = ? WHERE id = ?", w.Workspace.ID)
	dueNow(t, s.App, "users SET purge_at = ? WHERE id = ?", w.User.ID)
	runPurge(t, s.App)

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM users WHERE id = ?", w.User.ID); n != 0 {
		t.Fatal("the user row is still there")
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM sessions WHERE user_id = ?", "web:"+idString(w.User.ID)); n != 0 {
		t.Errorf("%d sessions of the person remain after the purge", n)
	}
}
