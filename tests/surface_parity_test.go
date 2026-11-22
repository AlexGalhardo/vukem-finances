package tests

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// restOnly are the operations of /api/v1 that are not in the registry, each
// with the reason. They are not finance actions: one reads the token itself and
// the others are the device protocol of the mobile and desktop engines.
func restOnly(name string) (string, bool) {
	switch {
	case name == "openapi":
		return "it serves the document of the API, which describes the operations", true
	case name == "me.show":
		return "it describes the bearer token of the request, which only REST has", true
	case strings.HasPrefix(name, "sync."):
		return "the sync protocol is a device protocol with its own headers, limits and versions (docs/specs/sync-protocol.md)", true
	}

	return "", false
}

// surfaceGaps lists what is wrong between the registry and the routes of the
// REST surface: an operation with no route, a route with no operation.
func surfaceGaps(reg *operations.Registry, routes []router.RouteInfo) []string {
	var gaps []string

	byName := map[string]router.RouteInfo{}

	for _, r := range routes {
		if strings.HasPrefix(r.Path, api.BasePath+"/") && strings.HasPrefix(r.Name, "api.") {
			byName[strings.TrimPrefix(r.Name, "api.")] = r
		}
	}

	for _, m := range reg.Operations() {
		if !m.Offered(operations.SurfaceREST) {
			continue
		}

		row, declared := api.OperationNamed(m.Name)
		if !declared {
			gaps = append(gaps, m.Name+" has no row in the REST table")
			continue
		}

		route, served := byName[m.Name]

		switch {
		case !served:
			gaps = append(gaps, m.Name+" has no REST route")
		case route.Method != row.Method || route.Path != api.BasePath+row.Path:
			gaps = append(gaps, m.Name+" is served at "+route.Method+" "+route.Path+", not where its row says")
		}
	}

	for name := range byName {
		if _, ok := reg.Named(name); ok {
			continue
		}

		if _, ok := restOnly(name); !ok {
			gaps = append(gaps, "the REST route "+name+" is not an operation of the registry and not a declared REST-only one")
		}
	}

	return gaps
}

// TestSurfaceParity: every operation the registry offers on REST has its route,
// and every route is an operation. Epics 31 and 32 add the CLI and MCP surfaces
// to this test; an operation leaves a surface only through OptOut, with a reason.
func TestSurfaceParity(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	reg := api.Describe().Registry

	if len(reg.Operations()) == 0 {
		t.Fatal("the registry is empty")
	}

	for _, gap := range surfaceGaps(reg, app.Router.Routes()) {
		t.Error(gap)
	}

	for _, m := range reg.Operations() {
		for s, why := range m.OptOut {
			if strings.TrimSpace(why) == "" {
				t.Errorf("%s leaves %s with no reason", m.Name, s)
			}
		}
	}
}

// Removing the REST route of any operation must fail the parity check.
func TestSurfaceParityFailsWithoutARoute(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	reg := api.Describe().Registry
	routes := app.Router.Routes()

	for _, m := range reg.Operations() {
		without := slices.DeleteFunc(slices.Clone(routes), func(r router.RouteInfo) bool { return r.Name == "api."+m.Name })

		if len(without) == len(routes) {
			t.Errorf("%s has no route to remove", m.Name)
			continue
		}

		gaps := surfaceGaps(reg, without)
		if len(gaps) != 1 || !strings.Contains(gaps[0], m.Name+" has no REST route") {
			t.Errorf("without the route of %s the gaps are %v, want exactly that one", m.Name, gaps)
		}
	}
}

// The handlers of /api/v1 never call a finance service: every action goes
// through operations.Invoke (epic 30, acceptance 5).
func TestAPIHandlersGoThroughInvoke(t *testing.T) {
	t.Parallel()

	forbidden := []string{"vukem-finances/app/finance\"", "vukem-finances/app/reports\"", "vukem-finances/app/export\"", "a.Fin.", ".Fin."}

	files, err := filepath.Glob("../app/api/*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no handler files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		for _, bad := range forbidden {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s uses %q: a handler of /api/v1 must go through operations.Invoke", file, bad)
			}
		}
	}
}
