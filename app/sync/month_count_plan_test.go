package sync_test

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"

	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// TestMonthQuotaCountUsesTheDateIndex: a push checks the plan's monthly quota
// once per transaction it creates, so the count must read only the rows of
// that month. A function of the column (substr) makes SQLite read every row of
// the workspace: 40 ms a mutation on 100000 rows (QA-500).
func TestMonthQuotaCountUsesTheDateIndex(t *testing.T) {
	t.Parallel()

	db, err := database.Open(t.Context(), "file:"+t.TempDir()+"/plan.db")
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

	rows, err := db.SQL().QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+syncproto.MonthCountSQL, syncproto.MonthArgs(1, "2026-11")...)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	var plan []string

	for rows.Next() {
		var id, parent, unused int
		var detail string

		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}

		plan = append(plan, detail)
	}

	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	all := strings.Join(plan, "; ")
	if !strings.Contains(all, "occurred_on>") || !strings.Contains(all, "occurred_on<") {
		t.Errorf("the month count does not search by a range of occurred_on: %s", all)
	}
}
