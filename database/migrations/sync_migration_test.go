package migrations_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"
	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

const syncMigration = "20261121200000_add_sync_columns"

func openDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(t.Context(), "file:"+t.TempDir()+"/sync.db")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func exec(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.SQL().ExecContext(t.Context(), query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// TestSyncMigration applies the sync migration to a database that already
// holds rows in every synced table and checks that each one got a valid,
// unique uid and kept its data.
func TestSyncMigration(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	var before []schema.Migration

	for _, m := range migrations.All() {
		if m.ID < syncMigration {
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

	for _, ws := range []string{"One", "Two"} {
		exec(t, db, "INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES (?, 'USD', '2026-01-02T03:04:05.006Z', '2026-01-02T03:04:05.006Z')", ws)
	}

	for ws := 1; ws <= 2; ws++ {
		for i := range 1500 {
			exec(t, db, "INSERT INTO categories (workspace_id, name, kind, created_at, updated_at) VALUES (?, ?, 'expense', '2026-01-02T03:04:05.006Z', '2026-01-02T03:04:05.006Z')", ws, "cat"+string(rune('a'+i%26))+strings.Repeat("x", i/26))
		}

		exec(t, db, "INSERT INTO accounts (workspace_id, name, kind, created_at, updated_at) VALUES (?, 'Main', 'checking', '2026-01-02T03:04:05.006Z', '2026-01-02T03:04:05.006Z')", ws)
		exec(t, db, "INSERT INTO transactions (workspace_id, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (?, (SELECT id FROM accounts WHERE workspace_id = ?), (SELECT MIN(id) FROM categories WHERE workspace_id = ?), 'expense', 1234, 'lunch', '2026-01-02', '2026-01-02T03:04:05.006Z', '2026-01-02T03:04:05.006Z')", ws, ws, ws)
		exec(t, db, "INSERT INTO budgets (workspace_id, category_id, limit_cents, created_at, updated_at) VALUES (?, (SELECT MIN(id) FROM categories WHERE workspace_id = ?), 50000, '2026-01-02T03:04:05.006Z', '2026-01-02T03:04:05.006Z')", ws, ws)
	}

	all, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := all.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{"accounts", "categories", "transactions", "budgets"} {
		rows, err := db.SQL().QueryContext(t.Context(), "SELECT workspace_id, uid, version, deleted_at FROM "+table)
		if err != nil {
			t.Fatal(err)
		}

		seen := map[string]bool{}
		n := 0

		for rows.Next() {
			var (
				ws      int64
				uid     string
				version int64
				deleted *string
			)

			if err := rows.Scan(&ws, &uid, &version, &deleted); err != nil {
				t.Fatal(err)
			}

			n++

			if !ulid.Valid(uid) {
				t.Fatalf("%s: uid %q is not a valid ULID", table, uid)
			}

			if at, _ := ulid.TimeOf(uid); at.Year() != 2026 {
				t.Errorf("%s: uid %q carries the year %d, want the row's 2026", table, uid, at.Year())
			}

			key := string(rune('0'+ws)) + uid
			if seen[key] {
				t.Fatalf("%s: duplicate uid %q", table, uid)
			}

			seen[key] = true

			if version != 1 || deleted != nil {
				t.Errorf("%s: version %d deleted_at %v, want 1 and NULL", table, version, deleted)
			}
		}

		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}

		_ = rows.Close()

		if n == 0 {
			t.Fatalf("%s: no rows survived the migration", table)
		}
	}

	var amount int64
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT amount_cents FROM transactions WHERE workspace_id = 1").Scan(&amount); err != nil || amount != 1234 {
		t.Fatalf("transaction amount = %d, %v; want 1234", amount, err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	for _, tbl := range snap.Tables {
		switch tbl.Name {
		case "categories", "budgets":
			if !slices.ContainsFunc(tbl.Indexes, func(ix schema.IndexInfo) bool {
				return ix.Unique && strings.Contains(strings.ToLower(ix.Where), "deleted_at is null")
			}) {
				t.Errorf("%s: no partial unique index on live rows: %+v", tbl.Name, tbl.Indexes)
			}
		}
	}

	exec(t, db, "UPDATE categories SET deleted_at = '2026-02-01 00:00:00' WHERE workspace_id = 1 AND name = 'cata'")
	exec(t, db, "INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'cata', 'expense', '2026-02-01', '2026-02-01')", ulid.NewGenerator().New())
}
