package models

import (
	"context"
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Recurrence is the rule of a transaction that repeats (rent, salary, a
// subscription): docs/specs/recurrence.md. Its transactions are generated
// from StartDate on and carry its UID as recurrence_uid.
type Recurrence struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64            `json:"workspace_id"`
	UID         string           `json:"uid"`
	Version     int64            `json:"version" orm:"readonly"`
	AccountID   int64            `json:"account_id"`
	CategoryID  sql.Null[int64]  `json:"category_id"`
	Kind        string           `json:"kind"`
	AmountCents int64            `json:"amount_cents"`
	Description string           `json:"description"`
	Frequency   string           `json:"frequency"`
	Interval    int              `json:"interval"`
	StartDate   string           `json:"start_date"`
	EndDate     sql.Null[string] `json:"end_date"`
	MaxCount    sql.Null[int64]  `json:"max_count"`

	// GeneratedThrough is the watermark: occurrences up to this date were
	// considered, generated or deleted by the person.
	GeneratedThrough sql.Null[string] `json:"generated_through"`
	Active           bool             `json:"active"`

	// LastError is the last failure of a generation run (server state, not synced).
	LastError sql.Null[string] `json:"last_error"`
}

// SetTenant implements tenancy.Scoped.
func (m *Recurrence) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the uid.
func (m *Recurrence) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.AmountCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}
