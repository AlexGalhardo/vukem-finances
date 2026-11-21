package tests

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

func tokenID(plain string) string {
	id, _, _ := strings.Cut(strings.TrimPrefix(plain, "vf_"), ".")
	return id
}

func postToken(t *testing.T, app *vukemtest.App, w who, days string, scopes ...string) *vukemtest.Response {
	return browser(t, app, w.User, w.Workspace.ID).Inertia("").PostForm("/tokens", url.Values{"name": {"ci"}, "scopes": scopes, "days": {days}})
}

func TestTokenCreate(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	secret := makeToken(t, app, ana, 30, "accounts:read", "transactions:read")
	if !strings.HasPrefix(secret, "vf_") {
		t.Errorf("secret = %q, want the vf_ prefix", secret)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var expires time.Time
	var scopes, resource string

	if err := db.QueryRowContext(t.Context(), "SELECT expires_at, abilities, resource FROM access_tokens WHERE id = ?", tokenID(secret)).
		Scan(&expires, &scopes, &resource); err != nil {
		t.Fatal(err)
	}

	if d := time.Until(expires); d < 29*24*time.Hour || d > 30*24*time.Hour+time.Minute {
		t.Errorf("expires in %v, want 30 days", d)
	}

	if scopes != `["accounts:read","transactions:read"]` || resource != strconv.FormatInt(ana.Workspace.ID, 10) {
		t.Errorf("abilities = %s, resource = %s", scopes, resource)
	}

	ana.Client.Inertia("").Get("/tokens").AssertInertia("Tokens").
		PropEquals("tokens.0.status", "live").PropEquals("tokens.0.name", "test token").PropEquals("max", float64(20))

	refused := map[string]*vukemtest.Response{
		"wildcard scope":  postToken(t, app, ana, "30", "*"),
		"unknown scope":   postToken(t, app, ana, "30", "accounts:read", "billing:manage"),
		"no scope":        postToken(t, app, ana, "30"),
		"too long a life": postToken(t, app, ana, "366", "accounts:read"),
		"no expiry":       postToken(t, app, ana, "0", "accounts:read"),
		"odd expiry":      postToken(t, app, ana, "45", "accounts:read"),
	}

	for name, res := range refused {
		if res.Status() == 200 {
			t.Errorf("%s: the token was made", name)
		}
	}

	if n := rowCount(t, app, "access_tokens"); n != 1 {
		t.Errorf("access_tokens has %d rows, want only the first token", n)
	}
}

func TestTokenLimits(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	var first string

	for i := range 20 {
		s := makeToken(t, app, ana, 90, "accounts:read")
		if i == 0 {
			first = s
		}
	}

	if res := postToken(t, app, ana, "90", "accounts:read"); res.Status() == 200 {
		t.Fatal("the 21st live token was made")
	}

	// Another person is not affected; revoking frees a place.
	makeToken(t, app, bob, 90, "accounts:read")

	ana.Client.Delete("/tokens/" + tokenID(first)).AssertRedirect("/tokens")
	makeToken(t, app, ana, 365, "accounts:read")

	// A second workspace of the same person has its own count.
	ws2, err := service(t, app).Create(t.Context(), ana.User, "Second", "USD")
	if err != nil {
		t.Fatal(err)
	}

	makeToken(t, app, who{ana.User, ws2, browser(t, app, ana.User, ws2.ID)}, 30, "accounts:read")
}

func TestTokenRevoke(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	secret := makeToken(t, app, ana, 30, "accounts:read")
	id := tokenID(secret)

	bob.Client.Delete("/tokens/" + id).AssertNotFound()

	ws2, err := service(t, app).Create(t.Context(), ana.User, "Second", "USD")
	if err != nil {
		t.Fatal(err)
	}

	other := browser(t, app, ana.User, ws2.ID)
	other.Delete("/tokens/" + id).AssertNotFound()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	revoked := func() bool {
		var n int64
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM access_tokens WHERE id = ? AND revoked_at IS NOT NULL", id).Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n == 1
	}

	if revoked() {
		t.Fatal("a foreign call revoked the token")
	}

	ana.Client.Delete("/tokens/" + id).AssertRedirect("/tokens")

	if !revoked() {
		t.Fatal("the token is not revoked")
	}

	ana.Client.Inertia("").Get("/tokens").AssertInertia("Tokens").PropEquals("tokens.0.status", "revoked")
	ana.Client.Delete("/tokens/" + id).AssertNotFound()
}

func TestSecurity_TokenShownOnce(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	res := postToken(t, app, ana, "30", "accounts:read").AssertOK().AssertHeaderContains("Cache-Control", "no-store")
	secret, _ := res.JSONPath("props.created.secret")

	plain, _ := secret.(string)
	if plain == "" {
		t.Fatal("no secret on the creation answer")
	}

	_, hashSecret, _ := strings.Cut(plain, ".")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var hash string
	if err := db.QueryRowContext(t.Context(), "SELECT hash FROM access_tokens WHERE id = ?", tokenID(plain)).Scan(&hash); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/tokens", "/dashboard", "/workspace"} {
		body := ana.Client.Inertia("").Get(path).Body()
		for _, leak := range []string{plain, hashSecret, hash} {
			if strings.Contains(body, leak) {
				t.Errorf("GET %s carries the secret or its hash", path)
			}
		}
	}

	ana.Client.Inertia("").Get("/tokens").AssertInertia("Tokens").PropEquals("created", nil)
}
