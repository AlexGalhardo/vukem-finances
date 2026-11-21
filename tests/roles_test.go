package tests

import (
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// team is a workspace with an owner and an admin, a member and a viewer.
type team struct {
	app                          *vukemtest.App
	owner, admin, member, viewer who
}

func newTeam(t *testing.T) team {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	mk := func(name, email, role string) who {
		u := newVerifiedUser(t, app, name, email)
		join(t, app, ana, u, role)

		return who{u, ana.Workspace, browser(t, app, u, ana.Workspace.ID)}
	}

	return team{
		app: app, owner: ana,
		admin:  mk("Adam", "adam@example.com", models.RoleAdmin),
		member: mk("Mia", "mia@example.com", models.RoleMember),
		viewer: mk("Vera", "vera@example.com", models.RoleViewer),
	}
}

func roleOf(t *testing.T, app *vukemtest.App, w who, u models.User) string {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var role string
	if err := db.QueryRowContext(t.Context(),
		"SELECT role FROM memberships WHERE workspace_id = ? AND user_id = ?", w.Workspace.ID, u.ID).Scan(&role); err != nil {
		t.Fatal(err)
	}

	return role
}

func owners(t *testing.T, app *vukemtest.App, w who) int64 {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int64
	if err := db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM memberships WHERE workspace_id = ? AND role = 'owner'", w.Workspace.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func setRole(c *vukemtest.Client, u models.User, role string) *vukemtest.Response {
	return c.PatchJSON("/workspace/members/"+idString(u.ID), map[string]any{"role": role})
}

func transfer(c *vukemtest.Client, to models.User) *vukemtest.Response {
	return c.PostForm("/workspace/owner", url.Values{"userId": {idString(to.ID)}})
}

func TestChangeRole(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)

	setRole(tm.owner.Client, tm.member.User, models.RoleAdmin).AssertStatus(http.StatusSeeOther)

	if got := roleOf(t, tm.app, tm.owner, tm.member.User); got != models.RoleAdmin {
		t.Fatalf("role = %s, want admin", got)
	}

	// An admin manages the roles of the others, and the new role takes effect at once.
	setRole(tm.admin.Client, tm.viewer.User, models.RoleMember).AssertStatus(http.StatusSeeOther)
	addTransaction(t, tm.viewer, firstAccount(t, tm.app, tm.owner), 0, "expense", "1.00", today()).
		AssertRedirect("/transactions")

	// A role that does not exist is refused.
	setRole(tm.owner.Client, tm.viewer.User, "root").AssertStatus(http.StatusUnprocessableEntity)

	// Members and viewers cannot change roles.
	setRole(tm.viewer.Client, tm.viewer.User, models.RoleAdmin).AssertStatus(http.StatusForbidden)
	setRole(tm.owner.Client, tm.admin.User, models.RoleViewer).AssertStatus(http.StatusSeeOther)
	setRole(tm.admin.Client, tm.viewer.User, models.RoleAdmin).AssertForbidden()

	// Someone who is not in the workspace is a 404, and no other workspace changes.
	stranger := signUp(t, tm.app, "Sam", "sam@example.com")
	setRole(tm.owner.Client, stranger.User, models.RoleViewer).AssertNotFound()

	if got := roleOf(t, tm.app, stranger, stranger.User); got != models.RoleOwner {
		t.Errorf("a role change reached another workspace: %s", got)
	}
}

func TestSecurity_AdminCannotPromoteToOwner(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)

	// Not another member, not themselves.
	for _, target := range []models.User{tm.member.User, tm.admin.User} {
		setRole(tm.admin.Client, target, models.RoleOwner).AssertStatus(http.StatusUnprocessableEntity)
	}

	// Not the owner either way, and the owner cannot be demoted by a role change.
	setRole(tm.admin.Client, tm.owner.User, models.RoleMember).AssertStatus(http.StatusConflict)
	setRole(tm.owner.Client, tm.owner.User, models.RoleAdmin).AssertStatus(http.StatusConflict)
	setRole(tm.owner.Client, tm.member.User, models.RoleOwner).AssertStatus(http.StatusUnprocessableEntity)

	if got := roleOf(t, tm.app, tm.owner, tm.owner.User); got != models.RoleOwner {
		t.Errorf("the owner's role became %s", got)
	}

	if got := owners(t, tm.app, tm.owner); got != 1 {
		t.Errorf("owners = %d, want 1", got)
	}
}

