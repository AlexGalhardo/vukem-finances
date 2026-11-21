package migrations

import "github.com/AlexGalhardo/vukem/schema"

// audit_events records who did a sensitive thing in a workspace: roles,
// ownership, members, invitations and settings. The actor is a plain id, not
// a foreign key, so the history outlives a deleted account.
func m20261121190100CreateAuditEventsTable() schema.Migration {
	return schema.Migration{
		ID: "20261121190100_create_audit_events_table",
		Up: func(s *schema.Schema) error {
			return s.Create("audit_events", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.BigInteger("actor_user_id")
				t.String("action", 60)
				t.String("subject")
				t.Timestamps()
				t.Index("workspace_id", "created_at")
			})
		},
		Down: func(s *schema.Schema) error { return s.DropIfExists("audit_events") },
	}
}
