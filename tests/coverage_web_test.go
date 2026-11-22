package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/social"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

// TestPagesOfTheShell loads the pages no other test names as a component, as
// a signed-in person or a visitor.
func TestPagesOfTheShell(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	app.Client(t).Inertia("").Get("/register").AssertInertia("Auth/Register")
	ana.Client.Inertia("").Get("/devices").AssertInertia("Devices")

	tgApp, _, _ := tgApp(t)
	tgAna := signUp(t, tgApp, "Ana", "ana@example.com")
	tgAna.Client.Inertia("").Get("/settings/telegram").AssertInertia("Settings/Telegram")

	s := oauthtest.New(t, nil)
	bob := s.SignUp(t, "Bob", "bob@example.com")
	s.Browser(t, bob, bob.Workspace.ID).Inertia("").Get("/connected-apps").AssertInertia("ConnectedApps")
}

func TestDraftStatusFollowsTheRole(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana, _, _, viewer := allRoles(t, app)
	asViewer := browser(t, app, viewer.User, ana.Workspace.ID)

	for name, c := range map[string]*vukemtest.Client{"owner": ana.Client, "viewer": asViewer} {
		var out struct {
			Enabled   bool `json:"enabled"`
			TextLimit int  `json:"textLimit"`
		}

		c.Get("/transactions/draft/status").AssertOK().JSON(&out)

		// The categorize feature is off until an owner turns it on.
		if out.Enabled || out.TextLimit <= 0 {
			t.Errorf("%s: draft status = %+v, want it off with a text limit", name, out)
		}
	}

	app.Client(t).Get("/transactions/draft/status").AssertRedirect("/login")
}

func TestEmailVerificationRoutes(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)
	c := app.Client(t).Inertia("")
	c.PostJSON("/register", signUpForm("Ana Silva", "ana@example.com", testPassword)).AssertRedirect("/register/check-email")

	verify := link(t, app, "ana@example.com")
	if !strings.HasPrefix(verify, "/email/verify/") {
		t.Fatalf("verification link = %s", verify)
	}

	unverified := clientFor(t, app, "dan@example.com", false)

	// A fresh mail on request, and a link with a broken signature opens nothing.
	before := len(app.Mails().SentTo("dan@example.com"))
	unverified.PostForm("/email/verification-notification", url.Values{}).AssertStatus(http.StatusSeeOther)

	if got := len(app.Mails().SentTo("dan@example.com")); got != before+1 {
		t.Fatalf("mails to dan = %d, want %d", got, before+1)
	}

	u, _ := url.Parse(verify)
	id := strings.TrimPrefix(u.Path, "/email/verify/")

	c.PostJSON("/login", map[string]any{"email": "ana@example.com", "password": testPassword}).AssertRedirect("/dashboard")

	tampered := u.Query()
	tampered.Set("signature", "tampered")

	c.Get("/email/verify/" + id + "?" + tampered.Encode()).AssertForbidden()

	if n := rowCountWhere(t, app, "users", "email = 'ana@example.com' AND email_verified_at IS NOT NULL"); n != 0 {
		t.Fatal("a tampered link verified the address")
	}

	c.Get("/email/verify/" + id + "?" + u.RawQuery).AssertRedirect("/dashboard")

	if n := rowCountWhere(t, app, "users", "email = 'ana@example.com' AND email_verified_at IS NOT NULL"); n != 1 {
		t.Fatal("the mailed link did not verify the address")
	}
}

func TestResetPasswordPageTakesTheTokenInThePath(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)
	register(t, app, app.Client(t).Inertia(""), "cid@example.com")

	c := app.Client(t).Inertia("")
	c.PostJSON("/forgot-password", map[string]any{"email": "cid@example.com"}).AssertRedirect("/forgot-password")

	u, err := url.Parse(link(t, app, "cid@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	token := strings.TrimPrefix(u.Path, "/reset-password/")
	c.Get("/reset-password/"+token).AssertInertia("Auth/ResetPassword").PropEquals("token", token)
}

func TestSocialCallbackNeedsAStartedFlow(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t, vukemtest.FakeSocial(social.User{ID: "g-1", Email: "ana@example.com", EmailVerified: true}))

	// A callback that arrives with no redirect before it (no state in the session) signs nobody in.
	res := app.Client(t).Get("/auth/google/callback?code=abc&state=forged")
	if res.Status() < 300 {
		t.Fatalf("a callback with no started flow answered %d", res.Status())
	}

	vukemtest.AssertDatabaseCount(t, app, "users", 0)
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 0)
}
