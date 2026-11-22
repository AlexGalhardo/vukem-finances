package tests

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/schema"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

// lifecycleApp is the saas edition with sessions in the database, the way the
// real server runs, so a session row is part of what deletion must remove.
func lifecycleApp(t *testing.T) *oauthtest.Stack {
	t.Helper()

	return oauthtest.New(t, map[string]string{"SESSION_DRIVER": "database"})
}

func asWho(p oauthtest.Person) who { return who{User: p.User, Workspace: p.Workspace} }

// seedWorkspaceData makes one row of everything a person keeps in a workspace,
// all carrying marker, through the public API, and trashes one transaction so a
// soft deleted row exists too.
func seedWorkspaceData(t *testing.T, app *vukemtest.App, w who, marker string) {
	t.Helper()

	makePro(t, app, w)

	c := apiClient(t, app, rawToken(t, app, w.User, w.Workspace.ID, allScopes()...))

	account := uidOf(t, c.PostJSON("/api/v1/accounts", map[string]any{"name": marker + "-account", "kind": "cash"}).AssertCreated(), "uid")
	category := uidOf(t, c.PostJSON("/api/v1/categories", map[string]any{"name": marker + "-category", "kind": "expense"}).AssertCreated(), "uid")
	c.PostJSON("/api/v1/tags", map[string]any{"name": marker + "-tag"}).AssertCreated()

	for i, desc := range []string{marker + "-lunch", marker + "-trashed"} {
		uid := uidOf(t, c.PostJSON("/api/v1/transactions", map[string]any{
			"account_uid": account, "category_uid": category, "kind": "expense", "amount_cents": 500 + i, "date": "2026-11-20",
			"description": desc, "tags": []string{marker + "-tag"},
		}).AssertCreated(), "uid")

		if i == 1 {
			c.Delete("/api/v1/transactions/" + uid).AssertStatus(204)
		}
	}

	c.PostJSON("/api/v1/budgets", map[string]any{"category_uid": category, "limit_cents": 9000}).AssertOK()
	goal := uidOf(t, c.PostJSON("/api/v1/goals", map[string]any{"name": marker + "-goal", "target_cents": 100000}).AssertCreated(), "uid")
	c.PostJSON("/api/v1/goals/"+goal+"/contributions", map[string]any{"amount_cents": 100, "note": marker + "-note"}).AssertCreated()
	c.PostJSON("/api/v1/recurrences", map[string]any{
		"account_uid": account, "kind": "expense", "amount_cents": 100, "frequency": "monthly", "interval": 1,
		"start_date": "2099-01-01", "description": marker + "-rule",
	}).AssertCreated()
	c.PostJSON("/api/v1/cards", map[string]any{"name": marker + "-card", "limit_cents": 100000, "closing_day": 10, "due_day": 20}).AssertCreated()

	newSyncDevice(t, app, rawToken(t, app, w.User, w.Workspace.ID, "sync"), marker+"-phone")
}

// tableColumns lists every table with its columns, from the schema of the app.
func tableColumns(t *testing.T, app *vukemtest.App) map[string][]string {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	out := map[string][]string{}

	for _, tb := range snap.Tables {
		for _, c := range tb.Columns {
			out[tb.Name] = append(out[tb.Name], c.Name)
		}
	}

	return out
}

// rowsHolding returns "table.column" for every column of every table that holds
// needle in any row, soft deleted rows included.
func rowsHolding(t *testing.T, app *vukemtest.App, needle string) []string {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var found []string

	for table, cols := range tableColumns(t, app) {
		if table == "migrations" {
			continue
		}

		for _, col := range cols {
			var n int

			q := fmt.Sprintf("SELECT COUNT(*) FROM %q WHERE instr(CAST(%q AS TEXT), ?) > 0", table, col)
			if err := db.SQL().QueryRowContext(t.Context(), q, needle).Scan(&n); err != nil {
				t.Fatalf("scanning %s.%s: %v", table, col, err)
			}

			if n > 0 {
				found = append(found, table+"."+col)
			}
		}
	}

	slices.Sort(found)

	return found
}

func lifecycleCount(t *testing.T, app *vukemtest.App, query string, args ...any) int {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.SQL().QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

// workspaceRowCounts counts the rows of every table that has a workspace_id.
func workspaceRowCounts(t *testing.T, app *vukemtest.App, workspaceID int64) map[string]int {
	t.Helper()

	out := map[string]int{}

	for table, cols := range tableColumns(t, app) {
		if slices.Contains(cols, "workspace_id") {
			out[table] = lifecycleCount(t, app, fmt.Sprintf("SELECT COUNT(*) FROM %q WHERE workspace_id = ?", table), workspaceID)
		}
	}

	return out
}

// dueNow moves the purge date of a row into the past: "table SET purge_at = ?
// WHERE id = ?" with the id as the argument.
func dueNow(t *testing.T, app *vukemtest.App, update string, id int64) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(t.Context(), "UPDATE "+update, time.Now().UTC().Add(-time.Hour), id); err != nil {
		t.Fatal(err)
	}
}

// endSubscriptions marks the Pro subscription the seed made as already
// cancelled: the purge calls Stripe for a live one, and these tests hold no Stripe.
func endSubscriptions(t *testing.T, app *vukemtest.App) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(t.Context(), "UPDATE workspace_subscriptions SET status = 'canceled'"); err != nil {
		t.Fatal(err)
	}
}

