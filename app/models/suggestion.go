package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Where a suggestion came from.
const (
	SuggestionRule = "rule"
	SuggestionAI   = "ai"
)

// Suggestion statuses.
const (
	SuggestionPending  = "pending"
	SuggestionAccepted = "accepted"
	SuggestionRejected = "rejected"
)

// Suggestion is a proposed category for a transaction, both by uid. Confidence
// is a whole percent, 0 to 100. AppliedAt is set on a suggestion applied
// without a review (the opt-in switch), which stays reversible until UndoneAt.
type Suggestion struct {
	orm.Model

	WorkspaceID    int64        `json:"workspace_id"`
	TransactionUID string       `json:"transaction_uid"`
	CategoryUID    string       `json:"category_uid"`
	Confidence     int          `json:"confidence"`
	Source         string       `json:"source"`
	Status         string       `json:"status"`
	AppliedAt      orm.NullTime `json:"applied_at"`
	UndoneAt       orm.NullTime `json:"undone_at"`
}

// Table maps Suggestion to suggestions.
func (Suggestion) Table() string { return "suggestions" }

// SetTenant implements tenancy.Scoped.
func (m *Suggestion) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Suggestion) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
