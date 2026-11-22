package migrations_test

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// TestTransactionCoverIndexesAnswerTheHotQueries is QA-503: the list total, the
// balances and the net worth series read from the index alone, and the
// migration rolls back.
func TestTransactionCoverIndexesAnswerTheHotQueries(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	m, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	for name, query := range map[string]string{
		"list total":       `SELECT COUNT(*) FROM transactions WHERE deleted_at IS NULL AND workspace_id = ?`,
		"balances":         `SELECT account_id, kind, SUM(amount_cents) FROM transactions WHERE deleted_at IS NULL AND workspace_id = ? AND occurred_on <= '2999-01-01' GROUP BY account_id, kind`,
		"net worth":        `SELECT occurred_on, account_id, kind, amount_cents, COUNT(*) FROM transactions WHERE deleted_at IS NULL AND workspace_id = ? AND occurred_on <= '2999-01-01' GROUP BY occurred_on, account_id, kind, amount_cents ORDER BY occurred_on`,
		"count by account": `SELECT account_id, COUNT(*) FROM transactions WHERE deleted_at IS NULL AND workspace_id = ? GROUP BY account_id`,
	} {
		rows, err := db.SQL().QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+query, 1)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		var plan []string

		for rows.Next() {
			var id, parent, unused int
			var detail string

			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}

			plan = append(plan, detail)
		}

		_ = rows.Close()

		if all := strings.Join(plan, "; "); !strings.Contains(all, "COVERING INDEX") {
			t.Errorf("%s does not read from a covering index: %s", name, all)
		}
	}

	if _, err := m.Rollback(t.Context(), 2); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var n int

	err = db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name LIKE 'transactions_workspace_%_cover_index'").Scan(&n)
	if err != nil || n != 0 {
		t.Errorf("cover indexes after rollback = %d (%v), want 0", n, err)
	}
}
