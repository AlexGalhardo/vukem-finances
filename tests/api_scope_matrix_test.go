package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// rawToken issues a token straight from the token store (no plan of the page:
// the matrix needs many), bound to the workspace.
func rawToken(t *testing.T, app *vukemtest.App, user models.User, workspace int64, scopes ...string) string {
	t.Helper()

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	plain, _, err := acc.Tokens.CreateWith(t.Context(), strconv.FormatInt(user.ID, 10), "matrix", scopes, time.Hour,
		auth.CreateOptions{Resource: strconv.FormatInt(workspace, 10)})
	if err != nil {
		t.Fatal(err)
	}

	return plain
}

// matrixCall is a valid request of one operation, over rows made for it.
type matrixCall struct {
	method, path string
	body         map[string]any
}

// seeded are the rows a call may point at, made fresh for every operation.
type seeded struct {
	account, account2, category, expenseCategory, tag, transaction, transfer, budget string
}

func seed(t *testing.T, f apiFixture, name string) seeded {
	t.Helper()

	c := apiClient(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, allScopes()...))
	var s seeded

	s.account = uidOf(t, c.PostJSON("/api/v1/accounts", map[string]any{"name": "A " + name, "kind": "cash"}).AssertCreated(), "uid")
	s.account2 = uidOf(t, c.PostJSON("/api/v1/accounts", map[string]any{"name": "B " + name, "kind": "cash"}).AssertCreated(), "uid")
	s.category = uidOf(t, c.PostJSON("/api/v1/categories", map[string]any{"name": "C " + name, "kind": "expense"}).AssertCreated(), "uid")
	s.expenseCategory = f.food.UID
	s.tag = uidOf(t, c.PostJSON("/api/v1/tags", map[string]any{"name": "t-" + name}).AssertCreated(), "uid")
	s.transaction = uidOf(t, c.PostJSON("/api/v1/transactions", map[string]any{
		"account_uid": s.account, "kind": "expense", "amount_cents": 100, "date": "2026-11-20",
	}).AssertCreated(), "uid")

	tr := c.PostJSON("/api/v1/transfers", map[string]any{
		"from_account_uid": s.account, "to_account_uid": s.account2, "amount_cents": 100, "date": "2026-11-20",
	}).AssertCreated()
	s.transfer = uidOf(t, tr, "uid")
	s.budget = uidOf(t, c.PostJSON("/api/v1/budgets", map[string]any{"category_uid": s.expenseCategory, "limit_cents": 100}).AssertOK(), "uid")

	return s
}

