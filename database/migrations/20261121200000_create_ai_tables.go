package migrations

import "github.com/AlexGalhardo/vukem/schema"

// The AI layer (epic 15, decision D-007: bring your own key).
//
// ai_credentials holds the key of a workspace encrypted (key_ciphertext, sealed
// with additional data that names the workspace and the row) and only its last
// four characters in clear. A revoked credential keeps its row for the audit
// trail but loses its ciphertext. workspace_ai_settings is one row per
// workspace: AI is off until the owner turns it on. ai_usage counts calls by
// month of the workspace timezone; a row is a reservation until the call ends.
func m20261121200000CreateAITables() schema.Migration {
	return schema.Migration{
		ID: "20261121200000_create_ai_tables",
		Up: func(s *schema.Schema) error {
			steps := []func() error{
				func() error {
					return s.Create("ai_credentials", func(t *schema.Table) {
						t.ID()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.String("provider", 30)
						t.String("label", 80).Default("")
						t.Text("key_ciphertext").Default("")
						t.String("key_hint", 4).Default("")
						t.String("base_url", 500).Default("")
						t.String("default_model", 100).Default("")
						t.BigInteger("created_by")
						t.Timestamp("last_used_at").Nullable()
						t.Timestamp("revoked_at").Nullable()
						t.Timestamps()
						t.Index("workspace_id", "revoked_at")
					})
				},
				func() error {
					return s.Create("workspace_ai_settings", func(t *schema.Table) {
						t.ID()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.Boolean("enabled").Default(false)
						t.Text("features").Default("{}")
						t.Integer("monthly_call_limit").Default(2000)
						t.BigInteger("enabled_by").Default(0)
						t.Timestamp("enabled_at").Nullable()
						t.Timestamps()
						t.UniqueIndex("workspace_id")
					})
				},
				func() error {
					return s.Create("ai_usage", func(t *schema.Table) {
						t.ID()
						t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
						t.String("month", 7)
						t.String("feature", 30)
						t.String("provider", 30)
						t.String("model", 100).Default("")
						t.Integer("input_tokens").Default(0)
						t.Integer("output_tokens").Default(0)
						t.String("status", 20)
						t.Timestamps()
						t.Index("workspace_id", "month")
					})
				},
			}

			for _, step := range steps {
				if err := step(); err != nil {
					return err
				}
			}

			return nil
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"ai_usage", "workspace_ai_settings", "ai_credentials"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
