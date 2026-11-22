package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 12: the reports group transactions by category, so they get an index
// that starts with the workspace and carries the date for the period range.
// Account grouping is served by the existing (account_id) index, tags by
// transaction_tags.
func m20261123100000AddReportIndexes() schema.Migration {
	return schema.Migration{
		ID: "20261123100000_add_report_indexes",
		Up: func(s *schema.Schema) error {
			return s.Raw("CREATE INDEX transactions_workspace_category_occurred_index ON transactions (workspace_id, category_id, occurred_on)")
		},
		Down: func(s *schema.Schema) error {
			return s.Raw("DROP INDEX IF EXISTS transactions_workspace_category_occurred_index")
		},
	}
}
