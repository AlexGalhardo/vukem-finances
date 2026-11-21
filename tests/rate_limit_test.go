package tests

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/config"
)

func TestWorkspaceRoutesAreRateLimited(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	for i := range config.ReadLimit {
		if res := ana.Client.Get("/accounts"); res.Status() != http.StatusOK {
			t.Fatalf("read %d answered %d", i+1, res.Status())
		}
	}

	res := ana.Client.Get("/accounts")
	res.AssertStatus(http.StatusTooManyRequests)

	if res.Header("Retry-After") == "" {
		t.Error("the 429 has no Retry-After")
	}

	// The limit is per user: Bob is not affected, and Ana's writes have their own budget.
	bob.Client.Get("/accounts").AssertOK()

	for i := range config.WriteLimit {
		if got := ana.Client.Delete("/transactions/999999").Status(); got != http.StatusNotFound {
			t.Fatalf("write %d answered %d", i+1, got)
		}
	}

	w := ana.Client.Delete("/transactions/999999")
	w.AssertStatus(http.StatusTooManyRequests)

	if w.Header("Retry-After") == "" {
		t.Error("the 429 of a write has no Retry-After")
	}
}

func TestWorkspaceImportsAndInvitesAreRateLimitedPerWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	admin := newVerifiedUser(t, app, "Adam", "adam@example.com")
	join(t, app, ana, admin, "admin")

	adam := who{admin, ana.Workspace, browser(t, app, admin, ana.Workspace.ID)}
	acc := firstAccount(t, app, ana)

	// Five uploads per workspace: the sixth, by another member, is refused.
	for i := range config.ImportLimit {
		if got := upload(t, ana, acc, statement).Status(); got != http.StatusOK {
			t.Fatalf("upload %d answered %d", i+1, got)
		}
	}

	upload(t, adam, acc, statement).AssertStatus(http.StatusTooManyRequests)

	// Ten invitations per workspace and hour.
	for i := range config.InviteLimit {
		email := "guest" + strconv.Itoa(i) + "@example.com"
		if got := ana.Client.PostForm("/workspace/invites", url.Values{"email": {email}, "role": {"viewer"}}).Status(); got == http.StatusTooManyRequests {
			t.Fatalf("invite %d was limited", i+1)
		}
	}

	adam.Client.PostForm("/workspace/invites", url.Values{"email": {"late@example.com"}, "role": {"viewer"}}).
		AssertStatus(http.StatusTooManyRequests)
	vukemtest.AssertDatabaseMissing(t, app, "invites", map[string]any{"email": "late@example.com"})

	// Another workspace has its own budget.
	bob := signUp(t, app, "Bob", "bob@example.com")
	if got := bob.Client.PostForm("/workspace/invites", url.Values{"email": {"x@example.com"}, "role": {"viewer"}}).Status(); got == http.StatusTooManyRequests {
		t.Error("another workspace shares the invite budget")
	}
}
