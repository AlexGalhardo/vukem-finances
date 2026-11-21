package migrations

import "github.com/AlexGalhardo/vukem/schema"

// The timezone decides what "today" and "this month" are for the people of a
// workspace (an IANA name, validated by the app); dates stay ISO strings.
// monthly_summary_sent holds one row per workspace and month already mailed.
func m20261121190000AddWorkspaceTimezone() schema.Migration {
	return schema.Migration{
		ID: "20261121190000_add_workspace_timezone",
		Up: func(s *schema.Schema) error {
			if err := s.Table("workspaces", func(t *schema.Table) {
				t.String("timezone", 64).Default("UTC")
			}); err != nil {
				return err
			}

			return s.Create("monthly_summary_sent", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("month", 7)
				t.Timestamps()
				t.UniqueIndex("workspace_id", "month")
			})
		},
		Down: func(s *schema.Schema) error {
			if err := s.DropIfExists("monthly_summary_sent"); err != nil {
				return err
			}

			return s.Table("workspaces", func(t *schema.Table) { t.DropColumn("timezone") })
		},
	}
}
