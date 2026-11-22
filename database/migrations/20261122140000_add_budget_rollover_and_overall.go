package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Epic 11: a budget gains rollover and its category becomes nullable (a row
// without a category is the overall budget of the workspace). SQLite cannot
// drop NOT NULL, so the synced table is rebuilt, which drops its triggers;
// they are installed again after it, from an entity written out here so the
// checksum never moves with the registry. At most one live overall budget per
// workspace, enforced by a partial unique index (a NULL never collides in a
// plain unique index).
func m20261122140000AddBudgetRolloverAndOverall() schema.Migration {
	next := changes.Entity{Name: "budget", Table: "budgets", Fields: []string{"category_id", "limit_cents", "rollover"}}
	prev := changes.Entity{Name: "budget", Table: "budgets", Fields: []string{"category_id", "limit_cents"}}

	return schema.Migration{
		ID:            "20261122140000_add_budget_rollover_and_overall",
		NoTransaction: true,
		Up: func(s *schema.Schema) error {
			err := s.Rebuild("budgets", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("category_id").Nullable().Constrained("categories").CascadeOnDelete()
				t.BigInteger("limit_cents")
				t.Boolean("rollover").Default(false)
				t.Timestamps()
				syncColumns(t)
				t.UniqueIndex("workspace_id", "category_id").Where("deleted_at IS NULL")
			}, schema.DropTriggers())
			if err != nil {
				return err
			}

			err = s.Raw("CREATE UNIQUE INDEX budgets_overall_unique ON budgets (workspace_id) WHERE category_id IS NULL AND deleted_at IS NULL")
			if err != nil {
				return err
			}

			return changes.InstallTriggers(s, next)
		},
		Down: func(s *schema.Schema) error {
			// An overall budget has no category to go back to.
			if err := s.Raw("DELETE FROM budgets WHERE category_id IS NULL"); err != nil {
				return err
			}

			if err := s.Raw("DROP INDEX IF EXISTS budgets_overall_unique"); err != nil {
				return err
			}

			err := s.Rebuild("budgets", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("category_id").Constrained("categories").CascadeOnDelete()
				t.BigInteger("limit_cents")
				t.Timestamps()
				syncColumns(t)
				t.UniqueIndex("workspace_id", "category_id").Where("deleted_at IS NULL")
			}, schema.DropTriggers(), schema.DropColumns("rollover"))
			if err != nil {
				return err
			}

			return changes.InstallTriggers(s, prev)
		},
	}
}
