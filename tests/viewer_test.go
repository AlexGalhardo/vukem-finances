package tests

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// ownScope are the workspace routes that act on the caller's own account, not
// on the workspace they are looking at: any role may call them (a viewer reads and
// settles their own notifications). A viewer may make
// an API token: the role limits what it can do at every request (TestAPIScopeMatrix).
// A viewer syncs read-only, so they also revoke their own devices (devices.destroy);
// the pushed rows are checked against the role one by one (TestSyncPushPermissions).
// Any member may also press Add device: the phone it pairs holds the same role ceiling (TestPairing_TokenHasTheRoleCeiling).
var ownScope = []string{"workspaces.store", "workspaces.switch", "workspace.leave", "tokens.store", "tokens.destroy",
	"notifications.read", "notifications.read_all", "notifications.preferences", "devices.destroy", "devices.pair",
	// The personal export and the account deletion are about the person, not the workspace: any member, a viewer too,
	// confirms their own password first (tests/privacy_*_test.go).
	"account.export.confirm", "account.export"}

// ownerOnlyReads are GET routes that are not open to a viewer: polling a PIX
// payment re-reads it at the provider and can activate it, so it is billing
// management, which only the owner does. The JSON export of the workspace holds
// the budgets, goals and member names, so it needs data.manage (owner, admin).
var ownerOnlyReads = []string{"billing.pix.show", "workspace.export"}

func viewerOf(t *testing.T, app *vukemtest.App, owner who) who {
	t.Helper()

	u := newVerifiedUser(t, app, "Vera", "vera@example.com")
	join(t, app, owner, u, models.RoleViewer)

	return who{u, owner.Workspace, browser(t, app, u, owner.Workspace.ID)}
}

func TestViewerIsReadOnly(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	vera := viewerOf(t, app, ana)
	addTransaction(t, ana, firstAccount(t, app, ana), 0, "expense", "11.00", "2026-01-10").AssertRedirect("/transactions")

	// A card, so the card page the viewer reads (/cards/1) exists.
	makePro(t, app, ana)
	ana.Client.PostForm("/cards", url.Values{
		"name": {"Visa"}, "limit": {"1000.00"}, "closingDay": {"10"}, "dueDay": {"20"},
	}).AssertRedirect("/cards")

	// The write limit is per person and the workspace has more write routes
	// than it allows in a minute, so viewers take turns: each route is still
	// tried by a viewer, and a 429 can never stand in for the 403.
	const writesPerViewer = 50

	viewers, attempts := []who{vera}, len(ownerOnlyReads)

	for _, route := range app.Router.Routes() {
		if route.Method != http.MethodGet && slices.Contains(route.Middleware, tenantMiddleware) && !slices.Contains(ownScope, route.Name) {
			attempts++
		}
	}

	for i := 1; i <= attempts/writesPerViewer; i++ {
		u := newVerifiedUser(t, app, "Vera", "vera"+strconv.Itoa(i)+"@example.com")
		join(t, app, ana, u, models.RoleViewer)
		viewers = append(viewers, who{u, ana.Workspace, browser(t, app, u, ana.Workspace.ID)})
	}

	before := map[string]int64{}
	for _, table := range []string{"transactions", "accounts", "categories", "budgets", "imports", "invites", "memberships", "credit_cards"} {
		before[table] = rowCount(t, app, table)
	}

	reads, writes := 0, 0

	for _, route := range app.Router.Routes() {
		if !slices.Contains(route.Middleware, tenantMiddleware) || slices.Contains(ownScope, route.Name) {
			continue
		}

		target := pathParam.ReplaceAllString(route.Path, "1")

		t.Run(route.Name, func(t *testing.T) {
			if route.Method == http.MethodGet && slices.Contains(ownerOnlyReads, route.Name) {
				writes++

				vera.Client.Get(target).AssertForbidden()

				return
			}

			if route.Method == http.MethodGet {
				reads++

				vera.Client.Get(target).AssertOK()

				return
			}

			viewers[writes/writesPerViewer].Client.Do(route.Method, target, nil, "").AssertForbidden()

			writes++
		})
	}

	if reads == 0 || writes == 0 {
		t.Fatalf("the viewer was checked on %d reads and %d writes", reads, writes)
	}

	for table, n := range before {
		if got := rowCount(t, app, table); got != n {
			t.Errorf("%s has %d rows after the viewer's calls, want %d", table, got, n)
		}
	}
}

func TestSecurity_ViewerCannotEscalate(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	vera := viewerOf(t, app, ana)

	// A viewer cannot invite, not even a more powerful role than their own.
	vera.Client.PostForm("/workspace/invites", url.Values{"email": {"eve@example.com"}, "role": {"admin"}}).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "invites", 0)

	// Nor remove the owner, nor rewrite the workspace.
	vera.Client.Delete("/workspace/members/" + idString(ana.User.ID)).AssertForbidden()
	putForm(vera.Client, "/workspace", url.Values{"name": {"Mine now"}, "currency": {"USD"}}).AssertForbidden()
	vukemtest.AssertDatabaseHas(t, app, "memberships", map[string]any{
		"workspace_id": ana.Workspace.ID, "user_id": ana.User.ID, "role": models.RoleOwner,
	})
	vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{"id": ana.Workspace.ID, "name": ana.Workspace.Name})

	// The owner may invite a viewer (the Free plan caps the members).
	makePro(t, app, ana)
	ana.Client.PostForm("/workspace/invites", url.Values{"email": {"vic@example.com"}, "role": {"viewer"}}).
		AssertRedirect("/workspace")
	vukemtest.AssertDatabaseHas(t, app, "invites", map[string]any{"email": "vic@example.com", "role": "viewer"})
}
