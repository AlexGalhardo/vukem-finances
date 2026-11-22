package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// markPending puts the workspace in its grace period the way the request does.
func markPending(t *testing.T, app *vukemtest.App, w who) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(),
		"UPDATE workspaces SET state = 'pending_deletion', purge_at = datetime('now', '+30 days') WHERE id = ?", w.Workspace.ID); err != nil {
		t.Fatal(err)
	}
}

func readInventory(t *testing.T) operations.Inventory {
	t.Helper()

	raw, err := os.ReadFile(inventoryFile)
	if err != nil {
		t.Fatal(err)
	}

	var inv operations.Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatal(err)
	}

	return inv
}

// TestPendingDeletionRefusesWrites: while a workspace waits to be purged, every
// write of resources/operations.json is refused on REST, the CLI and MCP (all of
// them through Invoke), and so are the web forms, the Telegram bot and sync.
func TestPendingDeletionRefusesWrites(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	inv := readInventory(t)

	// Rows to point the REST calls at, made while the workspace is still active.
	rows := map[string]seeded{}

	for _, e := range inv.Operations {
		if e.Kind != string(operations.KindRead) {
			rows[e.Name] = seed(t, f, strings.ReplaceAll(e.Name, ".", "-"))
		}
	}

	phone := newSyncDevice(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "sync"), "phone")

	markPending(t, f.app, f.who)

	// Sequential on purpose: the last one cancels the deletion.
	t.Run("REST", func(t *testing.T) {
		c := apiClient(t, f.app, f.token)
		writes := 0

		for _, e := range inv.Operations {
			if e.Kind == string(operations.KindRead) {
				continue
			}

			build, ok := matrixCalls[e.Name]
			if !ok {
				t.Errorf("operation %s has no REST call in matrixCalls", e.Name)
				continue
			}

			writes++

			call := build(rows[e.Name], strings.ReplaceAll(e.Name, ".", "-"))
			if call.method == http.MethodGet {
				t.Errorf("%s is a %s but its call is a GET", e.Name, e.Kind)
				continue
			}

			res := send(c, call)
			if res.Status() != http.StatusLocked {
				t.Errorf("%s: status %d, want 423: %s", e.Name, res.Status(), res.Body())
				continue
			}

			if code, _ := res.JSONPath("code"); code != "workspace_pending_deletion" {
				t.Errorf("%s: code %v", e.Name, code)
			}
		}

		if writes == 0 {
			t.Fatal("the inventory has no write operation")
		}

		// Reads still work: the data stays readable until the purge.
		c.Get("/api/v1/accounts").AssertOK()
	})

	t.Run("every surface through Invoke", func(t *testing.T) {
		ws, err := workspaces.New(f.app.App)
		if err != nil {
			t.Fatal(err)
		}

		fin, err := finance.New(f.app.App, ws)
		if err != nil {
			t.Fatal(err)
		}

		cat := operations.NewCatalog(operations.Config{Edition: ws.Edition, Now: f.app.App.Now},
			operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)})

		cases := isolationCases(cat, theirs{})

		for _, surface := range operations.Surfaces() {
			actor := operations.NewActor(surface, f.who.User, f.who.Workspace.ID, models.RoleOwner, allScopes(), "")

			for _, e := range inv.Operations {
				tc, ok := cases[e.Name]
				if !ok {
					t.Errorf("operation %s has no case", e.Name)
					continue
				}

				_, err := tc.run(context.Background(), actor)

				if e.Kind == string(operations.KindRead) {
					if operations.IsCode(err, operations.CodePendingDeletion) {
						t.Errorf("%s (%s): a read was refused", e.Name, surface)
					}

					continue
				}

				if !operations.IsCode(err, operations.CodePendingDeletion) {
					t.Errorf("%s (%s): err = %v, want %s", e.Name, surface, err, operations.CodePendingDeletion)
				}
			}
		}
	})

	t.Run("sync", func(t *testing.T) {
		for _, build := range syncMatrixCalls {
			assertProblem(t, send(phone.client(), build("pending")), http.StatusGone, "workspace_deleted")
		}
	})

	t.Run("web", func(t *testing.T) {
		c := f.who.Client

		for _, w := range []struct{ method, path string }{
			{"POST", "/accounts"}, {"POST", "/transactions"}, {"POST", "/categories"}, {"PUT", "/workspace"},
			{"POST", "/workspace/invites"}, {"POST", "/tokens"}, {"DELETE", "/tokens/x"}, {"POST", "/budgets"},
		} {
			res := c.Do(w.method, w.path, strings.NewReader(url.Values{"name": {"x"}}.Encode()), "application/x-www-form-urlencoded")
			if res.Status() != http.StatusLocked {
				t.Errorf("%s %s: status %d, want 423", w.method, w.path, res.Status())
			}
		}

		c.Get("/dashboard").AssertOK()
		c.Get("/transactions").AssertOK()

		// The way out stays open: the owner cancels (403 or 303, never 423).
		if res := c.PostForm("/workspace/deletion/cancel", url.Values{}); res.Status() == http.StatusLocked {
			t.Error("cancelling the deletion was refused")
		}
	})
}

func TestPendingDeletionRefusesTheBot(t *testing.T) {
	t.Parallel()

	app, fake, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9001)

	// A draft made before the request must not be confirmable during the grace period.
	say(t, bot, 9001, "-5,00 cafe")
	groceries := buttonData(t, fake, 9001, "Groceries")

	markPending(t, app, w)

	tap(t, bot, 9001, groceries)

	if got := len(transactionsOf(t, app, w)); got != 0 {
		t.Fatalf("%d transactions saved by confirming a draft in a pending workspace", got)
	}

	say(t, bot, 9001, "-12,50 almoco")
	mustContain(t, lastText(t, fake, 9001), "read only")

	if got := len(transactionsOf(t, app, w)); got != 0 {
		t.Fatalf("%d transactions saved in a pending workspace", got)
	}
}
