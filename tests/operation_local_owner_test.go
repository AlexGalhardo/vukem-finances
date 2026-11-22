package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

// The local owner is the actor of the console of a personal installation: its
// owner, in its workspace. A saas server has none, and an installation with no
// owner has nobody to act as.
func TestLocalOwner(t *testing.T) {
	t.Parallel()

	var scopes []string
	for _, s := range api.Scopes() {
		scopes = append(scopes, s.Name)
	}

	t.Run("personal", func(t *testing.T) {
		t.Parallel()

		app := newPersonalApp(t)

		if _, err := operations.LocalOwner(t.Context(), service(t, app), operations.SurfaceCLI, scopes); !operations.IsCode(err, operations.CodeNotFound) {
			t.Fatalf("with no owner: %v, want not_found", err)
		}

		ana := signUp(t, app, "Ana", "ana@example.com")

		actor, err := operations.LocalOwner(t.Context(), service(t, app), operations.SurfaceCLI, scopes)
		if err != nil {
			t.Fatal(err)
		}

		if actor.User().ID != ana.User.ID || actor.WorkspaceID() != ana.Workspace.ID || actor.Role() != models.RoleOwner || actor.Surface() != operations.SurfaceCLI {
			t.Fatalf("actor = %+v, want Ana, her workspace, owner, cli", actor)
		}

		ws := service(t, app)

		fin, err := finance.New(app.App, ws)
		if err != nil {
			t.Fatal(err)
		}

		cat := operations.NewCatalog(operations.Config{Edition: ws.Edition}, operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)})

		res, err := operations.Invoke(t.Context(), cat.Accounts.List, actor, operations.NoInput{})
		if err != nil || len(res.Value.Data) == 0 {
			t.Fatalf("the local owner could not read its accounts: %v", err)
		}
	})

	t.Run("saas", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		signUp(t, app, "Ana", "ana@example.com")

		if _, err := operations.LocalOwner(t.Context(), service(t, app), operations.SurfaceCLI, scopes); !operations.IsCode(err, operations.CodeForbidden) {
			t.Fatalf("on saas: %v, want forbidden", err)
		}
	})
}