func runPurge(t *testing.T, app *vukemtest.App) {
	t.Helper()

	for range 2 { // idempotent
		if err := jobs.HandlePrivacyPurge(app.App)(t.Context(), jobs.PrivacyPurge{}); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDataLifecycle_ExportHoldsEverythingTheyOwnAndNoSecret is the export of a
// person who owns a workspace with data in every area: every marker is in it,
// no secret is.
func TestDataLifecycle_ExportHoldsEverythingTheyOwnAndNoSecret(t *testing.T) {
	t.Parallel()

	s := lifecycleApp(t)
	ana := s.SignUp(t, "Ana Lifecycle", "ana-lifecycle@example.com")
	w := withPassword(t, s.App, asWho(ana))

	seedWorkspaceData(t, s.App, w, "lcx")

	secret := rawToken(t, s.App, w.User, w.Workspace.ID, "accounts:read")

	w.Client.PostForm("/account/export", passwordForm(knownPassword)).AssertRedirect("/account/privacy?export=ready")
	body := w.Client.Get("/account/export.json").AssertOK().Body()

	for _, want := range []string{
		"ana-lifecycle@example.com", "Ana Lifecycle", "lcx-account", "lcx-category", "lcx-tag", "lcx-lunch", "lcx-goal",
		"lcx-note", "lcx-rule", "lcx-card",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the export has no %q", want)
		}
	}

	for _, never := range []string{w.User.Password, secret, "two_factor_secret", "key_ciphertext"} {
		if never != "" && strings.Contains(body, never) {
			t.Errorf("the export holds %.30q", never)
		}
	}
}

// TestDataLifecycle_WorkspaceDeletionLeavesNoRow follows a workspace from the
// request to the purge: read-only in between, then no row of it in any table
// (soft deleted ones included), no trace of its text anywhere in the database,
// and another workspace untouched.
func TestDataLifecycle_WorkspaceDeletionLeavesNoRow(t *testing.T) {
	t.Parallel()

	s := lifecycleApp(t)
	ana := s.SignUp(t, "Ana Lifecycle", "ana-lifecycle@example.com")
	bob := s.SignUp(t, "Bob Keeper", "bob-keeper@example.com")
	w := withPassword(t, s.App, asWho(ana))
	b := asWho(bob)

	seedWorkspaceData(t, s.App, w, "lcw")
	seedWorkspaceData(t, s.App, b, "lck")

	token := rawToken(t, s.App, w.User, w.Workspace.ID, allScopes()...)

	kept := workspaceRowCounts(t, s.App, b.Workspace.ID)

	before := workspaceRowCounts(t, s.App, w.Workspace.ID)
	for _, table := range []string{"accounts", "transactions", "goals", "budgets", "recurrences", "credit_cards", "tags", "sync_devices", "changes"} {
		if before[table] == 0 {
			t.Fatalf("the seed left no row in %s", table)
		}
	}

	if lifecycleCount(t, s.App, "SELECT COUNT(*) FROM transactions WHERE workspace_id = ? AND deleted_at IS NOT NULL", w.Workspace.ID) != 1 {
		t.Fatal("the seed has no soft deleted transaction")
	}

	w.Client.PostForm("/workspace/deletion", deletionForm(w.Workspace.Name, knownPassword)).AssertRedirect("/account/privacy")

	// The grace period: nothing is erased yet, and nothing can be written.
	apiClient(t, s.App, token).PostJSON("/api/v1/accounts", map[string]any{"name": "late", "kind": "cash"}).AssertStatus(423)

	if got := workspaceRowCounts(t, s.App, w.Workspace.ID); got["transactions"] != before["transactions"] {
		t.Fatalf("rows changed during the grace period: %v", got)
	}

	// The cancel is real: it brings the workspace back, and the request can be made again.
	w.Client.PostForm("/workspace/deletion/cancel", url.Values{}).AssertRedirect("/account/privacy")
	w.Client.PostForm("/workspace/deletion", deletionForm(w.Workspace.Name, knownPassword)).AssertRedirect("/account/privacy")

	endSubscriptions(t, s.App)
	dueNow(t, s.App, "workspaces SET purge_at = ? WHERE id = ?", w.Workspace.ID)
	runPurge(t, s.App)

	for table, n := range workspaceRowCounts(t, s.App, w.Workspace.ID) {
		if n != 0 {
			t.Errorf("%s still holds %d rows of the purged workspace", table, n)
		}
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM workspaces WHERE id = ?", w.Workspace.ID); n != 0 {
		t.Error("the workspace row is still there")
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM access_tokens WHERE resource = ?", idString(w.Workspace.ID)); n != 0 {
		t.Errorf("%d tokens of the purged workspace remain", n)
	}

	for _, needle := range []string{"lcw-", w.Workspace.Name} {
		if where := rowsHolding(t, s.App, needle); len(where) > 0 {
			t.Errorf("%q of the purged workspace is still in %v", needle, where)
		}
	}

	for table, n := range workspaceRowCounts(t, s.App, b.Workspace.ID) {
		if n != kept[table] {
			t.Errorf("%s of another workspace: %d rows, were %d", table, n, kept[table])
		}
	}

	if n := lifecycleCount(t, s.App, "SELECT COUNT(*) FROM users WHERE id = ?", w.User.ID); n != 1 {
		t.Error("deleting the workspace deleted the person")
	}
}
