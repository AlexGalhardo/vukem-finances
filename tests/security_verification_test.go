package tests

import (
	"log/slog"
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/social"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

func registerForm(name, email string) map[string]any {
	return map[string]any{
		"name": name, "email": email, "password": testPassword, "password_confirmation": testPassword,
	}
}

func userByEmail(t *testing.T, app *vukemtest.App, email string) models.User {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Where("email", email).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return u
}

func TestSecurity_SaaSRequiresEmailVerification(t *testing.T) {
	t.Parallel()

	// Unset, and explicitly off outside production: a self-asserted address is
	// never marked verified, and the owner of the address gets the link.
	for name, env := range map[string]map[string]string{
		"unset": {}, "false": {"AUTH_REQUIRE_EMAIL_VERIFICATION": "false"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app := newSaaSApp(t, env)
			c := app.Client(t).Inertia("")
			c.PostJSON("/register", registerForm("Alexandra", "alexandra@example.com")).
				AssertRedirect("/register/check-email")

			if sent := app.Mails().SentTo("alexandra@example.com"); len(sent) != 1 {
				t.Fatalf("verification mails = %d, want 1", len(sent))
			}

			if userByEmail(t, app, "alexandra@example.com").EmailVerified() {
				t.Fatal("a self-asserted address must not be marked verified")
			}

			// Signed in but unverified: the app asks for the verification.
			c.PostJSON("/login", map[string]any{"email": "alexandra@example.com", "password": testPassword}).
				AssertRedirect("/dashboard")
			c.Get("/dashboard").AssertRedirect("/email/verify")
		})
	}

	// Production refuses an explicit off, in saas only.
	log := slog.New(slog.DiscardHandler)
	for flag, want := range map[string]bool{"": false, "true": false, "TRUE": false, "false": true, "0": true, "no": true} {
		err := accounts.CheckEmailVerification(config.EditionSaaS, flag, true, log)
		if (err != nil) != want {
			t.Fatalf("saas production flag %q: err = %v, want error %v", flag, err, want)
		}
	}

	if err := accounts.CheckEmailVerification(config.EditionSaaS, "false", false, log); err != nil {
		t.Fatalf("outside production the flag is ignored, got %v", err)
	}

	if err := accounts.CheckEmailVerification(config.EditionPersonal, "false", true, log); err != nil {
		t.Fatalf("personal accounts are verified when created, got %v", err)
	}
}

func TestSecurity_SelfAssertedAdminEmailIsNotAdmin(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t, map[string]string{"AUTH_REQUIRE_EMAIL_VERIFICATION": "false"})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	// The attacker signs up with the admin's address, and with the address of a pending invite.
	attacker := app.Client(t).Inertia("")
	attacker.PostJSON("/register", registerForm("Mallory", "boss@example.com")).AssertRedirect("/register/check-email")
	attacker.PostJSON("/login", map[string]any{"email": "boss@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")

	attacker.Get("/admin/revenue").AssertRedirect("/email/verify")
	attacker.Get("/admin").AssertForbidden()

	ana.Client.PostForm("/workspace/invites", url.Values{"email": {"victim@example.com"}, "role": {"admin"}}).
		AssertRedirect("/workspace")

	token := inviteTokenRe.FindStringSubmatch(app.Mails().SentTo("victim@example.com")[0].HTML)[1]

	squatter := app.Client(t).Inertia("")
	squatter.PostJSON("/register", registerForm("Mallory", "victim@example.com")).AssertRedirect("/register/check-email")
	squatter.PostJSON("/login", map[string]any{"email": "victim@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")
	squatter.PostJSON("/invites/"+token, nil).AssertRedirect("/email/verify")

	vukemtest.AssertDatabaseCount(t, app, "memberships", 1)
}

func TestSecurity_SquattedAccountLosesAccessOnGoogleLogin(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t, vukemtest.FakeSocial(social.User{
		ID: "google-owner", Email: "owner@example.com", EmailVerified: true, Name: "Owner",
	}))

	// First registrant of someone else's address, with a live session.
	squatter := app.Client(t).Inertia("")
	squatter.PostJSON("/register", registerForm("Mallory", "owner@example.com")).AssertRedirect("/register/check-email")
	squatter.PostJSON("/login", map[string]any{"email": "owner@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")
	squatter.Get("/dashboard").AssertRedirect("/email/verify")

	// The real owner signs in with Google.
	financeGoogleSignIn(t, app.Client(t)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, app, "users", 1)

	if !userByEmail(t, app, "owner@example.com").EmailVerified() {
		t.Fatal("Google proved the address: the account is verified now")
	}

	// The squatter's session and password stopped working.
	squatter.Get("/dashboard").AssertRedirect("/login")

	again := app.Client(t).Inertia("")
	again.PostJSON("/login", map[string]any{"email": "owner@example.com", "password": testPassword}).
		AssertSessionMissing("login.web")
	again.Get("/dashboard").AssertRedirect("/login")
}
