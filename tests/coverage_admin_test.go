package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// TestAdminPanelIsReadOnly walks every route of the support panel: an admin
// reads the dashboard, the lists and one row, and every form that would
// create, change or delete is refused (the policies grant ViewAny and View only).
func TestAdminPanelIsReadOnly(t *testing.T) {
	t.Parallel()

	app := newAdminApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	admin := adminClient(t, app, adminEmail)
	other := adminClient(t, app, "user@example.com")

	admin.Inertia("").Get("/admin").AssertInertia("Admin/Dashboard")
	admin.Inertia("").Get("/admin/users").AssertInertia("Admin/ResourceList")
	admin.Inertia("").Get("/admin/users/" + idString(ana.User.ID)).AssertInertia("Admin/ResourceShow")
	admin.Inertia("").Get("/admin/workspaces/" + idString(ana.Workspace.ID)).AssertInertia("Admin/ResourceShow")

	admin.Get("/admin-api").AssertOK()
	admin.Get("/admin-api/_assets/admin.css").AssertOK()
	admin.Get("/admin-api/_assets/htmx.min.js").AssertOK()
	admin.Get("/admin-api/workspaces/" + idString(ana.Workspace.ID)).AssertOK()
	admin.Get("/admin-api/workspace-subscriptions/1").AssertOK()

	for _, rt := range []string{
		"GET /admin-api/users/create",
		"POST /admin-api/users",
		"GET /admin-api/users/" + idString(ana.User.ID) + "/edit",
		"POST /admin-api/users/" + idString(ana.User.ID),
		"GET /admin-api/users/" + idString(ana.User.ID) + "/delete",
		"GET /admin-api/workspaces/create",
		"POST /admin-api/workspaces",
		"GET /admin-api/workspaces/" + idString(ana.Workspace.ID) + "/edit",
		"POST /admin-api/workspaces/" + idString(ana.Workspace.ID),
		"GET /admin-api/workspaces/" + idString(ana.Workspace.ID) + "/delete",
		"POST /admin-api/workspaces/" + idString(ana.Workspace.ID) + "/delete",
		"GET /admin-api/workspace-subscriptions/create",
		"POST /admin-api/workspace-subscriptions",
		"GET /admin-api/workspace-subscriptions/1/edit",
		"POST /admin-api/workspace-subscriptions/1",
		"GET /admin-api/workspace-subscriptions/1/delete",
		"POST /admin-api/workspace-subscriptions/1/delete",
	} {
		method, path, _ := strings.Cut(rt, " ")

		for name, cl := range map[string]*vukemtest.Client{"admin": admin, "signed-in user": other} {
			var res *vukemtest.Response

			if method == http.MethodGet {
				res = cl.Get(path)
			} else {
				res = cl.PostForm(path, url.Values{"name": {"Changed"}})
			}

			if res.Status() != http.StatusForbidden {
				t.Errorf("%s %s as %s = %d, want 403", method, path, name, res.Status())
			}
		}
	}

	if n := rowCountWhere(t, app, "workspaces", "name = 'Changed'"); n != 0 {
		t.Errorf("a refused form renamed %d workspaces", n)
	}

	if n := rowCount(t, app, "users"); n != 3 {
		t.Errorf("users = %d, want the 3 that were made", n)
	}
}
