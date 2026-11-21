package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Epic 10: recurrences holds the rules of repeating transactions, and
// transactions gain the natural key of a generated occurrence
// (recurrence_uid, occurrence_on) and the installment fields. The unique
// index on the natural key counts soft-deleted rows too, so a deleted
// occurrence is never generated again and two runs converge. The change-log
// triggers of transactions are reinstalled because the writable columns grew;
// the entities are written out so this migration's checksum never moves with
// the registry. last_error is server state and stays out of the change log.
func m20261122130000CreateRecurrences() schema.Migration {
	rules := changes.Entity{
		Name: "recurrence", Table: "recurrences",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "frequency", "interval",
			"start_date", "end_date", "max_count", "generated_through", "active",
		},
	}
	transactions := changes.Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid", "tag_uids",
			"statement_ref", "recurrence_uid", "occurrence_on", "installment_group_uid", "installment_no", "installment_total",
		},
	}

	return schema.Migration{
		ID: "20261122130000_create_recurrences",
		Up: func(s *schema.Schema) error {
			if err := s.Create("recurrences", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.BigInteger("account_id")
				t.BigInteger("category_id").Nullable()
				t.String("kind", 10)
				t.BigInteger("amount_cents")
				t.String("description").Default("")
				t.String("frequency", 10)
				t.Integer("interval").Default(1)
				t.String("start_date", 10)
				t.String("end_date", 10).Nullable()
				t.Integer("max_count").Nullable()
				t.String("generated_through", 10).Nullable()
				t.Boolean("active").Default(true)
				t.String("last_error").Nullable()
				t.Timestamps()
				syncColumns(t)
				t.Index("workspace_id", "account_id")
			}); err != nil {
				return err
			}

			if err := s.Table("transactions", func(t *schema.Table) {
				t.ULID("recurrence_uid").Nullable()
				t.String("occurrence_on", 10).Nullable()
				t.ULID("installment_group_uid").Nullable()
				t.Integer("installment_no").Nullable()
				t.Integer("installment_total").Nullable()
			}); err != nil {
				return err
			}

			if err := s.Raw("CREATE UNIQUE INDEX transactions_recurrence_occurrence_unique ON transactions (workspace_id, recurrence_uid, occurrence_on) WHERE recurrence_uid IS NOT NULL"); err != nil {
				return err
			}

			if err := s.Raw("CREATE INDEX transactions_installment_group_index ON transactions (workspace_id, installment_group_uid) WHERE installment_group_uid IS NOT NULL"); err != nil {
				return err
			}

			for _, e := range []changes.Entity{rules, transactions} {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			prev := changes.Entity{
				Name: "transaction", Table: "transactions",
				Fields: []string{
					"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid", "tag_uids",
					"statement_ref",
				},
			}

			if err := s.DropIfExists("recurrences"); err != nil {
				return err
			}

			if err := changes.InstallTriggers(s, prev); err != nil {
				return err
			}

			for _, ix := range []string{"transactions_recurrence_occurrence_unique", "transactions_installment_group_index"} {
				if err := s.Raw("DROP INDEX IF EXISTS " + ix); err != nil {
					return err
				}
			}

			return s.Table("transactions", func(t *schema.Table) {
				t.DropColumn("recurrence_uid")
				t.DropColumn("occurrence_on")
				t.DropColumn("installment_group_uid")
				t.DropColumn("installment_no")
				t.DropColumn("installment_total")
			})
		},
	}
}
