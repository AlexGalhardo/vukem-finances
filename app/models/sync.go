package models

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Change ops.
const (
	OpUpsert = "upsert"
	OpDelete = "delete"
)

// Change is one line of the per-workspace change log. Database triggers
// write it (see app/changes); it says which row changed and which columns,
// never what they held, so it carries no value that could leak.
type Change struct {
	Seq         int64     `db:"seq"         json:"seq"`
	WorkspaceID int64     `db:"workspace_id" json:"-"`
	Entity      string    `db:"entity"      json:"entity"`
	UID         string    `db:"uid"         json:"uid"`
	Op          string    `db:"op"          json:"op"`
	RowVersion  int64     `db:"row_version" json:"row_version"`
	Fields      string    `db:"fields"      json:"fields"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// PrimaryKey implements orm.Keyer: the log is keyed by seq, not id.
func (Change) PrimaryKey() string { return "seq" }

// Table maps Change to changes.
func (Change) Table() string { return "changes" }

// SyncState holds, per workspace, the highest change seq that compaction
// removed: a client whose cursor is below it must pull a full snapshot.
type SyncState struct {
	orm.Model

	WorkspaceID         int64 `json:"workspace_id"`
	CompactedThroughSeq int64 `json:"compacted_through_seq"`
}

// Table maps SyncState to sync_state.
func (SyncState) Table() string { return "sync_state" }

// Conflict kinds.
const (
	ConflictField         = "field"
	ConflictEditedDeleted = "edited_deleted_row"
)

// SyncConflict keeps a value that lost a merge, so the client can tell the user.
type SyncConflict struct {
	orm.Model

	WorkspaceID  int64  `json:"workspace_id"`
	Entity       string `json:"entity"`
	UID          string `json:"uid"`
	Field        string `json:"field"`
	Kind         string `json:"kind"`
	LostValue    string `json:"lost_value"`
	LostByDevice string `json:"lost_by_device"`
}

// Table maps SyncConflict to sync_conflicts.
func (SyncConflict) Table() string { return "sync_conflicts" }

var _ orm.Keyer = Change{}

// SetTenant implements tenancy.Scoped.
func (m *Change) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Change) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *SyncState) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *SyncState) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *SyncConflict) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *SyncConflict) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
