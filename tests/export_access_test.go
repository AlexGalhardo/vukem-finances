package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// joined is the browser of u, already a member of the workspace of owner.
func joined(t *testing.T, app *vukemtest.App, owner who, u models.User) who {
	t.Helper()

	return who{u, owner.Workspace, browser(t, app, u, owner.Workspace.ID)}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}

	return false
}

func TestExportPermissions(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	record(t, app, ana, acc, 0, "expense", "3.00", "2026-01-02", "shared row")

	for _, role := range []string{models.RoleOwner, models.RoleAdmin, models.RoleMember, models.RoleViewer} {
		actor := ana

		if role != models.RoleOwner {
			u := newVerifiedUser(t, app, "User "+role, role+"@example.com")
			join(t, app, ana, u, role)

			actor = joined(t, app, ana, u)
		}

		actor.Client.Get("/transactions/export.csv").AssertOK().AssertSee("shared row")

		json := actor.Client.Get("/workspace/export.json")
		if role == models.RoleOwner || role == models.RoleAdmin {
			json.AssertOK()
		} else {
			json.AssertForbidden()

			if containsAny(json.Body(), "shared row", `"format"`) {
				t.Errorf("%s: the refusal carries data", role)
			}
		}
	}

	t.Run("a signed out visitor gets neither", func(t *testing.T) {
		anon := app.Client(t)
		for _, path := range []string{"/transactions/export.csv", "/workspace/export.json"} {
			if got := anon.Get(path).Status(); got != http.StatusFound && got != http.StatusSeeOther && got != http.StatusUnauthorized {
				t.Errorf("%s answered %d to a signed out visitor", path, got)
			}
		}
	})
}

func TestExportRateLimited(t *testing.T) {
	t.Parallel()

	app, ana, _ := proWorkspace(t)
	viewer := newVerifiedUser(t, app, "Vera", "vera@example.com")
	join(t, app, ana, viewer, models.RoleViewer)
	vera := joined(t, app, ana, viewer)

	// A viewer refused the JSON spends nothing of the workspace's downloads.
	for range config.ExportLimit + 3 {
		vera.Client.Get("/workspace/export.json").AssertForbidden()
	}

	for i := range config.ExportLimit {
		path := "/transactions/export.csv"
		if i%2 == 1 {
			path = "/workspace/export.json"
		}

		ana.Client.Get(path).AssertOK()
	}

	for _, path := range []string{"/transactions/export.csv", "/workspace/export.json"} {
		res := ana.Client.Get(path).AssertStatus(http.StatusTooManyRequests)
		if res.Header("Retry-After") == "" {
			t.Errorf("%s: the 429 has no Retry-After", path)
		}
	}

	// The limit is per workspace: the viewer of the same one shares it, another workspace does not.
	vera.Client.Get("/transactions/export.csv").AssertStatus(http.StatusTooManyRequests)

	bob := signUp(t, app, "Bob", "bob@example.com")
	bob.Client.Get("/transactions/export.csv").AssertOK()
}

func TestCrossTenantMatrixExports(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	record(t, app, ana, acc, 0, "expense", "3.00", "2026-01-02", "ANA PRIVATE")

	bob := signUp(t, app, "Bob", "bob@example.com")

	// The routes take no id: the workspace is the session's. A session that names a workspace its
	// user is not in falls back to the user's own (the tenant middleware), so Bob can never reach
	// Ana's data by naming her workspace.
	intruder := browser(t, app, bob.User, ana.Workspace.ID)

	for _, path := range []string{"/transactions/export.csv", "/workspace/export.json"} {
		res := intruder.Get(path).AssertOK()
		if containsAny(res.Body(), "ANA PRIVATE", ana.Workspace.Name) {
			t.Errorf("%s: Bob received data of Ana's workspace", path)
		}
	}

	// What Bob does export is his own workspace, never Ana's.
	bob.Client.Get("/transactions/export.csv").AssertOK().AssertDontSee("ANA PRIVATE")
	bob.Client.Get("/workspace/export.json").AssertOK().AssertDontSee("ANA PRIVATE")
}
