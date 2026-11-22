package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// MCPGrant binds an OAuth grant of the remote MCP endpoint to the workspace the
// person chose when they consented. The protocol state (codes, refresh tokens,
// scopes, revocation) is in the framework tables; this row is what makes the
// grant a thing of one workspace and shows it on the connected-apps page.
type MCPGrant struct {
	orm.Model

	WorkspaceID int64        `json:"workspace_id"`
	UserID      int64        `json:"user_id"`
	GrantID     string       `json:"-"`
	ClientID    string       `json:"client_id"`
	ClientName  string       `json:"client_name"`
	LastUsedAt  orm.NullTime `json:"last_used_at"`
}

// Table maps MCPGrant to mcp_grants.
func (MCPGrant) Table() string { return "mcp_grants" }

// SetTenant implements tenancy.Scoped.
func (m *MCPGrant) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *MCPGrant) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
