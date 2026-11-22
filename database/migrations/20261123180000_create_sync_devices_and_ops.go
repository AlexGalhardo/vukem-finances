package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 18: the devices that sync a workspace and the results of the pushes
// they made. A device is bound to the API token that registered it; its
// device_id is the ULID a client sends in X-Device-Id. sync_ops keeps the
// result of every pushed op_id for 30 days, so a retry returns the stored
// answer and changes nothing. Both tables are server state, not synced.
func m20261123180000CreateSyncDevicesAndOps() schema.Migration {
	return schema.Migration{
		ID: "20261123180000_create_sync_devices_and_ops",
		Up: func(s *schema.Schema) error {
			if err := s.Create("sync_devices", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("token_id")
				t.String("device_id", 26)
				t.String("name", 80)
				t.String("platform", 20)
				t.Timestamp("last_seen_at").Nullable()
				t.BigInteger("last_pull_seq").Default(0)
				t.Timestamp("revoked_at").Nullable()
				t.Timestamps()
				t.UniqueIndex("workspace_id", "device_id")
				t.Index("workspace_id", "user_id")
				t.Index("token_id")
			}); err != nil {
				return err
			}

			return s.Create("sync_ops", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.String("op_id", 26)
				t.String("device_id", 26)
				t.Text("result")
				t.Timestamps()
				t.UniqueIndex("workspace_id", "op_id")
				t.Index("workspace_id", "device_id", "created_at")
				t.Index("created_at")
			})
		},
		Down: func(s *schema.Schema) error {
			for _, table := range []string{"sync_ops", "sync_devices"} {
				if err := s.DropIfExists(table); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
