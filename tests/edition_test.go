package tests

import (
	"maps"
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// newEditionApp builds the app in edition with extra environment values.
func newEditionApp(t *testing.T, edition string, env ...map[string]string) *vukemtest.App {
	t.Helper()

	opts := vukemtest.Options{Env: map[string]string{"APP_EDITION": edition}}
	for _, e := range env {
		maps.Copy(opts.Env, e)
	}

	return vukemtest.New(t, bootstrap.New, opts)
}

// newSaaSApp builds the multi-tenant edition: the one the finance suite covers.
func newSaaSApp(t *testing.T, env ...map[string]string) *vukemtest.App {
	t.Helper()

	return newEditionApp(t, "saas", env...)
}

// newPersonalApp builds the self-hosted edition.
func newPersonalApp(t *testing.T, env ...map[string]string) *vukemtest.App {
	t.Helper()

	return newEditionApp(t, "personal", env...)
}

func TestLimitsPersonalHasNoCaps(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	lim, err := service(t, app).Limits(workspaces.WithTenant(t.Context(), ana.Workspace.ID))
	if err != nil {
		t.Fatal(err)
	}

	if lim != workspaces.Personal || lim.Plan != "personal" {
		t.Fatalf("limits = %+v, want the personal plan", lim)
	}

	// What the Free plan refuses with a 402 to /billing is allowed.
	ana.Client.PostForm("/accounts", url.Values{"name": {"Savings"}, "kind": {"savings"}}).AssertRedirect("/accounts")
	groceries := categoryID(t, app, ana, "Groceries")
	ana.Client.PostForm("/budgets", url.Values{"categoryId": {strconv.FormatInt(groceries, 10)}, "limit": {"300"}}).
		AssertRedirect("/budgets")
	ana.Client.Inertia("").Get("/dashboard").AssertInertia("Dashboard").PropEquals("workspace.plan", "personal")
}
