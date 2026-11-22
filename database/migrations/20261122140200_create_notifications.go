package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 11: the notifications of a person in a workspace and their per-kind
// channel preferences. Both are server state, not synced. The unique index on
// (workspace_id, user_id, kind, subject_key) is the dedupe key: an alert that
// was already made is never made twice, however many times the evaluation
// runs. in_app and email record the channels chosen when the row was made;
// emailed_at is the claim of the mail, so a retried or concurrent run sends
// it once.
func m20261122140200CreateNotifications() schema.Migration {
	return schema.Migration{
		ID: "20261122140200_create_notifications",
		Up: func(s *schema.Schema) error {
			if err := s.Create("notifications", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("kind", 40)
				t.String("subject_key", 120)
				t.String("title_key", 60)
				t.Text("params").Default("{}")
				t.Boolean("in_app").Default(true)
				t.Boolean("email").Default(false)
				t.Timestamp("emailed_at").Nullable()
				t.Timestamp("read_at").Nullable()
				t.Timestamps()
				t.UniqueIndex("workspace_id", "user_id", "kind", "subject_key")
				t.Index("workspace_id", "user_id", "created_at")
			}); err != nil {
				return err
			}

			return s.Create("notification_preferences", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("kind", 40)
				t.Boolean("in_app")
				t.Boolean("email")
				t.Timestamps()
				t.UniqueIndex("workspace_id", "user_id", "kind")
			})
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"notification_preferences", "notifications"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
