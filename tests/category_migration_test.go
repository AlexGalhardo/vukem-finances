package tests

import (
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

const categoryMigration = "20261121220000_add_categories_tags_and_import_hash"

// TestCategoryMigration applies the categories, tags and import hash migration
// to a database that already holds categories and transactions, and checks the
// defaults, the indexes, the tag uniqueness and the reinstalled triggers.
func TestCategoryMigration(t *testing.T) {
	t.Parallel()

	db, err := database.Open(t.Context(), "file:"+t.TempDir()+"/categories.db")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	run := func(query string, args ...any) {
		t.Helper()

		if _, err := db.SQL().ExecContext(t.Context(), query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	var before []schema.Migration

	for _, m := range migrations.All() {
		if m.ID < categoryMigration {
			before = append(before, m)
		}
	}

	pre, err := schema.NewMigrator(db, before)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pre.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	const stamp = "'2026-01-02T03:04:05Z'"

	run("INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES ('One', 'USD', " + stamp + ", " + stamp + ")")
	run("INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, '01JH0000000000000000000A00', 'Main', 'checking', " + stamp + ", " + stamp + ")")
	run("INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, '01JH0000000000000000000C00', 'Housing', 'expense', " + stamp + ", " + stamp + ")")
	run("INSERT INTO transactions (workspace_id, uid, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, '01JH00000000000000000TX000', 1, 1, 'expense', 1234, 'rent', '2026-01-02', " + stamp + ", " + stamp + ")")

	all, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := all.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	var (
		parent   *int64
		color    string
		icon     string
		archived *string
	)

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT parent_id, color, icon, archived_at FROM categories WHERE id = 1").
		Scan(&parent, &color, &icon, &archived); err != nil || parent != nil || color != "" || icon != "" || archived != nil {
		t.Fatalf("category = %v %q %q %v, %v; want NULL, empty, empty, NULL", parent, color, icon, archived, err)
	}

	var (
		tagUIDs string
		hash    *string
		amount  int64
	)

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT tag_uids, import_hash, amount_cents FROM transactions WHERE id = 1").
		Scan(&tagUIDs, &hash, &amount); err != nil || tagUIDs != "[]" || hash != nil || amount != 1234 {
		t.Fatalf("transaction = %q %v %d, %v; want [], NULL, 1234", tagUIDs, hash, amount, err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	for _, tbl := range snap.Tables {
		if tbl.Name != "transactions" {
			continue
		}

		if !slices.ContainsFunc(tbl.Indexes, func(ix schema.IndexInfo) bool {
			return len(ix.Columns) == 2 && ix.Columns[0] == "workspace_id" && ix.Columns[1] == "import_hash" && !ix.Unique
		}) {
			t.Errorf("transactions: no plain index on (workspace_id, import_hash): %+v", tbl.Indexes)
		}
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}

	// The reinstalled triggers log the new writable columns.
	run("UPDATE categories SET color = 'blue' WHERE id = 1")
	run("UPDATE transactions SET tag_uids = '[\"01JH0000000000000000000T00\"]' WHERE id = 1")
	run("INSERT INTO tags (workspace_id, uid, name, created_at, updated_at) VALUES (1, '01JH0000000000000000000T00', 'Trip', " + stamp + ", " + stamp + ")")

	for entity, want := range map[string]string{"category": `["color"]`, "transaction": `["tag_uids"]`} {
		var fields string
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT fields FROM changes WHERE entity = ? ORDER BY seq DESC LIMIT 1", entity).Scan(&fields); err != nil || fields != want {
			t.Errorf("%s change fields = %q, %v; want %s", entity, fields, err, want)
		}
	}

	var tagChanges int
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM changes WHERE entity = 'tag'").Scan(&tagChanges); err != nil || tagChanges != 1 {
		t.Errorf("tag changes = %d, %v; want 1", tagChanges, err)
	}

	// Names are unique per workspace without regard to case, among live tags only.
	_, err = db.SQL().ExecContext(t.Context(), "INSERT INTO tags (workspace_id, uid, name, created_at, updated_at) VALUES (1, '01JH0000000000000000000T01', 'TRIP', "+stamp+", "+stamp+")")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("a second tag named TRIP = %v, want a unique violation", err)
	}

	run("UPDATE tags SET deleted_at = " + stamp + " WHERE uid = '01JH0000000000000000000T00'")
	run("INSERT INTO tags (workspace_id, uid, name, created_at, updated_at) VALUES (1, '01JH0000000000000000000T01', 'TRIP', " + stamp + ", " + stamp + ")")
}