// matrixCalls has one entry per operation of the API: a new operation without
// one fails TestAPIScopeMatrix.
var matrixCalls = map[string]func(s seeded, name string) matrixCall{ //nolint:gochecknoglobals // a table
	"me.show":       func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/me"} },
	"accounts.list": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/accounts"} },
	"accounts.show": func(s seeded, _ string) matrixCall { return matrixCall{method: "GET", path: "/accounts/" + s.account} },
	"accounts.create": func(_ seeded, n string) matrixCall {
		return matrixCall{"POST", "/accounts", map[string]any{"name": "N " + n, "kind": "cash"}}
	},
	"accounts.update": func(s seeded, _ string) matrixCall {
		return matrixCall{"PATCH", "/accounts/" + s.account, map[string]any{"name": "Renamed"}}
	},
	"accounts.archive": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "POST", path: "/accounts/" + s.account2 + "/archive"}
	},

	"categories.list": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/categories"} },
	"categories.show": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "GET", path: "/categories/" + s.category}
	},
	"categories.create": func(_ seeded, n string) matrixCall {
		return matrixCall{"POST", "/categories", map[string]any{"name": "N " + n, "kind": "expense"}}
	},
	"categories.update": func(s seeded, _ string) matrixCall {
		return matrixCall{"PATCH", "/categories/" + s.category, map[string]any{"color": "blue"}}
	},
	"categories.archive": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "POST", path: "/categories/" + s.category + "/archive"}
	},

	"tags.list": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/tags"} },
	"tags.create": func(_ seeded, n string) matrixCall {
		return matrixCall{"POST", "/tags", map[string]any{"name": "n-" + n}}
	},
	"tags.update": func(s seeded, _ string) matrixCall {
		return matrixCall{"PATCH", "/tags/" + s.tag, map[string]any{"color": "red"}}
	},
	"tags.delete": func(s seeded, _ string) matrixCall { return matrixCall{method: "DELETE", path: "/tags/" + s.tag} },

	"transactions.list": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/transactions"} },
	"transactions.show": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "GET", path: "/transactions/" + s.transaction}
	},
	"transactions.create": func(s seeded, _ string) matrixCall {
		return matrixCall{"POST", "/transactions", map[string]any{"account_uid": s.account, "kind": "expense", "amount_cents": 5, "date": "2026-11-20"}}
	},
	"transactions.update": func(s seeded, _ string) matrixCall {
		return matrixCall{"PATCH", "/transactions/" + s.transaction, map[string]any{"amount_cents": 7}}
	},
	"transactions.delete": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "DELETE", path: "/transactions/" + s.transaction}
	},

	"transfers.create": func(s seeded, _ string) matrixCall {
		return matrixCall{"POST", "/transfers", map[string]any{"from_account_uid": s.account, "to_account_uid": s.account2, "amount_cents": 5, "date": "2026-11-20"}}
	},
	"transfers.delete": func(s seeded, _ string) matrixCall {
		return matrixCall{method: "DELETE", path: "/transfers/" + s.transfer}
	},

	"budgets.list": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/budgets"} },
	"budgets.set": func(s seeded, _ string) matrixCall {
		return matrixCall{"POST", "/budgets", map[string]any{"category_uid": s.expenseCategory, "limit_cents": 200}}
	},
	"budgets.delete": func(s seeded, _ string) matrixCall { return matrixCall{method: "DELETE", path: "/budgets/" + s.budget} },

	"summary.show": func(seeded, string) matrixCall { return matrixCall{method: "GET", path: "/summary"} },
}

// syncMatrixCalls has one entry per operation of the sync protocol. They need
// the protocol headers and a registered device, and a viewer's push is refused
// per mutation in a 200, so checkSyncOperation holds them to that instead.
var syncMatrixCalls = map[string]func(name string) matrixCall{ //nolint:gochecknoglobals // a table
	"sync.info": func(string) matrixCall { return matrixCall{method: "GET", path: "/sync/info"} },
	"sync.devices.create": func(n string) matrixCall {
		return matrixCall{"POST", "/sync/devices", map[string]any{"name": "D " + n, "platform": "ios"}}
	},
	"sync.devices.list": func(string) matrixCall { return matrixCall{method: "GET", path: "/sync/devices"} },
	"sync.pull":         func(string) matrixCall { return matrixCall{method: "GET", path: "/sync/pull?cursor=0&limit=10"} },
	"sync.snapshot":     func(string) matrixCall { return matrixCall{method: "GET", path: "/sync/snapshot?entity=account"} },
	"sync.push": func(n string) matrixCall {
		m := mut("account", newULID(), "upsert", 0, map[string]any{"name": "S " + n + " " + newULID(), "kind": "cash"})

		return matrixCall{"POST", "/sync/push", map[string]any{"mutations": []map[string]any{m}}}
	},
}

