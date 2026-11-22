package models

import (
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
)

// Roles of a workspace member. The owner pays, the admin manages people
// and data, the member records transactions, the viewer only reads.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Workspace is the tenant: it owns every account, transaction and budget,
// and it is what Stripe bills. Money is int64 cents in Currency (ISO 4217).
type Workspace struct {
	orm.Model

	Name             string           `json:"name"`
	Currency         string           `json:"currency"`
	Timezone         string           `json:"timezone"`
	StripeCustomerID sql.Null[string] `json:"-"`

	// State is StateActive or StatePending (an empty value, a row made before
	// epic 34, is active). PurgeAt is when the purge job removes a pending
	// workspace; PurgeError is the reason the last purge attempt stopped.
	State               string       `json:"-"`
	PurgeAt             orm.NullTime `json:"-"`
	DeletionRequestedAt orm.NullTime `json:"-"`
	DeletionRequestedBy int64        `json:"-"`
	PurgeError          string       `json:"-"`
}

// States of a workspace.
const (
	StateActive          = "active"
	StatePendingDeletion = "pending_deletion"
)

// PendingDeletion reports whether the workspace is in its grace period.
func (w Workspace) PendingDeletion() bool { return w.State == StatePendingDeletion }

// MonthlySummarySent records that the summary of Month ("2026-11") went out
// for the workspace, so a missed or doubled scheduler run is harmless.
type MonthlySummarySent struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	Month       string `json:"month"`
}

// Table maps MonthlySummarySent to monthly_summary_sent.
func (MonthlySummarySent) Table() string { return "monthly_summary_sent" }

// AuditEvent records who did a sensitive thing in a workspace.
type AuditEvent struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	ActorUserID int64  `json:"actor_user_id"`
	Action      string `json:"action"`
	Subject     string `json:"subject"`
}

// Membership puts a user in a workspace with a role.
type Membership struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	UserID      int64  `json:"user_id"`
	Role        string `json:"role"`
}

// Invite is a pending invitation by email; only the SHA-256 of its token is stored.
type Invite struct {
	orm.Model

	WorkspaceID int64        `json:"workspace_id"`
	Email       string       `json:"email"`
	Role        string       `json:"role"`
	TokenHash   string       `json:"-"`
	ExpiresAt   orm.NullTime `json:"expires_at"`
	AcceptedAt  orm.NullTime `json:"accepted_at"`
}

// EmailSuppression is an address that bounced or complained: it gets no
// more mail (mail.Mailer.SetSuppression, filled by the Resend webhook).
type EmailSuppression struct {
	orm.Model

	Address string `json:"address"`
	Reason  string `json:"reason"`
}
