package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 20: the short codes that pair a phone with a workspace. Only the hash of
// a code is stored; a code is single use (used_at) and lives 10 minutes. The
// table is server state, not synced.
func m20261124090000CreatePairingCodes() schema.Migration {
	return schema.Migration{
		ID: "20261124090000_create_pairing_codes",
		Up: func(s *schema.Schema) error {
			return s.Create("pairing_codes", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("code_hash", 64)
				t.Timestamp("expires_at")
				t.Timestamp("used_at").Nullable()
				t.Timestamps()
				t.UniqueIndex("code_hash")
				t.Index("user_id", "workspace_id")
				t.Index("expires_at")
			})
		},
		Down: func(s *schema.Schema) error { return s.DropIfExists("pairing_codes") },
	}
}
