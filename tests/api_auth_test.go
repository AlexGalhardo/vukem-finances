package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/auth"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func assertProblem(t *testing.T, res interface {
	JSONPath(string) (any, bool)
	Status() int
}, status int, code string) {
	t.Helper()

	if res.Status() != status {
		t.Fatalf("status = %d, want %d", res.Status(), status)
	}

	if got, _ := res.JSONPath("code"); got != code {
		t.Errorf("code = %v, want %s", got, code)
	}
}

func TestAPIAuthMatrix(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	uid := strconv.FormatInt(ana.User.ID, 10)
	good := makeToken(t, app, ana, 30, "accounts:read")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	expired := makeToken(t, app, ana, 30, "accounts:read")
	if _, err := db.ExecContext(t.Context(), "UPDATE access_tokens SET expires_at = ? WHERE id = ?",
		time.Now().Add(-time.Hour).UTC(), tokenID(expired)); err != nil {
		t.Fatal(err)
	}

	revoked := makeToken(t, app, ana, 30, "accounts:read")
	browser(t, app, ana.User, ana.Workspace.ID).Delete("/tokens/" + tokenID(revoked)).AssertRedirect("/tokens")

	noWorkspace, _, err := acc.Tokens.Create(t.Context(), uid, "legacy", []string{"accounts:read"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	wildcard, _, err := acc.Tokens.CreateWith(t.Context(), uid, "all", []string{"*"}, time.Hour,
		auth.CreateOptions{Resource: strconv.FormatInt(ana.Workspace.ID, 10)})
	if err != nil {
		t.Fatal(err)
	}

	for name, token := range map[string]string{
		"missing": "", "garbage": "nope", "unknown id": "vf_AAAA.BBBB", "expired": expired, "revoked": revoked,
		"no workspace": noWorkspace, "wildcard": wildcard,
	} {
		c := app.Client(t)
		if token != "" {
			c = c.WithToken(token)
		}

		res := c.Get("/api/v1/me")
		if res.Status() != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, res.Status())
			continue
		}

		assertProblem(t, res, 401, "unauthorized")
		res.AssertHeaderContains("Content-Type", "problem+json").AssertHeader("WWW-Authenticate", "Bearer")
	}

	res := apiClient(t, app, good).Get("/api/v1/me").AssertOK()
	res.AssertJSONPath("token_name", "test token").AssertJSONPath("role", "owner").
		AssertJSONPath("workspace.id", float64(ana.Workspace.ID)).AssertJSONPath("workspace.currency", "USD")

	// Cookies are ignored on the API, and a bearer token does nothing on the web.
	browser(t, app, ana.User, ana.Workspace.ID).Get("/api/v1/me").AssertUnauthorized()
	apiClient(t, app, good).Get("/dashboard").AssertRedirect("/login")

	// 403 for a scope the token lacks.
	assertProblem(t, apiClient(t, app, good).PostJSON("/api/v1/accounts", map[string]any{"name": "X", "kind": "cash"}), 403, "missing_scope")
}

func TestSecurity_TokenOtherWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	bobToken := makeToken(t, app, bob, 30, "accounts:read")
	anaAccount := firstOf[models.Account](t, app, ana, "")

	c := apiClient(t, app, bobToken)
	c.Get("/api/v1/me").AssertOK().AssertJSONPath("workspace.id", float64(bob.Workspace.ID))
	assertProblem(t, c.Get("/api/v1/accounts/"+anaAccount.UID), 404, "not_found")
}

func TestSecurity_RemovedMemberToken(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	cara := newVerifiedUser(t, app, "Cara", "cara@example.com")
	join(t, app, ana, cara, models.RoleMember)

	caraWho := who{cara, ana.Workspace, browser(t, app, cara, ana.Workspace.ID)}
	token := makeToken(t, app, caraWho, 30, "accounts:read")

	apiClient(t, app, token).Get("/api/v1/me").AssertOK()

	// Removed through the app, the person's tokens of the workspace are revoked: they do not
	// come back if the person is invited again.
	ana.Client.Delete("/workspace/members/" + strconv.FormatInt(cara.ID, 10)).AssertRedirect("/workspace")
	apiClient(t, app, token).Get("/api/v1/me").AssertUnauthorized()

	join(t, app, ana, cara, models.RoleMember)
	apiClient(t, app, token).Get("/api/v1/me").AssertUnauthorized()

	// A membership that disappears by another path leaves the token alive but useless: 404.
	fresh := makeToken(t, app, caraWho, 30, "accounts:read")
	apiClient(t, app, fresh).Get("/api/v1/me").AssertOK()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "DELETE FROM memberships WHERE user_id = ?", cara.ID); err != nil {
		t.Fatal(err)
	}

	assertProblem(t, apiClient(t, app, fresh).Get("/api/v1/me"), 404, "not_found")
}

func TestAPIUnknownRoutesAnswerJSON(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	c := apiClient(t, app, makeToken(t, app, ana, 30, "accounts:read"))

	assertProblem(t, c.Get("/api/v1/nope"), 404, "not_found")
	assertProblem(t, c.Delete("/api/v1/me"), 404, "not_found")
	assertProblem(t, app.Client(t).Get("/api/v1/nope"), 404, "not_found")
	c.Get("/api/v1/nope").AssertHeaderContains("Content-Type", "problem+json")
}
