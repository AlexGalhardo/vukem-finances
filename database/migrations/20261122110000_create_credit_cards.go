package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Epic 09: credit_cards holds the settings of a card whose money lives in an
// account of kind credit_card, and transactions gain statement_ref, the
// closing date of the statement a card purchase or payment belongs to. The
// change-log triggers of transactions are reinstalled because the writable
// columns grew; the entities are written out so this migration's checksum
// never moves with the registry. No column holds a card number, an expiry or
// a security code: only the last four digits.
func m20261122110000CreateCreditCards() schema.Migration {
	cards := changes.Entity{
		Name: "credit_card", Table: "credit_cards",
		Fields: []string{"account_id", "limit_cents", "closing_day", "due_day", "payment_account_id", "brand", "last4"},
	}
	transactions := changes.Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{
			"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid", "tag_uids",
			"statement_ref",
		},
	}

	return schema.Migration{
		ID: "20261122110000_create_credit_cards",
		Up: func(s *schema.Schema) error {
			if err := s.Create("credit_cards", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.BigInteger("account_id")
				t.BigInteger("limit_cents").Default(0)
				t.Integer("closing_day")
				t.Integer("due_day")
				t.BigInteger("payment_account_id").Nullable()
				t.String("brand", 20).Default("")
				t.String("last4", 4).Default("")
				t.Timestamps()
				syncColumns(t)
				t.Index("workspace_id", "account_id")
			}); err != nil {
				return err
			}

			// One live card per account; a deleted card does not hold the account.
			if err := s.Raw("CREATE UNIQUE INDEX credit_cards_account_id_unique ON credit_cards (account_id) WHERE deleted_at IS NULL"); err != nil {
				return err
			}

			if has, err := s.HasColumn("transactions", "statement_ref"); err != nil {
				return err
			} else if !has {
				if err := s.Table("transactions", func(t *schema.Table) {
					t.String("statement_ref", 10).Nullable()
					t.Index("account_id", "statement_ref")
				}); err != nil {
					return err
				}
			}

			for _, e := range []changes.Entity{cards, transactions} {
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
				},
			}

			if err := s.DropIfExists("credit_cards"); err != nil {
				return err
			}

			if err := s.Raw("DROP INDEX IF EXISTS transactions_account_id_statement_ref_index"); err != nil {
				return err
			}

			if err := s.Table("transactions", func(t *schema.Table) { t.DropColumn("statement_ref") }); err != nil {
				return err
			}

			return changes.InstallTriggers(s, prev)
		},
	}
}
