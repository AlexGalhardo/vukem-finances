package migrations

import (
	"fmt"

	"github.com/AlexGalhardo/vukem/schema"
)

// Epic 24 task 10 (QA-503): the dashboard and the reports add up every live
// transaction of a period at request time, which no index can make cheap at
// 100000 rows. transaction_month_totals holds those sums per workspace, month,
// account, category, kind and transfer flag, kept by triggers on transactions,
// so every writer (the services, imports, sync push, jobs, the purge) is
// counted without any discipline, like the change log.
//
//   - Only live rows count: a soft delete subtracts, a restore adds.
//   - category_id 0 means no category; transfer is 1 for a transfer leg.
//   - Money is exact: the cents of a row are split into a high part (cents >> 32)
//     and a low part (the rest, below 2^32) that are summed apart, so a
//     bucket never overflows an int64 whatever the amounts. The reader joins
//     them as hi * 2^32 + lo.
//   - A removal only updates (and deletes a bucket that reached zero): it never
//     inserts, so it cannot fail on a workspace that is being deleted.
//
// The backfill is one grouped INSERT ... SELECT over the covering index of the
// transactions: its memory is the number of buckets, not of rows, and it runs
// before the triggers exist, in the one migration transaction, so no write is
// missed or counted twice. A later migration that rebuilds transactions drops
// these triggers (a rebuild does): it must reinstall them, and the isolation
// guard test fails when they are missing.
func m20261127100000CreateTransactionMonthTotals() schema.Migration {
	const (
		table    = "transaction_month_totals"
		key      = "workspace_id, month, account_id, category_id, kind, transfer"
		insertTr = "transactions_totals_insert"
		updateTr = "transactions_totals_update"
		deleteTr = "transactions_totals_delete"
	)

	hi := func(r string) string { return r + ".amount_cents >> 32" }
	lo := func(r string) string { return r + ".amount_cents - ((" + r + ".amount_cents >> 32) << 32)" }

	// sameBucket matches the bucket of the row r (OLD or NEW).
	sameBucket := func(r string) string {
		return fmt.Sprintf("workspace_id = %[1]s.workspace_id AND month = substr(%[1]s.occurred_on, 1, 7) "+
			"AND account_id = %[1]s.account_id AND category_id = COALESCE(%[1]s.category_id, 0) "+
			"AND kind = %[1]s.kind AND transfer = (%[1]s.transfer_uid IS NOT NULL)", r)
	}

	add := func(r string) string {
		return fmt.Sprintf(`INSERT INTO %[1]s (workspace_id, month, account_id, category_id, kind, transfer, n, amount_hi, amount_lo)
	SELECT %[2]s.workspace_id, substr(%[2]s.occurred_on, 1, 7), %[2]s.account_id, COALESCE(%[2]s.category_id, 0),
		%[2]s.kind, %[2]s.transfer_uid IS NOT NULL, 1, %[3]s, %[4]s
	WHERE %[2]s.deleted_at IS NULL
	ON CONFLICT (%[5]s) DO UPDATE SET n = n + 1, amount_hi = amount_hi + excluded.amount_hi, amount_lo = amount_lo + excluded.amount_lo;`,
			table, r, hi(r), lo(r), key)
	}

	remove := func(r string) string {
		return fmt.Sprintf(`UPDATE %[1]s SET n = n - 1, amount_hi = amount_hi - (%[3]s), amount_lo = amount_lo - (%[4]s)
	WHERE %[2]s AND %[5]s.deleted_at IS NULL;
	DELETE FROM %[1]s WHERE %[2]s AND n = 0 AND %[5]s.deleted_at IS NULL;`,
			table, sameBucket(r), hi(r), lo(r), r)
	}

	triggers := []string{
		fmt.Sprintf("CREATE TRIGGER %s AFTER INSERT ON transactions\nBEGIN\n\t%s\nEND", insertTr, add("NEW")),
		fmt.Sprintf(`CREATE TRIGGER %s AFTER UPDATE ON transactions
WHEN (OLD.deleted_at IS NULL OR NEW.deleted_at IS NULL) AND ((OLD.deleted_at IS NULL) <> (NEW.deleted_at IS NULL)
	OR OLD.account_id IS NOT NEW.account_id OR OLD.category_id IS NOT NEW.category_id OR OLD.kind IS NOT NEW.kind
	OR OLD.amount_cents IS NOT NEW.amount_cents OR OLD.occurred_on IS NOT NEW.occurred_on
	OR OLD.transfer_uid IS NOT NEW.transfer_uid)
BEGIN
	%s
	%s
END`, updateTr, remove("OLD"), add("NEW")),
		fmt.Sprintf("CREATE TRIGGER %s AFTER DELETE ON transactions\nBEGIN\n\t%s\nEND", deleteTr, remove("OLD")),
	}

	return schema.Migration{
		ID: "20261127100000_create_transaction_month_totals",
		Up: func(s *schema.Schema) error {
			if err := s.Create(table, func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("month", 7)
				t.BigInteger("account_id")
				t.BigInteger("category_id").Default(0)
				t.String("kind", 20)
				t.Integer("transfer").Default(0)
				t.BigInteger("n").Default(0)
				t.BigInteger("amount_hi").Default(0)
				t.BigInteger("amount_lo").Default(0)
				t.UniqueIndex("workspace_id", "month", "account_id", "category_id", "kind", "transfer")
			}); err != nil {
				return err
			}

			backfill := fmt.Sprintf(`INSERT INTO %s (%s, n, amount_hi, amount_lo)
SELECT workspace_id, substr(occurred_on, 1, 7), account_id, COALESCE(category_id, 0), kind, transfer_uid IS NOT NULL,
	COUNT(*), SUM(amount_cents >> 32), SUM(amount_cents - ((amount_cents >> 32) << 32))
FROM transactions WHERE deleted_at IS NULL
GROUP BY workspace_id, substr(occurred_on, 1, 7), account_id, COALESCE(category_id, 0), kind, transfer_uid IS NOT NULL`, table, key)

			if err := s.Raw(backfill); err != nil {
				return err
			}

			for _, stmt := range triggers {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, name := range []string{insertTr, updateTr, deleteTr} {
				if err := s.Raw("DROP TRIGGER IF EXISTS " + name); err != nil {
					return err
				}
			}

			return s.DropIfExists(table)
		},
	}
}
