package tests

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem/vukemtest"
)

// inventoryDoc is docs/qa/inventory.md, generated from the operations, the
// surfaces they are offered on and the roles.
const inventoryDoc = "../docs/qa/inventory.md"

// policy is the table the suite holds every surface to, written here and not
// read from app/workspaces: what each role may do with an operation by the
// ability it asks for.
var policy = map[string]map[string]bool{ //nolint:gochecknoglobals // a constant table
	models.RoleOwner:  {"workspace.read": true, "transactions.record": true, "data.manage": true},
	models.RoleAdmin:  {"workspace.read": true, "transactions.record": true, "data.manage": true},
	models.RoleMember: {"workspace.read": true, "transactions.record": true},
	models.RoleViewer: {"workspace.read": true},
}

// offeredOn lists the surfaces an operation of the inventory is offered on.
func offeredOn(e operations.InventoryEntry) []string {
	var out []string

	for _, s := range e.Surfaces {
		if s.Offered {
			out = append(out, s.Surface)
		}
	}

	return out
}

// roleUsers adds an admin, a member and a viewer to the workspace of the owner.
func roleUsers(t *testing.T, app *vukemtest.App, owner who) map[string]who {
	t.Helper()

	out := map[string]who{models.RoleOwner: owner}

	for _, role := range []string{models.RoleAdmin, models.RoleMember, models.RoleViewer} {
		u := newVerifiedUser(t, app, "Authz "+role, role+"@authz.example.com")
		join(t, app, owner, u, role)

		out[role] = who{User: u, Workspace: owner.Workspace}
	}

	return out
}

// newCatalog is the registry of the app, as an adapter builds it.
func newCatalog(t *testing.T, app *vukemtest.App) *operations.Catalog {
	t.Helper()

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	cache, err := app.App.Cache()
	if err != nil {
		t.Fatal(err)
	}

	return operations.NewCatalog(operations.Config{Edition: ws.Edition, Store: cache.Store(), Now: app.App.Now},
		operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)})
}

// scopesWithout is every scope of the API but one.
func scopesWithout(scope string) []string {
	var out []string

	for _, s := range api.Scopes() {
		if s.Name != scope {
			out = append(out, s.Name)
		}
	}

	return out
}

// TestSecurity_Authz_Matrix holds every operation, on every surface it is
// offered on, for each of the four roles, to the policy table: a role the table
// allows gets through, one it does not gets "forbidden", and a credential
// without the scope of the operation is refused as "missing_scope" whatever the
// role. REST runs over HTTP with a token of a real member; the CLI and MCP run
// the registry as their adapters call it (Invoke with an actor of the surface,
// the role the adapter read from the membership).
func TestSecurity_Authz_Matrix(t *testing.T) {
	ops := readInventory(t).Operations
	base := newAPIFixture(t)
	baseUsers := roleUsers(t, base.app, base.who)
	cat := newCatalog(t, base.app)
	cases := isolationCases(cat, theirs{})

	roles := []string{models.RoleOwner, models.RoleAdmin, models.RoleMember, models.RoleViewer}

	cliAndMCP := func(t *testing.T, op operations.InventoryEntry, surface operations.Surface, role string) {
		t.Helper()

		tc, ok := cases[op.Name]
		if !ok {
			t.Fatalf("no registry call for %s", op.Name)
		}

		as := func(scopes []string) operations.Actor {
			return operations.NewActor(surface, baseUsers[role].User, base.who.Workspace.ID, role, scopes, "authz")
		}

		_, err := tc.run(t.Context(), as(scopesWithout(op.Scope)))
		if !operations.IsCode(err, operations.CodeMissingScope) {
			t.Errorf("without the scope %s: err = %v, want missing_scope", op.Scope, err)
		}

		_, err = tc.run(t.Context(), as([]string{op.Scope}))
		if policy[role][op.Ability] {
			if operations.IsCode(err, operations.CodeForbidden) || operations.IsCode(err, operations.CodeMissingScope) {
				t.Errorf("a %s was refused (%v), the policy allows %s", role, err, op.Ability)
			}
		} else if !operations.IsCode(err, operations.CodeForbidden) {
			t.Errorf("a %s got err = %v, want forbidden (it needs %s)", role, err, op.Ability)
		}
	}

	// REST is the real thing: a token of a real member, over HTTP. Each operation
	// gets an app of its own, so the rows one case makes never limit another.
	rest := func(t *testing.T, op operations.InventoryEntry, f apiFixture, users map[string]who, role string) {
		t.Helper()

		build, ok := matrixCalls[op.Name]
		if !ok {
			t.Fatalf("no REST call for %s", op.Name)
		}

		name := strings.ReplaceAll(op.Name, ".", "-") + "-" + role
		call := build(seed(t, f, name), name)
		member := users[role]

		other := "summary:read"
		if op.Scope == other {
			other = "accounts:read"
		}

		without := apiClient(t, f.app, rawToken(t, f.app, member.User, member.Workspace.ID, other))
		assertProblem(t, send(without, call), http.StatusForbidden, "missing_scope")

		with := apiClient(t, f.app, rawToken(t, f.app, member.User, member.Workspace.ID, op.Scope))
		res := send(with, call)

		if policy[role][op.Ability] {
			if res.Status() < 200 || res.Status() > 299 {
				t.Errorf("a %s got %d, the policy allows %s: %s", role, res.Status(), op.Ability, res.Body())
			}
		} else {
			assertProblem(t, res, http.StatusForbidden, "forbidden")
		}
	}

	var ran atomic.Int64

	t.Run("cases", func(t *testing.T) {
		for _, op := range ops {
			t.Run(op.Name, func(t *testing.T) {
				t.Parallel()

				var f apiFixture

				var users map[string]who

				if slices.Contains(offeredOn(op), "rest") {
					f = newAPIFixture(t)
					users = roleUsers(t, f.app, f.who)
				}

				for _, surface := range offeredOn(op) {
					for _, role := range roles {
						t.Run(surface+"/"+role, func(t *testing.T) {
							ran.Add(1)

							switch surface {
							case "rest":
								rest(t, op, f, users, role)
							case "cli":
								cliAndMCP(t, op, operations.SurfaceCLI, role)
							case "mcp":
								cliAndMCP(t, op, operations.SurfaceMCP, role)
							default:
								t.Fatalf("surface %q has no authorization case", surface)
							}
						})
					}
				}
			})
		}
	})

	t.Logf("authorization cases: %d", ran.Load())

	doc, err := os.ReadFile(inventoryDoc)
	if os.IsNotExist(err) {
		// A project generated from the scaffold has no QA tooling and no inventory; this repository must have both.
		if _, statErr := os.Stat("../scripts/qa/inventory"); os.IsNotExist(statErr) {
			return
		}
	}

	if err != nil {
		t.Fatal(err)
	}

	m := regexp.MustCompile(`Authorization cases: (\d+)`).FindSubmatch(doc)
	if m == nil {
		t.Fatal("docs/qa/inventory.md has no authorization case count: run go run ./scripts/qa/inventory")
	}

	if want, _ := strconv.Atoi(string(m[1])); int(ran.Load()) != want {
		t.Errorf("ran %d authorization cases, docs/qa/inventory.md counts %d (operations x surfaces x roles)", ran.Load(), want)
	}
}

