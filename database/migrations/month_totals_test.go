package migrations_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

const totalsMigration = "20261127100000_create_transaction_month_totals"

const stamp = "'2026-01-02T03:04:05.006Z'"

// TestMonthTotalsBackfillAndTriggers is QA-503: a database that already holds
// transactions (live, trashed, transfer legs, huge amounts, two workspaces) gets
// a summary equal to a recomputation, the triggers keep it equal through random
// writes, and the migration rolls back.
func TestMonthTotalsBackfillAndTriggers(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	var before []schema.Migration

	for _, m := range migrations.All() {
		if m.ID < totalsMigration {
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

	rng := rand.New(rand.NewPCG(7, 11))

	insertRandom := func(ws int) {
		t.Helper()

		var category any
		if rng.IntN(3) > 0 {
			category = rng.IntN(3)
		}

		var transfer any
		if rng.IntN(8) == 0 {
			transfer = "tr"
		}

		kind := [2]string{"income", "expense"}[rng.IntN(2)]
		amount := 1 + rng.Int64N(100000)

		if rng.IntN(20) == 0 {
			amount = 9007199254740991 - rng.Int64N(10)
		}

		exec(t, db, "INSERT INTO transactions (workspace_id, account_id, category_id, kind, amount_cents, description, occurred_on, transfer_uid, uid, created_at, updated_at) "+
			"VALUES (?, (SELECT MIN(id) + ? FROM accounts WHERE workspace_id = ?), (SELECT MIN(id) + ? FROM categories WHERE workspace_id = ? AND ? IS NOT NULL), ?, ?, 'x', ?, ?, hex(randomblob(13)), "+stamp+", "+stamp+")",
			ws, rng.IntN(3), ws, category, ws, category, kind, amount, randomDate(rng), transfer)
	}

	for ws := 1; ws <= 2; ws++ {
		exec(t, db, "INSERT INTO workspaces (name, currency, created_at, updated_at) VALUES ('W', 'USD', "+stamp+", "+stamp+")")

		for i := range 3 {
			exec(t, db, "INSERT INTO accounts (workspace_id, name, kind, uid, created_at, updated_at) VALUES (?, 'A', 'checking', hex(randomblob(13)), "+stamp+", "+stamp+")", ws)
			exec(t, db, "INSERT INTO categories (workspace_id, name, kind, uid, created_at, updated_at) VALUES (?, ?, 'expense', hex(randomblob(13)), "+stamp+", "+stamp+")", ws, "c"+string(rune('a'+i)))
		}

		for range 400 {
			insertRandom(ws)
		}
	}

	exec(t, db, "INSERT INTO transactions (workspace_id, account_id, kind, amount_cents, description, occurred_on, uid, created_at, updated_at, deleted_at) "+
		"VALUES (1, 1, 'expense', 9007199254740991, 'trashed', '2026-03-03', 'trashed-uid', "+stamp+", "+stamp+", "+stamp+")")

	all, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := all.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	check := func(step string) {
		t.Helper()

		if err := finance.CheckMonthTotals(t.Context(), db); err != nil {
			t.Fatalf("%s: %v", step, err)
		}
	}

	check("after the backfill")

	// Write only in workspace 1, then prove workspace 2 did not move.
	twoBefore := workspaceSummary(t, db, 2)

	for i := range 600 {
		switch rng.IntN(5) {
		case 0:
			insertRandom(1)
		case 1:
			exec(t, db, "UPDATE transactions SET amount_cents = ?, occurred_on = ? WHERE id = (SELECT id FROM transactions WHERE workspace_id = 1 ORDER BY random() LIMIT 1)",
				1+rng.Int64N(9007199254740991), randomDate(rng))
		case 2:
			exec(t, db, "UPDATE transactions SET deleted_at = "+stamp+" WHERE id = (SELECT id FROM transactions WHERE workspace_id = 1 AND deleted_at IS NULL ORDER BY random() LIMIT 1)")
		case 3:
			exec(t, db, "UPDATE transactions SET deleted_at = NULL WHERE id = (SELECT id FROM transactions WHERE workspace_id = 1 AND deleted_at IS NOT NULL ORDER BY random() LIMIT 1)")
		default:
			exec(t, db, "UPDATE transactions SET account_id = 1 + abs(random()) % 3, kind = CASE kind WHEN 'income' THEN 'expense' ELSE 'income' END, "+
				"transfer_uid = CASE WHEN random() % 2 = 0 THEN NULL ELSE 'x' END, category_id = NULL "+
				"WHERE id = (SELECT id FROM transactions WHERE workspace_id = 1 ORDER BY random() LIMIT 1)")
		}

		if i%50 == 0 {
			check("random write")
		}
	}

	exec(t, db, "DELETE FROM transactions WHERE workspace_id = 1 AND id % 3 = 0")
	check("after hard deletes")

	if got := workspaceSummary(t, db, 2); got != twoBefore {
		t.Errorf("workspace 2 summary moved: %q, was %q", got, twoBefore)
	}

	var triggers int

	err = db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'transactions_totals_%'").Scan(&triggers)
	if err != nil || triggers != 3 {
		t.Errorf("totals triggers = %d (%v), want 3", triggers, err)
	}

	if _, err := all.Rollback(t.Context(), 1); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	err = db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'transactions_totals_%' OR name = 'transaction_month_totals'").Scan(&triggers)
	if err != nil || triggers != 0 {
		t.Errorf("objects left after rollback = %d (%v), want 0", triggers, err)
	}

}

func randomDate(rng *rand.Rand) string {
	return "2026-" + twoDigits(1+rng.IntN(12)) + "-" + twoDigits(1+rng.IntN(28))
}

func twoDigits(n int) string { return string([]byte{byte('0' + n/10), byte('0' + n%10)}) }

// workspaceSummary renders the summary rows of a workspace, in key order.
func workspaceSummary(t *testing.T, db *database.DB, ws int) string {
	t.Helper()

	rows, err := db.SQL().QueryContext(t.Context(),
		"SELECT month, account_id, category_id, kind, transfer, n, amount_hi, amount_lo FROM transaction_month_totals WHERE workspace_id = ? ORDER BY 1, 2, 3, 4, 5", ws)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = rows.Close() }()

	var out string

	for rows.Next() {
		var month, kind string

		var a, c, tr, n, hi, lo int64

		if err := rows.Scan(&month, &a, &c, &kind, &tr, &n, &hi, &lo); err != nil {
			t.Fatal(err)
		}

		out += fmt.Sprintln(month, a, c, kind, tr, n, hi, lo)
	}

	return out
}
