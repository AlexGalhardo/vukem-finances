package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// AICredential is the key of a workspace for one AI provider. KeyCiphertext
// never leaves the server: it is not in JSON, and the key opens only with the
// workspace and row ids as additional data (app/ai/credentials.go).
type AICredential struct {
	orm.Model

	WorkspaceID   int64        `json:"workspace_id"`
	Provider      string       `json:"provider"`
	Label         string       `json:"label"`
	KeyCiphertext string       `json:"-"`
	KeyHint       string       `json:"key_hint"`
	BaseURL       string       `json:"base_url"`
	DefaultModel  string       `json:"default_model"`
	CreatedBy     int64        `json:"created_by"`
	LastUsedAt    orm.NullTime `json:"last_used_at"`
	RevokedAt     orm.NullTime `json:"revoked_at"`
}

// Table maps AICredential to ai_credentials.
func (AICredential) Table() string { return "ai_credentials" }

// WorkspaceAISettings is whether AI is on for a workspace, which features may
// use it and the monthly call cap. Features is JSON: {"categorize":true}.
type WorkspaceAISettings struct {
	orm.Model

	WorkspaceID      int64        `json:"workspace_id"`
	Enabled          bool         `json:"enabled"`
	Features         string       `json:"features"`
	MonthlyCallLimit int          `json:"monthly_call_limit"`
	EnabledBy        int64        `json:"enabled_by"`
	EnabledAt        orm.NullTime `json:"enabled_at"`
	// AutoApplySuggestions applies category suggestions of at least 90 % confidence
	// without a review; off by default.
	AutoApplySuggestions bool `json:"auto_apply_suggestions"`
}

// Table maps WorkspaceAISettings to workspace_ai_settings.
func (WorkspaceAISettings) Table() string { return "workspace_ai_settings" }

// AIUsage is one AI call: a reservation while it runs, then ok or failed.
type AIUsage struct {
	orm.Model

	WorkspaceID  int64  `json:"workspace_id"`
	Month        string `json:"month"`
	Feature      string `json:"feature"`
	Provider     string `json:"provider"`
	AIModel      string `json:"model" db:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Status       string `json:"status"`
}

// Table maps AIUsage to ai_usage.
func (AIUsage) Table() string { return "ai_usage" }

// SetTenant implements tenancy.Scoped.
func (m *AICredential) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *AICredential) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *WorkspaceAISettings) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *WorkspaceAISettings) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *AIUsage) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *AIUsage) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
