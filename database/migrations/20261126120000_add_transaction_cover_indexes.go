package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 24 task 10 (QA-503): the balances, the dashboard, the net worth series
// and the list totals read every live transaction of a workspace and today go
// back to the table for the account, the kind and the amount of each row (180
// to 290 ms at 100000 rows). These indexes hold every column those queries
// read, so SQLite answers them from the index alone, and the first one is in
// the order of the net worth series GROUP BY, so it needs no sort.
func m20261126120000AddTransactionCoverIndexes() schema.Migration {
	const (
		byDate    = "transactions_workspace_occurred_cover_index"
		byAccount = "transactions_workspace_account_cover_index"
	)

	return schema.Migration{
		ID: "20261126120000_add_transaction_cover_indexes",
		Up: func(s *schema.Schema) error {
			for _, stmt := range []string{
				"CREATE INDEX " + byDate + " ON transactions (workspace_id, occurred_on, account_id, kind, amount_cents, deleted_at, transfer_uid, category_id)",
				"CREATE INDEX " + byAccount + " ON transactions (workspace_id, account_id, kind, occurred_on, amount_cents, deleted_at)",
			} {
				if err := s.Raw(stmt); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, name := range []string{byDate, byAccount} {
				if err := s.Raw("DROP INDEX IF EXISTS " + name); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
