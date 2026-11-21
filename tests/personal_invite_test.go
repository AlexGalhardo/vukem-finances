package tests

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

var inviteTokenRe = regexp.MustCompile(`/invites/([^/"\s?]+)`)

// personalInvite makes a personal installation with an owner and one invite
// for bob@example.com, and returns the invite token.
func personalInvite(t *testing.T, role string) (*vukemtest.App, who, string) {
	t.Helper()

	app := newPersonalApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	ana.Client.PostForm("/workspace/invites", url.Values{"email": {"bob@example.com"}, "role": {role}}).
		AssertRedirect("/workspace")

	sent := app.Mails().SentTo("bob@example.com")
	if len(sent) != 1 {
		t.Fatalf("invite mails = %d, want 1", len(sent))
	}

	m := inviteTokenRe.FindStringSubmatch(sent[0].HTML)
	if m == nil {
		t.Fatalf("no invite link in %s", sent[0].HTML)
	}

	return app, ana, m[1]
}

func TestPersonalInviteRegister(t *testing.T) {
	t.Parallel()

	app, ana, token := personalInvite(t, "admin")
	guest := app.Client(t).Inertia("")

	// The invite link of a person with no account leads to the registration.
	guest.Get("/invites/" + token).AssertRedirect("/invites/" + token + "/register")
	guest.Get("/invites/"+token+"/register").AssertOK().AssertInertia("InviteRegister").
		PropEquals("email", "bob@example.com")

	// The email is the invite's: a field of the form cannot change it.
	guest.PostJSON("/invites/"+token+"/register", map[string]any{
		"name": "Bob", "email": "mallory@example.com",
		"password": testPassword, "password_confirmation": testPassword,
	}).AssertRedirect("/login")

	vukemtest.AssertDatabaseCount(t, app, "users", 2)
	vukemtest.AssertDatabaseHas(t, app, "users", map[string]any{"email": "bob@example.com"})

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	bob, err := orm.QueryOf[models.User](db).Where("email", "bob@example.com").First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !bob.EmailVerified() {
		t.Fatal("the invited account is verified: the link went to its address")
	}

	vukemtest.AssertDatabaseHas(t, app, "memberships", map[string]any{
		"workspace_id": ana.Workspace.ID, "user_id": bob.ID, "role": "admin",
	})
	vukemtest.AssertDatabaseCount(t, app, "workspaces", 1)

	// The token is spent: a second registration, and the page, are 404.
	guest.PostJSON("/invites/"+token+"/register", map[string]any{
		"name": "Bob 2", "password": testPassword, "password_confirmation": testPassword,
	}).AssertNotFound()
	guest.Get("/invites/" + token + "/register").AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 2)

	// Bob signs in and lands in the same workspace.
	bobClient := app.Client(t).Inertia("")
	bobClient.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")
	bobClient.Get("/dashboard").AssertInertia("Dashboard").
		PropEquals("workspace.current.id", float64(ana.Workspace.ID)).PropEquals("workspace.role", "admin")
}

func TestPersonalInviteRegisterRefusals(t *testing.T) {
	t.Parallel()

	app, _, token := personalInvite(t, "member")
	guest := app.Client(t).Inertia("")
	form := map[string]any{"name": "Bob", "password": testPassword, "password_confirmation": testPassword}

	// An unknown token.
	guest.Get("/invites/nope/register").AssertNotFound()
	guest.PostJSON("/invites/nope/register", form).AssertNotFound()

	// An expired one.
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := orm.QueryOf[models.Invite](db).Where("email", "bob@example.com").
		Update(t.Context(), orm.Set{"expires_at": time.Now().Add(-time.Hour).UTC()}); err != nil {
		t.Fatal(err)
	}

	guest.PostJSON("/invites/"+token+"/register", form).AssertNotFound()
	guest.Get("/invites/" + token).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 1)
}

func TestSecurity_InviteTokenNotInQuery(t *testing.T) {
	t.Parallel()

	app, _, token := personalInvite(t, "member")
	guest := app.Client(t).Inertia("")

	// A token in a query string is no way in.
	guest.Get("/register?invite=" + token).AssertNotFound()
	guest.Get("/invites/register?token=" + token).AssertNotFound()
	guest.PostJSON("/register?invite="+token, map[string]any{
		"name": "Mallory", "email": "mallory@example.com",
		"password": testPassword, "password_confirmation": testPassword,
	}).AssertNotFound()
	vukemtest.AssertDatabaseCount(t, app, "users", 1)

	// The path form works and the token stays out of the access log.
	guest.Get("/invites/" + token + "/register").AssertOK()
	guest.PostJSON("/invites/"+token+"/register", map[string]any{
		"name": "Bob", "password": testPassword, "password_confirmation": testPassword,
	}).AssertRedirect("/login")

	logs := app.Logs()
	if !strings.Contains(logs, "/invites/") {
		t.Fatalf("the access log has no invite request to check:\n%s", logs)
	}

	if strings.Contains(logs, token) {
		t.Fatalf("the invite token is in the logs:\n%s", logs)
	}
}

func TestPersonalInviteLinkShownToOwner(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	ana.Client.PostForm("/workspace/invites", url.Values{"email": {"bob@example.com"}, "role": {"member"}}).
		AssertRedirect("/workspace")

	// The next page carries the full link and its expiry, once.
	page := ana.Client.Inertia("").Get("/workspace").AssertInertia("Workspace")

	token := inviteTokenRe.FindStringSubmatch(app.Mails().SentTo("bob@example.com")[0].HTML)[1]
	flash, _ := page.Page()["flash"].(map[string]any)

	link, _ := flash["inviteLink"].(map[string]any)
	if u, _ := link["url"].(string); !strings.HasSuffix(u, "/invites/"+token) || !strings.HasPrefix(u, "http") {
		t.Fatalf("flash invite link = %v, want the full link to /invites/%s", link, token)
	}

	if exp, _ := link["expiresAt"].(string); exp == "" {
		t.Fatalf("flash invite link %v has no expiry", link)
	}

	ana.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").MissingFlash("inviteLink")
}
