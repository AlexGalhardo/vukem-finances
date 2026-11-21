package tests

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// firstLimited calls do until it answers 429 and returns the number of the
// call that did. The in-memory limiter refills continuously, so a slow run
// (the race detector) may gain a few credits; slack covers that, and a call
// number below limit+1 is still a failure.
func firstLimited(t *testing.T, limit int, do func() *vukemtest.Response) *vukemtest.Response {
	t.Helper()

	const slack = 30

	for i := 1; i <= limit+slack; i++ {
		res := do()
		if res.Status() != http.StatusTooManyRequests {
			continue
		}

		if i <= limit {
			t.Fatalf("call %d was limited, the limit is %d", i, limit)
		}

		return res
	}

	t.Fatalf("no call up to %d was limited", limit+slack)

	return nil
}

func TestWorkspaceRoutesAreRateLimited(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	res := firstLimited(t, config.ReadLimit, func() *vukemtest.Response { return ana.Client.Get("/accounts") })
	if res.Header("Retry-After") == "" {
		t.Error("the 429 has no Retry-After")
	}

	// The limit is per user: Bob is not affected, and Ana's writes have their own budget.
	bob.Client.Get("/accounts").AssertOK()

	res = firstLimited(t, config.WriteLimit, func() *vukemtest.Response { return ana.Client.Delete("/transactions/999999") })
	if res.Header("Retry-After") == "" {
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
