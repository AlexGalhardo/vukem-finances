package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// ChangeTriggersID is the migration after which every synced table has its
// change-log triggers; the boot check waits for it.
const ChangeTriggersID = "20261121200100_install_change_triggers"

// The triggers of the four synced tables. They run after
// 20261121200000_add_sync_columns because its table rebuilds drop triggers.
// The entities are written out, not read from the registry, so this
// migration's checksum does not move when a later epic changes the registry.
func m20261121200100InstallChangeTriggers() schema.Migration {
	entities := []changes.Entity{
		{Name: "account", Table: "accounts", Fields: []string{"name", "kind", "opening_cents"}},
		{Name: "category", Table: "categories", Fields: []string{"name", "kind"}},
		{
			Name: "transaction", Table: "transactions",
			Fields: []string{"account_id", "category_id", "kind", "amount_cents", "description", "occurred_on"},
		},
		{Name: "budget", Table: "budgets", Fields: []string{"category_id", "limit_cents"}},
	}

	return schema.Migration{
		ID: ChangeTriggersID,
		Up: func(s *schema.Schema) error {
			for _, e := range entities {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, e := range entities {
				for _, name := range changes.TriggerNames(e.Table) {
					if err := s.Raw("DROP TRIGGER IF EXISTS " + name); err != nil {
						return err
					}
				}
			}

			return nil
		},
	}
}
