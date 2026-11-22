package models

import (
	"context"
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Goal is a savings goal: a target amount and, optionally, a date to reach it
// by. Its progress is the sum of its contributions, which are manual notes and
// never move money between accounts.
type Goal struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64            `json:"workspace_id"`
	UID         string           `json:"uid"`
	Version     int64            `json:"version" orm:"readonly"`
	Name        string           `json:"name"`
	TargetCents int64            `json:"target_cents"`
	TargetDate  sql.Null[string] `json:"target_date"`
	ArchivedAt  orm.NullTime     `json:"archived_at"`
}

// GoalContribution is an amount put towards a goal on a date. A negative
// amount is a withdrawal.
type GoalContribution struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	GoalID      int64  `json:"goal_id"`
	AmountCents int64  `json:"amount_cents"`
	OnDate      string `json:"on_date"`
	Note        string `json:"note"`
}

// SetTenant implements tenancy.Scoped.
func (m *Goal) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the uid.
func (m *Goal) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.TargetCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *GoalContribution) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the uid.
func (m *GoalContribution) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.AmountCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}
