package migrations

import (
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// Epic 11: savings goals and their manual contributions, both synced. A
// contribution carries a signed amount (a withdrawal is negative) and never
// moves money between accounts. The entities are written out so the checksum
// never moves with the registry.
func m20261122140100CreateGoals() schema.Migration {
	goals := changes.Entity{
		Name: "goal", Table: "goals",
		Fields: []string{"name", "target_cents", "target_date", "archived_at"},
	}
	contributions := changes.Entity{
		Name: "goal_contribution", Table: "goal_contributions",
		Fields: []string{"goal_id", "amount_cents", "on_date", "note"},
	}

	return schema.Migration{
		ID: "20261122140100_create_goals",
		Up: func(s *schema.Schema) error {
			if err := s.Create("goals", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("name", 120)
				t.BigInteger("target_cents")
				t.String("target_date", 10).Nullable()
				t.Timestamp("archived_at").Nullable()
				t.Timestamps()
				syncColumns(t)
			}); err != nil {
				return err
			}

			if err := s.Create("goal_contributions", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("goal_id").Constrained("goals").CascadeOnDelete()
				t.BigInteger("amount_cents")
				t.String("on_date", 10)
				t.String("note", 200).Default("")
				t.Timestamps()
				syncColumns(t)
				t.Index("workspace_id", "goal_id")
			}); err != nil {
				return err
			}

			for _, e := range []changes.Entity{goals, contributions} {
				if err := changes.InstallTriggers(s, e); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"goal_contributions", "goals"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
