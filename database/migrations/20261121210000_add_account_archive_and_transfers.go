package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// accountKinds are the kinds an account may have. credit_card is reserved
// for the cards epic and refused until then.
var accountKinds = []string{"checking", "savings", "cash", "investment", "wallet"}

// Accounts gain archived_at and position, transactions gain transfer_uid (the
// pairing of the two legs of a transfer), and an account kind outside the
// enum becomes checking. The change-log triggers of both tables are
// reinstalled because the writable columns grew; the entities are written
// out so this migration's checksum never moves with the registry.
func m20261121210000AddAccountArchiveAndTransfers() schema.Migration {
	accounts := changes.Entity{
		Name: "account", Table: "accounts",
		Fields: []string{"name", "kind", "opening_cents", "archived_at", "position"},
	}
	transactions := changes.Entity{
		Name: "transaction", Table: "transactions",
		Fields: []string{"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on", "transfer_uid"},
	}

	return schema.Migration{
		ID: "20261121210000_add_account_archive_and_transfers",
		Up: func(s *schema.Schema) error {
			// Before the columns and the new triggers: the old trigger logs
			// the kind change, so a client learns of the correction.
			if err := s.Raw("UPDATE accounts SET kind = 'checking' WHERE kind NOT IN (" + quoted(accountKinds) + ")"); err != nil {
				return err
			}

			if has, err := s.HasColumn("accounts", "archived_at"); err != nil {
				return err
			} else if !has {
				if err := s.Table("accounts", func(t *schema.Table) {
					t.Timestamp("archived_at").Nullable()
					t.Integer("position").Default(0)
				}); err != nil {
					return err
				}
			}

			if has, err := s.HasColumn("transactions", "transfer_uid"); err != nil {
				return err
			} else if !has {
				if err := s.Table("transactions", func(t *schema.Table) {
					t.ULID("transfer_uid").Nullable()
					t.Index("workspace_id", "transfer_uid").Where("transfer_uid IS NOT NULL")
				}); err != nil {
					return err
				}
			}

			for _, e := range []changes.Entity{accounts, transactions} {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			// The previous trigger set is the one of the earlier migration.
			prev := []changes.Entity{
				{Name: "account", Table: "accounts", Fields: []string{"name", "kind", "opening_cents"}},
				{
					Name: "transaction", Table: "transactions",
					Fields: []string{"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on"},
				},
			}

			if err := s.Raw("DROP INDEX IF EXISTS transactions_workspace_id_transfer_uid_index"); err != nil {
				return err
			}

			if err := s.Table("transactions", func(t *schema.Table) { t.DropColumn("transfer_uid") }); err != nil {
				return err
			}

			if err := s.Table("accounts", func(t *schema.Table) {
				t.DropColumn("archived_at")
				t.DropColumn("position")
			}); err != nil {
				return err
			}

			for _, e := range prev {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func quoted(values []string) string {
	out := ""

	for i, v := range values {
		if i > 0 {
			out += ", "
		}

		out += "'" + v + "'"
	}

	return out
}
