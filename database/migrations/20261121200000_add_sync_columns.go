package migrations

import (
	"fmt"
	"strings"

	"github.com/AlexGalhardo/vukem/schema"
)

// syncTables are the synced entities of the sync data model
// (docs/specs/sync-data-model.md).
var syncTables = []string{"accounts", "categories", "transactions", "budgets"}

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ulidSQL builds a SQLite expression that returns a valid ULID: the ten
// characters of the timestamp come from the row's created_at (so a backfilled
// id sorts with its row), the sixteen of entropy from random(). It runs inside
// one UPDATE, so the backfill needs no Go loop and no batches.
func ulidSQL() string {
	ms := "(CAST(COALESCE(strftime('%s', substr(created_at, 1, 19)), 0) AS INTEGER) * 1000)"

	var parts []string

	for i := range 10 {
		parts = append(parts, fmt.Sprintf("substr('%s', ((%s >> %d) & 31) + 1, 1)", crockford, ms, 5*(9-i)))
	}

	for range 16 {
		parts = append(parts, fmt.Sprintf("substr('%s', (random() & 31) + 1, 1)", crockford))
	}

	return strings.Join(parts, " || ")
}

// The four synced tables get uid (a ULID a client can invent offline),
// version and deleted_at (soft delete). The columns are added nullable, uid is
// backfilled, then each table is rebuilt so uid is NOT NULL and the unique
// indexes become partial (a deleted name can be used again). Rebuild needs
// NoTransaction, so every step is safe to run again after a failure.
// Triggers are installed by the next migration, after the rebuilds, because a
// rebuild drops them.
func m20261121200000AddSyncColumns() schema.Migration {
	return schema.Migration{
		ID:            "20261121200000_add_sync_columns",
		NoTransaction: true,
		Up: func(s *schema.Schema) error {
			for _, table := range syncTables {
				if err := addSyncColumns(s, table); err != nil {
					return err
				}

				if err := s.Raw("UPDATE " + table + " SET uid = " + ulidSQL() + " WHERE uid IS NULL"); err != nil {
					return err
				}
			}

			rebuilds := []struct {
				table string
				def   func(t *schema.Table)
			}{
				{"accounts", func(t *schema.Table) {
					t.ID()
					t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
					t.String("name")
					t.String("kind")
					t.BigInteger("opening_cents").Default(0)
					t.Timestamps()
					syncColumns(t)
					t.Index("workspace_id")
				}},
				{"categories", func(t *schema.Table) {
					t.ID()
					t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
					t.String("name")
					t.String("kind")
					t.Timestamps()
					syncColumns(t)
					t.UniqueIndex("workspace_id", "name").Where("deleted_at IS NULL")
				}},
				{"transactions", func(t *schema.Table) {
					t.ID()
					t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
					t.ForeignID("account_id").Constrained("accounts").CascadeOnDelete()
					t.ForeignID("category_id").Nullable().Constrained("categories")
					t.String("kind")
					t.BigInteger("amount_cents")
					t.String("description")
					t.String("occurred_on", 10)
					t.ForeignID("import_id").Nullable()
					t.Timestamps()
					syncColumns(t)
					t.Index("workspace_id", "occurred_on")
					t.Index("account_id")
				}},
				{"budgets", func(t *schema.Table) {
					t.ID()
					t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
					t.ForeignID("category_id").Constrained("categories").CascadeOnDelete()
					t.BigInteger("limit_cents")
					t.Timestamps()
					syncColumns(t)
					t.UniqueIndex("workspace_id", "category_id").Where("deleted_at IS NULL")
				}},
			}

			for _, r := range rebuilds {
				if err := s.Rebuild(r.table, r.def, schema.DropTriggers()); err != nil {
					return err
				}
			}

			return createChangeLog(s)
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"sync_conflicts", "sync_state", "changes"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			for _, table := range syncTables {
				if err := s.Rebuild(table, legacyDefinition(table), schema.DropTriggers()); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func addSyncColumns(s *schema.Schema, table string) error {
	has, err := s.HasColumn(table, "uid")
	if err != nil || has {
		return err
	}

	return s.Table(table, func(t *schema.Table) {
		t.ULID("uid").Nullable()
		t.SoftDeletes()
		t.BigInteger("version").Default(1)
	})
}

// syncColumns is the tail every synced table definition shares.
func syncColumns(t *schema.Table) {
	t.ULID("uid")
	t.SoftDeletes()
	t.BigInteger("version").Default(1)
	t.UniqueIndex("workspace_id", "uid")
}

// createChangeLog creates changes (the per-workspace ordered log of synced
// rows), sync_state (what compaction removed) and sync_conflicts (values lost
// to a last-writer-wins merge). changes.seq is an AUTOINCREMENT rowid, so it
// only grows, even after compaction deletes the newest row.
func createChangeLog(s *schema.Schema) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS changes (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id INTEGER NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
			entity TEXT NOT NULL,
			uid CHAR(26) NOT NULL,
			op TEXT NOT NULL CHECK (op IN ('upsert', 'delete')),
			row_version INTEGER NOT NULL,
			fields TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`CREATE INDEX IF NOT EXISTS changes_workspace_id_seq_index ON changes (workspace_id, seq)`,
		`CREATE INDEX IF NOT EXISTS changes_row_index ON changes (workspace_id, entity, uid, seq)`,
	}

	for _, stmt := range stmts {
		if err := s.Raw(stmt); err != nil {
			return err
		}
	}

	if err := s.Create("sync_state", func(t *schema.Table) {
		t.ID()
		t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
		t.BigInteger("compacted_through_seq").Default(0)
		t.Timestamps()
		t.UniqueIndex("workspace_id")
	}); err != nil {
		return err
	}

	return s.Create("sync_conflicts", func(t *schema.Table) {
		t.ID()
		t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
		t.String("entity", 40)
		t.ULID("uid")
		t.String("field", 60)
		t.String("kind", 40).Default("field")
		t.Text("lost_value").Default("")
		t.String("lost_by_device", 80).Default("")
		t.Timestamps()
		t.Index("workspace_id", "created_at")
	})
}

// legacyDefinition is a synced table as it was before the sync columns.
func legacyDefinition(table string) func(t *schema.Table) {
	return func(t *schema.Table) {
		t.ID()
		t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()

		switch table {
		case "accounts":
			t.String("name")
			t.String("kind")
			t.BigInteger("opening_cents").Default(0)
			t.Timestamps()
			t.Index("workspace_id")
		case "categories":
			t.String("name")
			t.String("kind")
			t.Timestamps()
			t.UniqueIndex("workspace_id", "name")
		case "transactions":
			t.ForeignID("account_id").Constrained("accounts").CascadeOnDelete()
			t.ForeignID("category_id").Nullable().Constrained("categories")
			t.String("kind")
			t.BigInteger("amount_cents")
			t.String("description")
			t.String("occurred_on", 10)
			t.ForeignID("import_id").Nullable()
			t.Timestamps()
			t.Index("workspace_id", "occurred_on")
			t.Index("account_id")
		case "budgets":
			t.ForeignID("category_id").Constrained("categories").CascadeOnDelete()
			t.BigInteger("limit_cents")
			t.Timestamps()
			t.UniqueIndex("workspace_id", "category_id")
		}
	}
}
