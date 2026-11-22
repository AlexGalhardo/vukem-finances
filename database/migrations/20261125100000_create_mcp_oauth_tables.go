package migrations

import (
	"github.com/AlexGalhardo/vukem/oauth"
	"github.com/AlexGalhardo/vukem/schema"
)

// Epic 33: remote MCP with OAuth. The framework tables (oauth_grants,
// oauth_codes, oauth_refresh_tokens) hold the protocol state; they have no
// workspace_id because a grant is bound to its workspace by mcp_grants, which
// is the tenant table: one row per grant, with the workspace the person chose
// on the consent screen, the person, the client name for the connected-apps
// page and when the grant was last used. The framework tables are on the
// allow-list of the isolation guard for that reason.
func m20261125100000CreateMCPOAuthTables() schema.Migration {
	const id = "20261125100000_create_mcp_oauth_tables"

	base := oauth.Migration(id)

	return schema.Migration{
		ID: id,
		Up: func(s *schema.Schema) error {
			if err := base.Up(s); err != nil {
				return err
			}

			return s.Create("mcp_grants", func(t *schema.Table) {
				t.ID()
				t.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
				t.ForeignID("user_id").Constrained("users").CascadeOnDelete()
				t.String("grant_id", 64).Unique()
				t.String("client_id", 512)
				t.String("client_name", 255)
				t.Timestamp("last_used_at").Nullable()
				t.Timestamps()
				t.Index("workspace_id", "user_id")
			})
		},
		Down: func(s *schema.Schema) error {
			if err := s.Drop("mcp_grants"); err != nil {
				return err
			}

			return base.Down(s)
		},
	}
}
