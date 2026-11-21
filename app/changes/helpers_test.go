package changes_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

const now = "datetime('now')"

// openDB returns a migrated SQLite file database with two workspaces.
func openDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(t.Context(), "file:"+t.TempDir()+"/changes.db")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	m, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"One", "Two"} {
		exec(t, db, "INSERT INTO workspaces (name, currency, timezone, created_at, updated_at) VALUES (?, 'USD', 'UTC', "+now+", "+now+")", name)
	}

	return db
}

func exec(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.SQL().ExecContext(t.Context(), query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

type logRow struct {
	Seq        int64
	Workspace  int64
	Entity     string
	UID        string
	Op         string
	RowVersion int64
	Fields     string
}

func logOf(t *testing.T, db *database.DB, where string, args ...any) []logRow {
	t.Helper()

	rows, err := db.SQL().QueryContext(t.Context(),
		"SELECT seq, workspace_id, entity, uid, op, row_version, fields FROM changes "+where+" ORDER BY seq", args...)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	var out []logRow

	for rows.Next() {
		var r logRow
		if err := rows.Scan(&r.Seq, &r.Workspace, &r.Entity, &r.UID, &r.Op, &r.RowVersion, &r.Fields); err != nil {
			t.Fatal(err)
		}

		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return out
}