// TestSecurity_Authz_RemovedMemberLosesEveryCredential removes a member through
// the app and checks that each way the person reached the workspace stops at
// once: the API token, the token of a sync device, the access token and the
// refresh token of an MCP grant. A credential of another person is untouched.
func TestSecurity_Authz_RemovedMemberLosesEveryCredential(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	carl := s.SignUp(t, "Carl", "carl@example.com")
	dora := s.SignUp(t, "Dora", "dora@example.com")
	s.AddMember(t, ana.Workspace.ID, carl, models.RoleMember)
	s.AddMember(t, ana.Workspace.ID, dora, models.RoleMember)

	api := func(token string) *vukemtest.Client {
		return apiClient(t, s.App, token).WithHeader("X-Sync-Protocol", "1")
	}

	carlAPI := rawToken(t, s.App, carl.User, ana.Workspace.ID, "accounts:read")
	carlSync := rawToken(t, s.App, carl.User, ana.Workspace.ID, "sync")
	doraAPI := rawToken(t, s.App, dora.User, ana.Workspace.ID, "accounts:read")
	carlGrant := s.Authorize(t, carl, ana.Workspace.ID, "accounts:read")
	doraGrant := s.Authorize(t, dora, ana.Workspace.ID, "accounts:read")

	api(carlAPI).Get("/api/v1/accounts").AssertOK()
	api(carlSync).Get("/api/v1/sync/info").AssertOK()
	s.Call(t, carlGrant.Access, "tools/list", nil).AssertOK()

	s.Browser(t, ana, ana.Workspace.ID).Delete("/workspace/members/" + idString(carl.User.ID)).AssertRedirect("/workspace")

	api(carlAPI).Get("/api/v1/accounts").AssertUnauthorized()
	api(carlSync).Get("/api/v1/sync/info").AssertUnauthorized()
	s.Call(t, carlGrant.Access, "tools/list", nil).AssertUnauthorized()

	refresh := s.Token(t, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {carlGrant.Refresh}, "client_id": {oauthtest.ClientID},
		"resource": {oauthtest.Resource},
	})
	if refresh.Status() == http.StatusOK {
		t.Error("the refresh token of a removed member still refreshes")
	}

	// Another member of the same workspace keeps working.
	api(doraAPI).Get("/api/v1/accounts").AssertOK()
	s.Call(t, doraGrant.Access, "tools/list", nil).AssertOK()
}
