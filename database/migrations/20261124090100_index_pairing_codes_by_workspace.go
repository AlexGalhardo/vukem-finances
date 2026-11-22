package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 20: the isolation guard wants every workspace table to have an index that
// starts with workspace_id, so a query scoped to a workspace never scans the table.
func m20261124090100IndexPairingCodesByWorkspace() schema.Migration {
	return schema.Migration{
		ID: "20261124090100_index_pairing_codes_by_workspace",
		Up: func(s *schema.Schema) error {
			return s.Table("pairing_codes", func(t *schema.Table) {
				t.Index("workspace_id", "expires_at")
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("pairing_codes", func(t *schema.Table) {
				t.DropIndex("pairing_codes_workspace_id_expires_at_index")
			})
		},
	}
}
