package tests

import (
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// ownScope are the workspace routes that act on the caller's own account, not
// on the workspace they are looking at: any role may call them.
var ownScope = []string{"workspaces.store", "workspaces.switch", "workspace.leave"}

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

	before := map[string]int64{}
	for _, table := range []string{"transactions", "accounts", "categories", "budgets", "imports", "invites", "memberships"} {
		before[table] = rowCount(t, app, table)
	}

	reads, writes := 0, 0

	for _, route := range app.Router.Routes() {
		if !slices.Contains(route.Middleware, tenantMiddleware) || slices.Contains(ownScope, route.Name) {
			continue
		}

		target := pathParam.ReplaceAllString(route.Path, "1")

		t.Run(route.Name, func(t *testing.T) {
			if route.Method == http.MethodGet {
				reads++

				vera.Client.Get(target).AssertOK()

				return
			}

			writes++

			vera.Client.Do(route.Method, target, nil, "").AssertForbidden()
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
