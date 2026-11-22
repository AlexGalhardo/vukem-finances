package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Notification is one alert for one person of a workspace. TitleKey and
// Params are rendered by the client in its own language: no prose is stored.
// (WorkspaceID, UserID, Kind, SubjectKey) is unique, so the same alert about
// the same subject is made once.
type Notification struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	UserID      int64  `json:"user_id"`
	Kind        string `json:"kind"`
	SubjectKey  string `json:"subject_key"`
	TitleKey    string `json:"title_key"`
	// Params is a JSON object of the values the title needs.
	Params string `json:"params"`
	// InApp says the row shows in the person's list; Email says a mail was
	// wanted, and EmailedAt is the claim made by the run that sends it.
	InApp     bool         `json:"in_app"`
	Email     bool         `json:"email"`
	EmailedAt orm.NullTime `json:"emailed_at"`
	ReadAt    orm.NullTime `json:"read_at"`
}

// NotificationPreference is the channel choice of one person for one kind.
type NotificationPreference struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	UserID      int64  `json:"user_id"`
	Kind        string `json:"kind"`
	InApp       bool   `json:"in_app"`
	Email       bool   `json:"email"`
}

// SetTenant implements tenancy.Scoped.
func (m *Notification) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Notification) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *NotificationPreference) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *NotificationPreference) BeforeCreate(ctx context.Context) error {
	return tenancy.Assign(ctx, m)
}
