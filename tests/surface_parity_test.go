package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/cli"
	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
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

// cliGaps lists what is wrong between the registry and the CLI: an operation
// offered on the CLI with no command, with a command that is not its name, a
// resource no registered Vukem command serves, or an input field with no flag;
// and a command with no operation.
func cliGaps(reg *operations.Registry, cmds []cli.Command, registered []string) []string {
	var gaps []string

	byOp := map[string]cli.Command{}

	for _, c := range cmds {
		byOp[c.Op.Name] = c

		if m, ok := reg.Named(c.Op.Name); !ok || !m.Offered(operations.SurfaceCLI) {
			gaps = append(gaps, "the command "+c.Name()+" is not an operation the CLI offers")
		}
	}

	for _, m := range reg.Operations() {
		if !m.Offered(operations.SurfaceCLI) {
			continue
		}

		c, ok := byOp[m.Name]
		if !ok {
			gaps = append(gaps, m.Name+" has no CLI command")
			continue
		}

		resource, rest, _ := strings.Cut(m.Name, ".")
		if c.Resource != resource || c.Verb != strings.ReplaceAll(rest, ".", "-") {
			gaps = append(gaps, m.Name+" is the command "+c.Name()+", not `"+resource+" "+strings.ReplaceAll(rest, ".", "-")+"`")
		}

		if !slices.Contains(registered, c.Resource) {
			gaps = append(gaps, m.Name+": no registered command serves the resource "+c.Resource)
		}

		if m.Input.Kind() == reflect.Struct {
			for f := range m.Input.Fields() {
				name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
				if name == "-" || !f.IsExported() {
					continue
				}

				if !slices.ContainsFunc(c.Fields, func(cf cli.Field) bool { return cf.Name == name }) {
					gaps = append(gaps, m.Name+": the input field "+name+" has no flag")
				}
			}
		}
	}

	return gaps
}

// mcpGaps lists what is wrong between the registry and the MCP tools: an
// operation offered on MCP with no tool, or a tool that is no operation. An
// operation leaves MCP only through OptOut, with a reason.
func mcpGaps(reg *operations.Registry, tools []string) []string {
	var gaps []string

	offered := map[string]bool{}

	for _, m := range reg.Operations() {
		name := appmcp.ToolName(m.Name)

		if !m.Offered(operations.SurfaceMCP) {
			if slices.Contains(tools, name) {
				gaps = append(gaps, m.Name+" opts out of MCP and still has the tool "+name)
			}

			continue
		}

		offered[name] = true

		if !slices.Contains(tools, name) {
			gaps = append(gaps, m.Name+" has no MCP tool "+name)
		}
	}

	for _, tool := range tools {
		if !offered[tool] {
			gaps = append(gaps, "the MCP tool "+tool+" is not an operation MCP offers")
		}
	}

	return gaps
}

func registeredCLICommands() []string {
	var names []string
	for _, c := range bootstrap.EarlyCommands() {
		names = append(names, c.Name())
	}

	return names
}

// TestSurfaceParity: every operation the registry offers on REST has its route,
// and every route is an operation; every operation it offers on the CLI has its
// command. and every operation it offers on MCP has its tool. An operation leaves a surface only through OptOut, with a reason.
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

	tree, err := cli.DefaultTree()
	if err != nil {
		t.Fatal(err)
	}

	for _, gap := range cliGaps(reg, tree.Commands, registeredCLICommands()) {
		t.Error(gap)
	}

	tools, err := appmcp.ToolNames()
	if err != nil {
		t.Fatal(err)
	}

	for _, gap := range mcpGaps(reg, tools) {
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

// Removing the command of any operation, or the Vukem command of a resource,
// must fail the parity check.
func TestSurfaceParityFailsWithoutACLICommand(t *testing.T) {
	t.Parallel()

	reg := api.Describe().Registry

	tree, err := cli.DefaultTree()
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Commands) == 0 {
		t.Fatal("the CLI has no command")
	}

	registered := registeredCLICommands()

	for i, c := range tree.Commands {
		without := slices.Delete(slices.Clone(tree.Commands), i, i+1)

		gaps := cliGaps(reg, without, registered)
		if len(gaps) != 1 || !strings.Contains(gaps[0], c.Op.Name+" has no CLI command") {
			t.Errorf("without the command of %s the gaps are %v, want exactly that one", c.Op.Name, gaps)
		}
	}

	for _, resource := range tree.Resources() {
		without := slices.DeleteFunc(slices.Clone(registered), func(n string) bool { return n == resource })

		if gaps := cliGaps(reg, tree.Commands, without); len(gaps) != len(tree.Of(resource)) {
			t.Errorf("without the resource command %s the gaps are %v, want one per verb", resource, gaps)
		}
	}
}

// Removing the MCP tool of any operation must fail the parity check.
func TestSurfaceParityFailsWithoutAnMCPTool(t *testing.T) {
	t.Parallel()

	reg := api.Describe().Registry

	tools, err := appmcp.ToolNames()
	if err != nil || len(tools) == 0 {
		t.Fatalf("tools %v: %v", tools, err)
	}

	for i, tool := range tools {
		without := slices.Delete(slices.Clone(tools), i, i+1)

		gaps := mcpGaps(reg, without)
		if len(gaps) != 1 || !strings.Contains(gaps[0], "has no MCP tool "+tool) {
			t.Errorf("without the tool %s the gaps are %v, want exactly that one", tool, gaps)
		}
	}

	if gaps := mcpGaps(reg, append(slices.Clone(tools), "made_up_tool")); len(gaps) != 1 {
		t.Errorf("an extra tool gives the gaps %v, want one", gaps)
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
