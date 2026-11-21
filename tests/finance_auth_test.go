package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/social"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/database/seeders"
)

func financeGoogleSignIn(t *testing.T, c *vukemtest.Client) *vukemtest.Response {
	t.Helper()
	r := c.Get("/auth/google/redirect")
	return c.Get(r.Header("Location"))
}

func TestFinanceGoogleOptional(t *testing.T) {
	t.Parallel()
	app := newApp(t)
	if accounts.GoogleEnabled(app.App) {
		t.Fatal("Google enabled without credentials")
	}
	app.Client(t).Get("/auth/google/redirect").AssertNotFound()
}

func TestSecurity_FinanceGoogleVerifiedLink(t *testing.T) {
	t.Parallel()
	app := newSaaSApp(t, vukemtest.FakeSocial(social.User{
		ID: "google-owner", Email: "OWNER@example.com", EmailVerified: true, Name: "Owner",
	}))
	newVerifiedUser(t, app, "Existing owner", "owner@example.com")
	financeGoogleSignIn(t, app.Client(t)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, app, "users", 1)
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 1)
	financeGoogleSignIn(t, app.Client(t)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 1)
}

func TestSecurity_FinanceGoogleRejectsUnverifiedClaim(t *testing.T) {
	t.Parallel()
	app := newSaaSApp(t, vukemtest.FakeSocial(social.User{
		ID: "attacker", Email: "owner@example.com",
	}))
	newVerifiedUser(t, app, "Owner", "owner@example.com")
	financeGoogleSignIn(t, app.Client(t)).AssertStatus(422)
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 0)
}

func TestSecurity_FinanceGoogleRequiresTwoFactor(t *testing.T) {
	t.Parallel()
	app := newSaaSApp(t, vukemtest.FakeSocial(social.User{
		ID: "google-2fa", Email: "owner@example.com", EmailVerified: true,
	}))
	u := newVerifiedUser(t, app, "Owner", "owner@example.com")
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	_, err = orm.QueryOf[models.TwoFactor](db).Where("id", u.ID).Update(t.Context(), orm.Set{
		"two_factor_confirmed_at": app.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	c := app.Client(t)
	financeGoogleSignIn(t, c).AssertRedirect("/two-factor-challenge")
	c.Get("/dashboard").AssertRedirect("/login")
}

func TestSecurity_FinanceLocalSeedPreservesCredentialsAndData(t *testing.T) {
	t.Parallel()
	const password = "local-example-password-123456"
	app := newApp(t, map[string]string{"ADMIN_EMAILS": "admin@example.test", "FINANCE_ADMIN_PASSWORD": password})
	for range 2 {
		if err := (seeders.FinanceAdminSeeder{}).SeedApp(t.Context(), app.App); err != nil {
			t.Fatal(err)
		}
		if err := (seeders.FinanceDemoSeeder{}).SeedApp(t.Context(), app.App); err != nil {
			t.Fatal(err)
		}
	}
	vukemtest.AssertDatabaseCount(t, app, "users", 1)
	vukemtest.AssertDatabaseCount(t, app, "workspaces", 1)
	vukemtest.AssertDatabaseCount(t, app, "transactions", 24)
	vukemtest.AssertDatabaseCount(t, app, "budgets", 3)
	// The local account is verified and uses the same native password login.
	app.Client(t).PostForm("/login", url.Values{
		"email":    {"admin@example.test"},
		"password": {password},
	}).AssertRedirect("/dashboard")
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	ws, err := orm.QueryOf[models.Workspace](db).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	edited := workspaces.WithTenant(t.Context(), ws.ID)
	if _, err := orm.QueryOf[models.Account](db).Update(edited, orm.Set{"name": "My edited checking"}); err != nil {
		t.Fatal(err)
	}
	if err := (seeders.FinanceDemoSeeder{}).SeedApp(t.Context(), app.App); err != nil {
		t.Fatal(err)
	}
	vukemtest.AssertDatabaseHas(t, app, "accounts", map[string]any{"name": "My edited checking"})
}

func TestSecurity_PersonalSocialDoesNotCreateUsers(t *testing.T) {
	t.Parallel()

	fake := social.User{ID: "google-stranger", Email: "stranger@example.com", EmailVerified: true, Name: "Stranger"}

	// An unknown identity gets the 404 of /register and creates nothing.
	app := newPersonalApp(t, vukemtest.FakeSocial(fake))
	financeGoogleSignIn(t, app.Client(t)).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 0)
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 0)

	// Not even an existing address with another person's account unverified.
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	squat := models.User{Name: "Squatter", Email: "stranger@example.com", Password: "not used"}
	if err := orm.Create(t.Context(), db, &squat); err != nil {
		t.Fatal(err)
	}

	financeGoogleSignIn(t, app.Client(t)).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 1)
	vukemtest.AssertDatabaseCount(t, app, "social_accounts", 0)

	// A linked identity signs in.
	known := newPersonalApp(t, vukemtest.FakeSocial(fake))
	owner := newVerifiedUser(t, known, "Owner", "owner@example.com")

	kdb, err := known.DB()
	if err != nil {
		t.Fatal(err)
	}

	if err := orm.Create(t.Context(), kdb, &models.SocialAccount{
		UserID: owner.ID, Provider: "google", ProviderID: "google-stranger",
	}); err != nil {
		t.Fatal(err)
	}

	financeGoogleSignIn(t, known.Client(t)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, known, "users", 1)

	// A verified local account with the same verified address gets linked, not created.
	byMail := newPersonalApp(t, vukemtest.FakeSocial(fake))
	newVerifiedUser(t, byMail, "Owner", "stranger@example.com")
	financeGoogleSignIn(t, byMail.Client(t)).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseCount(t, byMail, "users", 1)
	vukemtest.AssertDatabaseCount(t, byMail, "social_accounts", 1)
}
