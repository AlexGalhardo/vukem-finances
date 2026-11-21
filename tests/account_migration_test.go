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

const accountMigration = "20261121210000_add_account_archive_and_transfers"

// TestAccountMigration applies the accounts and transfers migration to a
// database that already holds accounts of every kind, free-form ones
// included, and checks the kinds, the new columns, the index and the
// triggers.
func TestAccountMigration(t *testing.T) {
	t.Parallel()

	db, err := database.Open(t.Context(), "file:"+t.TempDir()+"/accounts.db")
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
		if m.ID < accountMigration {
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

	run("INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES ('One', 'USD', '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z')")

	kinds := []string{"checking", "savings", "cash", "investment", "wallet", "Brokerage", "credit_card", ""}
	for i, kind := range kinds {
		run("INSERT INTO accounts (workspace_id, uid, name, kind, opening_cents, created_at, updated_at) VALUES (1, ?, ?, ?, 500, '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z')",
			"01JH0000000000000000000"+string(rune('A'+i))+"00", "Account "+string(rune('A'+i)), kind)
	}

	run("INSERT INTO transactions (workspace_id, uid, account_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, '01JH00000000000000000TX000', 1, 'expense', 1234, 'lunch', '2026-01-02', '2026-01-02T03:04:05Z', '2026-01-02T03:04:05Z')")

	all, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := all.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	rows, err := db.SQL().QueryContext(t.Context(), "SELECT name, kind, opening_cents, archived_at, position FROM accounts ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	want := []string{"checking", "savings", "cash", "investment", "wallet", "checking", "checking", "checking"}

	for i := 0; rows.Next(); i++ {
		var (
			name, kind string
			opening    int64
			archived   *string
			position   int
		)

		if err := rows.Scan(&name, &kind, &opening, &archived, &position); err != nil {
			t.Fatal(err)
		}

		if kind != want[i] || opening != 500 || archived != nil || position != 0 {
			t.Errorf("%s: kind %q opening %d archived %v position %d; want kind %q, 500, NULL, 0", name, kind, opening, archived, position, want[i])
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	var amount int64

	var transfer *string
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT amount_cents, transfer_uid FROM transactions").Scan(&amount, &transfer); err != nil || amount != 1234 || transfer != nil {
		t.Fatalf("transaction = %d, %v, %v; want 1234 and a null transfer_uid", amount, transfer, err)
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
			return len(ix.Columns) == 2 && ix.Columns[0] == "workspace_id" && ix.Columns[1] == "transfer_uid" &&
				strings.Contains(strings.ToLower(ix.Where), "transfer_uid is not null")
		}) {
			t.Errorf("transactions: no index on (workspace_id, transfer_uid): %+v", tbl.Indexes)
		}
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}

	// The reinstalled trigger logs the new writable columns.
	run("UPDATE accounts SET archived_at = '2026-02-01T00:00:00Z' WHERE id = 1")

	var fields string
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT fields FROM changes WHERE entity = 'account' ORDER BY seq DESC LIMIT 1").Scan(&fields); err != nil || fields != `["archived_at"]` {
		t.Fatalf("change fields = %q, %v; want [\"archived_at\"]", fields, err)
	}
}

// serverOnlyColumns are the synced-table columns no client writes, with the
// reason. Every other column must be a writable field of the registry entry.
var serverOnlyColumns = map[string]map[string]string{
	"recurrences": {
		"last_error": "server state of the generation job, never edited by a client",
	},
	"transactions": {
		"import_id":   "set by the server-side CSV import",
		"import_hash": "the duplicate key of the importer",
	},
}

// metaColumns exist on every synced table and are not client fields.
var metaColumns = []string{"id", "workspace_id", "uid", "version", "created_at", "updated_at", "deleted_at"}

// TestSyncFieldsCoverColumns fails when a column of a synced table is neither
// a writable field of its registry entry nor on the explicit server-only list,
// so a new column cannot silently stay out of the change log and of sync.
func TestSyncFieldsCoverColumns(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range changes.Entities() {
		rows, err := db.SQL().QueryContext(t.Context(), "SELECT name FROM pragma_table_info('"+e.Table+"')")
		if err != nil {
			t.Fatal(err)
		}

		columns := map[string]bool{}

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}

			columns[name] = true
		}

		_ = rows.Close()

		for column := range columns {
			_, serverOnly := serverOnlyColumns[e.Table][column]
			if !slices.Contains(metaColumns, column) && !slices.Contains(e.Fields, column) && !serverOnly {
				t.Errorf("%s.%s is neither in Fields of the %q entity nor on serverOnlyColumns", e.Table, column, e.Name)
			}
		}

		for _, f := range e.Fields {
			if !columns[f] {
				t.Errorf("entity %q lists Fields %q that %s does not have", e.Name, f, e.Table)
			}
		}

		for column := range serverOnlyColumns[e.Table] {
			if slices.Contains(e.Fields, column) {
				t.Errorf("%s.%s is on serverOnlyColumns and in Fields", e.Table, column)
			}
		}
	}
}
