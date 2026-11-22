package migrations

import "github.com/AlexGalhardo/vukem/schema"

// Epic 33: mcp_grants keeps the scopes the server approved when the person
// consented (what their role could grant at that moment), next to the scopes the
// framework stored with the grant. A grant whose scopes are not inside the
// approved set is not found, so no later change widens it.
func m20261125110000AddMCPGrantScopes() schema.Migration {
	return schema.Migration{
		ID: "20261125110000_add_mcp_grant_scopes",
		Up: func(s *schema.Schema) error {
			return s.Table("mcp_grants", func(t *schema.Table) {
				t.String("scopes", 1024).Default("")
			})
		},
		Down: func(s *schema.Schema) error {
			return s.Table("mcp_grants", func(t *schema.Table) {
				t.DropColumn("scopes")
			})
		},
	}
}
