package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 16: AI categorization and the finance assistant. All four tables are
// server state, never synced and never exported.
//
//   - description_rules is the memory of the workspace: a normalized
//     description (key) and the category the people of the workspace chose for
//     it. It uses only the workspace's own data and works with AI off.
//   - suggestions holds a proposed category for a transaction, by uid. Status
//     is pending, accepted or rejected; applied_at marks an auto-applied one
//     (reversible for 7 days) and undone_at the reversal. One row per
//     transaction, so a rejected suggestion is never proposed again.
//   - assistant_conversations and assistant_messages are private to the user
//     who asked, inside the workspace, and are kept 90 days.
//
// workspace_ai_settings gets the opt-in switch that applies confident
// suggestions without a review; it is off by default.
func m20261124163000CreateAIAssistantTables() schema.Migration {
	return schema.Migration{
		ID: "20261124163000_create_ai_assistant_tables",
		Up: func(s *schema.Schema) error {
			if err := s.Create("description_rules", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("key", 120)
				t.ForeignID("category_id").Constrained("categories").CascadeOnDelete()
				t.Integer("hits").Default(1)
				t.String("source", 24)
				t.Timestamps()
				t.UniqueIndex("workspace_id", "key")
			}); err != nil {
				return err
			}

			if err := s.Create("suggestions", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("transaction_uid", 26)
				t.String("category_uid", 26)
				t.Integer("confidence").Default(0)
				t.String("source", 8)
				t.String("status", 12).Default("pending")
				t.Timestamp("applied_at").Nullable()
				t.Timestamp("undone_at").Nullable()
				t.Timestamps()
				t.UniqueIndex("workspace_id", "transaction_uid")
				t.Index("workspace_id", "status")
			}); err != nil {
				return err
			}

			if err := s.Create("assistant_conversations", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("title", 120).Default("")
				t.Timestamps()
				t.Index("workspace_id", "user_id", "updated_at")
			}); err != nil {
				return err
			}

			if err := s.Create("assistant_messages", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("conversation_id").Constrained("assistant_conversations").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("role", 12)
				t.Text("content")
				t.Text("data").Default("{}")
				t.Timestamps()
				t.Index("workspace_id", "conversation_id")
				t.Index("workspace_id", "user_id", "created_at")
			}); err != nil {
				return err
			}

			return s.Table("workspace_ai_settings", func(t *schema.Table) {
				t.Boolean("auto_apply_suggestions").Default(false)
			})
		},
		Down: func(s *schema.Schema) error {
			if err := s.Table("workspace_ai_settings", func(t *schema.Table) { t.DropColumn("auto_apply_suggestions") }); err != nil {
				return err
			}

			for _, table := range []string{"assistant_messages", "assistant_conversations", "suggestions", "description_rules"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
