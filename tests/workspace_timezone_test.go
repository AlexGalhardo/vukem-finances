package tests

import (
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

func TestWorkspaceTimezoneValidation(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	save := func(tz string) *vukemtest.Response {
		return putForm(ana.Client, "/workspace", url.Values{"name": {"Home"}, "currency": {"USD"}, "timezone": {tz}})
	}

	vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{"id": ana.Workspace.ID, "timezone": "UTC"})

	for _, bad := range []string{"Mars/Base", "Local", "../etc/passwd", "utc+3"} {
		save(bad).AssertRedirect("/workspace")
		vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{"id": ana.Workspace.ID, "timezone": "UTC"})
	}

	save("America/Sao_Paulo").AssertRedirect("/workspace")
	vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{
		"id": ana.Workspace.ID, "name": "Home", "timezone": "America/Sao_Paulo",
	})

	// A form without the field leaves the zone alone.
	putForm(ana.Client, "/workspace", url.Values{"name": {"Home 2"}, "currency": {"USD"}}).AssertRedirect("/workspace")
	vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{
		"id": ana.Workspace.ID, "name": "Home 2", "timezone": "America/Sao_Paulo",
	})

	// A workspace of another user is not reached.
	bob := signUp(t, app, "Bob", "bob@example.com")
	vukemtest.AssertDatabaseHas(t, app, "workspaces", map[string]any{"id": bob.Workspace.ID, "timezone": "UTC"})
}
