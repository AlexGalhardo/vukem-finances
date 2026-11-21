package tests

import (
	"net/http"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

const setupToken = "setup-secret-for-tests"

func setupForm(email, code string) map[string]any {
	return map[string]any{
		"name": "Alexandra", "email": email, "currency": "BRL", "code": code,
		"password": testPassword, "password_confirmation": testPassword,
	}
}

func TestPersonalRegisterClosed(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	c := app.Client(t).Inertia("")

	c.Get("/register").AssertNotFound()
	c.Get("/register/check-email").AssertNotFound()
	c.PostJSON("/register", map[string]any{
		"name": "Mallory", "email": "mallory@example.com",
		"password": testPassword, "password_confirmation": testPassword,
	}).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 0)

	// Sign-up stays open in saas.
	newSaaSApp(t).Client(t).Inertia("").Get("/register").AssertOK()
}

func TestPersonalSetup(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	c := app.Client(t).Inertia("")

	c.Get("/setup").AssertOK().AssertInertia("Setup").PropEquals("currency", "USD")
	c.PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertRedirect("/login")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Where("email", "owner@example.com").First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !u.EmailVerified() {
		t.Fatal("the owner must be able to sign in at once")
	}

	m, err := orm.QueryOf[models.Membership](db).Where("user_id", u.ID).First(t.Context())
	if err != nil || m.Role != models.RoleOwner {
		t.Fatalf("membership = %+v, %v; want the owner role", m, err)
	}

	ws, err := orm.QueryOf[models.Workspace](db).Find(t.Context(), m.WorkspaceID)
	if err != nil || ws.Currency != "BRL" {
		t.Fatalf("workspace = %+v, %v; want BRL", ws, err)
	}

	// Then setup is gone for good.
	c.Get("/setup").AssertNotFound()
	c.PostJSON("/setup", setupForm("second@example.com", setupToken)).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 1)

	c.PostJSON("/login", map[string]any{"email": "owner@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")
}

func TestPersonalSetupIsOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})

	const racers = 6

	statuses := make([]int, racers)
	clients := make([]*vukemtest.Client, racers)

	for i := range clients {
		clients[i] = app.Client(t).Inertia("")
	}

	var wg sync.WaitGroup

	for i := range racers {
		wg.Go(func() {
			email := "owner" + string(rune('a'+i)) + "@example.com"
			statuses[i] = clients[i].PostJSON("/setup", setupForm(email, setupToken)).Status()
		})
	}

	wg.Wait()

	created := 0

	for _, s := range statuses {
		if s == http.StatusFound || s == http.StatusSeeOther {
			created++
		}
	}

	if created != 1 {
		t.Fatalf("statuses = %v, want exactly one redirect", statuses)
	}

	vukemtest.AssertDatabaseCount(t, app, "users", 1)
	vukemtest.AssertDatabaseCount(t, app, "workspaces", 1)
}

func TestSecurity_SetupNeedsSecret(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	c := app.Client(t).Inertia("")

	// No secret, then wrong ones: refused, nothing created.
	c.PostJSON("/setup", setupForm("owner@example.com", "")).AssertForbidden()
	c.PostJSON("/setup", setupForm("owner@example.com", "wrong")).AssertForbidden()
	c.PostJSON("/setup", setupForm("owner@example.com", setupToken+"x")).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "users", 0)

	// Five tries a minute per address: the sixth is throttled, even with the right code.
	c.PostJSON("/setup", setupForm("owner@example.com", "wrong")).AssertForbidden()
	c.PostJSON("/setup", setupForm("owner@example.com", "wrong")).AssertForbidden()
	c.PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertStatus(http.StatusTooManyRequests)
	vukemtest.AssertDatabaseCount(t, app, "users", 0)

	// The right code, from a clean address, creates the owner once.
	fresh := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	fresh.Client(t).Inertia("").PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertRedirect("/login")
	vukemtest.AssertDatabaseCount(t, fresh, "users", 1)
}

func TestSetupSecretIsGeneratedAndSpent(t *testing.T) {
	t.Parallel()

	s := accounts.NewSetupSecret("")
	if len(s.Code) < 26 {
		t.Fatalf("generated code %q is too short for 128 bits", s.Code)
	}

	if other := accounts.NewSetupSecret(""); other.Code == s.Code {
		t.Fatal("two boots printed the same code")
	}

	if s.Check("") || s.Check("nope") || !s.Check(s.Code) {
		t.Fatal("only the generated code must match")
	}

	s.Spend()

	if s.Check(s.Code) || s.Check("") {
		t.Fatal("a spent secret must match nothing")
	}

	if fixed := accounts.NewSetupSecret(" env-token "); fixed.Code != "" || !fixed.Check("env-token") {
		t.Fatal("SETUP_TOKEN must be used as given, without a generated code")
	}
}
