package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Device platforms.
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
	PlatformOther   = "other"
)

// DevicePlatforms lists the platforms a device may name.
func DevicePlatforms() []string {
	return []string{PlatformIOS, PlatformAndroid, PlatformWeb, PlatformOther}
}

// SyncDevice is a phone (or any client) that syncs a workspace. It belongs to
// the API token that registered it: DeviceID is the ULID the client sends in
// X-Device-Id, and a revoked device or a revoked token stops syncing.
type SyncDevice struct {
	orm.Model

	WorkspaceID int64        `json:"workspace_id"`
	UserID      int64        `json:"user_id"`
	TokenID     string       `json:"-"`
	DeviceID    string       `json:"device_id"`
	Name        string       `json:"name"`
	Platform    string       `json:"platform"`
	LastSeenAt  orm.NullTime `json:"last_seen_at"`
	LastPullSeq int64        `json:"last_pull_seq"`
	RevokedAt   orm.NullTime `json:"revoked_at"`
}

// Table maps SyncDevice to sync_devices.
func (SyncDevice) Table() string { return "sync_devices" }

// SetTenant implements tenancy.Scoped.
func (m *SyncDevice) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *SyncDevice) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SyncOp is the stored result of one pushed operation, kept for 30 days: a
// retried op_id returns Result unchanged.
type SyncOp struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	OpID        string `json:"op_id"`
	DeviceID    string `json:"device_id"`
	Result      string `json:"result"`
}

// Table maps SyncOp to sync_ops.
func (SyncOp) Table() string { return "sync_ops" }

// SetTenant implements tenancy.Scoped.
func (m *SyncOp) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *SyncOp) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// PairingCode is the hash of a short code that pairs a phone with a workspace:
// single use (UsedAt) and valid for 10 minutes. The code itself is never stored.
// app/sync/pairing.go reads and writes the table with SQL, in one transaction.
type PairingCode struct {
	orm.Model

	WorkspaceID int64        `json:"workspace_id"`
	UserID      int64        `json:"user_id"`
	CodeHash    string       `json:"-"`
	ExpiresAt   orm.NullTime `json:"expires_at"`
	UsedAt      orm.NullTime `json:"used_at"`
}

// Table maps PairingCode to pairing_codes.
func (PairingCode) Table() string { return "pairing_codes" }

// SetTenant implements tenancy.Scoped.
func (m *PairingCode) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *PairingCode) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
