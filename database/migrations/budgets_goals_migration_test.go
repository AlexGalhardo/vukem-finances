package migrations_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// TestBudgetsGoalsMigrationsRoundTrip applies epic 11, rolls its three
// migrations back with the change triggers installed (the rebuild of budgets
// must survive them and a live overall budget has no place to go back to),
// and applies them again. Later migrations (the report index of epic 12) are
// rolled back first, so the count of the rollback includes them.
func TestBudgetsGoalsMigrationsRoundTrip(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	m, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	const ts = "'2026-01-02T03:04:05.006Z'"

	exec(t, db, "INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES ('One', 'USD', "+ts+", "+ts+")")
	exec(t, db, "INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, '01J0000000000000000000000A', 'Food', 'expense', "+ts+", "+ts+")")
	exec(t, db, "INSERT INTO budgets (workspace_id, uid, category_id, limit_cents, rollover, created_at, updated_at) VALUES (1, '01J0000000000000000000000B', 1, 100, 1, "+ts+", "+ts+")")
	exec(t, db, "INSERT INTO budgets (workspace_id, uid, category_id, limit_cents, created_at, updated_at) VALUES (1, '01J0000000000000000000000C', NULL, 900, "+ts+", "+ts+")")

	if _, err := db.SQL().ExecContext(t.Context(),
		"INSERT INTO budgets (workspace_id, uid, category_id, limit_cents, created_at, updated_at) VALUES (1, '01J0000000000000000000000D', NULL, 1, "+ts+", "+ts+")"); err == nil {
		t.Fatal("a second live overall budget was accepted")
	}

	if _, err := m.Rollback(t.Context(), migrationsFrom(t, "20261122140000_add_budget_rollover_and_overall")); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var n int

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM budgets").Scan(&n); err != nil || n != 1 {
		t.Fatalf("budgets after rollback = %d (%v), want only the category one", n, err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}