// checkSyncOperation holds one sync operation to the scope and to the role: no
// scope is a 403, a viewer reads and registers a device but every mutation of
// theirs is refused, and the owner's is applied.
func checkSyncOperation(t *testing.T, f apiFixture, vera who, id string) {
	t.Helper()

	build := syncMatrixCalls[id]
	name := strings.ReplaceAll(id, ".", "-")

	without := apiClient(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "summary:read")).
		WithHeader("X-Sync-Protocol", "1")
	assertProblem(t, send(without, build(name)), http.StatusForbidden, "missing_scope")

	pushStatus := func(c *vukemtest.Client) (int, pushResult) {
		res := send(c, build(name))

		var out struct {
			Results []pushResult `json:"results"`
		}

		if id == "sync.push" {
			if err := json.Unmarshal([]byte(res.Body()), &out); err != nil || len(out.Results) != 1 {
				t.Fatalf("push answer: %v: %s", err, res.Body())
			}

			return res.Status(), out.Results[0]
		}

		return res.Status(), pushResult{}
	}

	viewer := newSyncDevice(t, f.app, rawToken(t, f.app, vera.User, vera.Workspace.ID, "sync"), "viewer "+name)

	status, got := pushStatus(viewer.client())
	if status < 200 || status > 299 {
		t.Errorf("a viewer got %d", status)
	}

	if id == "sync.push" && (got.Status != "rejected" || got.code() != "forbidden") {
		t.Errorf("a viewer's mutation = %+v, want it refused as forbidden", got)
	}

	owner := newSyncDevice(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "sync"), "owner "+name)

	status, got = pushStatus(owner.client())
	if status < 200 || status > 299 {
		t.Errorf("the owner got %d", status)
	}

	if id == "sync.push" && got.Status != "applied" {
		t.Errorf("the owner's mutation = %+v, want it applied", got)
	}
}

func send(c *vukemtest.Client, call matrixCall) *vukemtest.Response {
	target := api.BasePath + call.path

	switch call.method {
	case http.MethodGet:
		return c.Get(target)
	case http.MethodDelete:
		return c.Delete(target)
	case http.MethodPatch:
		return c.PatchJSON(target, call.body)
	default:
		return c.PostJSON(target, call.body)
	}
}

func TestAPIScopeMatrix(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	vera := viewerOf(t, f.app, f.who)

	// The operations come from the committed document, so a new one with no case fails here.
	raw, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Paths map[string]map[string]struct {
			ID    string `json:"operationId"`
			Scope string `json:"x-required-scope"`
		} `json:"paths"`
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}

	tested, inDoc := 0, 0

	for _, methods := range doc.Paths {
		for _, op := range methods {
			inDoc++

			if _, isSync := syncMatrixCalls[op.ID]; isSync {
				if op.Scope != "sync" {
					t.Errorf("operation %s requires scope %q, want sync", op.ID, op.Scope)
				}

				t.Run(op.ID, func(t *testing.T) { checkSyncOperation(t, f, vera, op.ID) })

				tested++

				continue
			}

			build, ok := matrixCalls[op.ID]
			if !ok {
				t.Errorf("operation %s has no case in matrixCalls", op.ID)
				continue
			}

			t.Run(op.ID, func(t *testing.T) {
				name := strings.ReplaceAll(op.ID, ".", "-")
				rows := seed(t, f, name)
				call := build(rows, name)
				read := call.method == http.MethodGet

				// A token that holds some other scope.
				other := "summary:read"
				if op.Scope == other {
					other = "accounts:read"
				}

				without := apiClient(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, other))

				if op.Scope == "" {
					if s := send(without, call).Status(); s != http.StatusOK {
						t.Errorf("a token with any scope got %d on a scopeless operation", s)
					}
				} else {
					assertProblem(t, send(without, call), http.StatusForbidden, "missing_scope")
				}

				// A viewer's token holds the scope, and the role is the ceiling: reads pass, writes do not.
				scopes := []string{other}
				if op.Scope != "" {
					scopes = []string{op.Scope}
				}

				viewerClient := apiClient(t, f.app, rawToken(t, f.app, vera.User, vera.Workspace.ID, scopes...))

				if read {
					if s := send(viewerClient, call).Status(); s != http.StatusOK {
						t.Errorf("a viewer's read got %d, want 200", s)
					}
				} else {
					assertProblem(t, send(viewerClient, call), http.StatusForbidden, "forbidden")
				}

				// The owner's token with the scope passes.
				with := apiClient(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, scopes...))
				if res := send(with, call); res.Status() < 200 || res.Status() > 299 {
					t.Errorf("a token with %q got %d: %s", op.Scope, res.Status(), res.Body())
				}
			})

			tested++
		}
	}

	if cases := len(matrixCalls) + len(syncMatrixCalls); tested != inDoc || tested != len(api.Operations()) || tested != cases {
		t.Errorf("tested %d operations, the document has %d, the table %d, the cases %d",
			tested, inDoc, len(api.Operations()), cases)
	}
}
