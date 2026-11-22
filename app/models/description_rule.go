package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Sources of a description rule.
const (
	RuleSourceUser               = "user"
	RuleSourceAcceptedSuggestion = "accepted_suggestion"
)

// DescriptionRule remembers the category the workspace chose for a normalized
// description (see finance.NormalizeDescription). It is learned from the
// workspace's own corrections, never leaves the server and never syncs.
type DescriptionRule struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	Key         string `json:"key" db:"key"`
	CategoryID  int64  `json:"category_id"`
	Hits        int    `json:"hits"`
	Source      string `json:"source"`
}

// Table maps DescriptionRule to description_rules.
func (DescriptionRule) Table() string { return "description_rules" }

// SetTenant implements tenancy.Scoped.
func (m *DescriptionRule) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *DescriptionRule) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