func TestTransferOwnership(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)

	// Only the owner transfers, and only to a member of this workspace.
	transfer(tm.admin.Client, tm.admin.User).AssertForbidden()
	transfer(tm.member.Client, tm.member.User).AssertForbidden()

	stranger := signUp(t, tm.app, "Sam", "sam@example.com")
	transfer(tm.owner.Client, stranger.User).AssertStatus(http.StatusSeeOther)
	transfer(tm.owner.Client, tm.owner.User).AssertStatus(http.StatusSeeOther)

	if got := owners(t, tm.app, tm.owner); got != 1 || roleOf(t, tm.app, tm.owner, tm.owner.User) != models.RoleOwner {
		t.Fatalf("a refused transfer changed the owners: %d", got)
	}

	transfer(tm.owner.Client, tm.member.User).AssertRedirect("/workspace")

	if got := roleOf(t, tm.app, tm.owner, tm.member.User); got != models.RoleOwner {
		t.Errorf("new owner role = %s", got)
	}

	if got := roleOf(t, tm.app, tm.owner, tm.owner.User); got != models.RoleAdmin {
		t.Errorf("old owner role = %s, want admin", got)
	}

	// The old owner lost the owner's abilities at once.
	transfer(tm.owner.Client, tm.admin.User).AssertForbidden()
	tm.owner.Client.Inertia("").Get("/billing").AssertInertia("Billing").PropEquals("canManage", false)
	tm.owner.Client.PostForm("/billing/portal", nil).AssertForbidden()
}

func TestLastOwner(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)

	// The owner cannot be removed, demoted or leave.
	tm.admin.Client.Delete("/workspace/members/" + idString(tm.owner.User.ID)).AssertStatus(http.StatusSeeOther)
	setRole(tm.admin.Client, tm.owner.User, models.RoleAdmin).AssertStatus(http.StatusConflict)
	tm.owner.Client.PostForm("/workspace/leave", nil).AssertStatus(http.StatusSeeOther)

	if got := roleOf(t, tm.app, tm.owner, tm.owner.User); got != models.RoleOwner {
		t.Fatalf("the owner is now %s", got)
	}

	// Anybody else may leave, and is out at once.
	tm.viewer.Client.PostForm("/workspace/leave", nil).AssertRedirect("/dashboard")
	vukemtest.AssertDatabaseMissing(t, tm.app, "memberships", map[string]any{
		"workspace_id": tm.owner.Workspace.ID, "user_id": tm.viewer.User.ID,
	})

	// Two transfers at the same time still leave exactly one owner.
	var wg sync.WaitGroup

	codes := make([]int, 2)

	for i, to := range []models.User{tm.admin.User, tm.member.User} {
		wg.Go(func() {
			c := browser(t, tm.app, tm.owner.User, tm.owner.Workspace.ID)
			codes[i] = transfer(c, to).Status()
		})
	}

	wg.Wait()

	if got := owners(t, tm.app, tm.owner); got != 1 {
		t.Fatalf("owners after two concurrent transfers = %d (statuses %v), want 1", got, codes)
	}

	if wins := countOf(codes, http.StatusSeeOther); wins != 1 {
		t.Errorf("statuses %v: exactly one transfer must win", codes)
	}

	// A removal racing a transfer cannot orphan the workspace either: the
	// target of the transfer is removed, or becomes the owner, never both.
	tm2 := newTeam(t)

	for _, run := range []func(){
		func() { transfer(browser(t, tm2.app, tm2.owner.User, tm2.owner.Workspace.ID), tm2.member.User) },
		func() {
			browser(t, tm2.app, tm2.admin.User, tm2.owner.Workspace.ID).
				Delete("/workspace/members/" + idString(tm2.member.User.ID))
		},
	} {
		wg.Go(run)
	}

	wg.Wait()

	if got := owners(t, tm2.app, tm2.owner); got != 1 {
		t.Errorf("owners after a transfer racing a removal = %d, want 1", got)
	}
}

func countOf(codes []int, code int) int {
	n := 0

	for _, c := range codes {
		if c == code {
			n++
		}
	}

	return n
}
