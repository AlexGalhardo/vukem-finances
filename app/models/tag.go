package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Tag limits.
const (
	// MaxTagsPerTransaction is how many tags one transaction may carry.
	MaxTagsPerTransaction = 10
	// MaxTagsPerWorkspace is how many live tags a workspace may hold.
	MaxTagsPerWorkspace = 200
	// MaxTagNameLength is the longest tag name, in characters.
	MaxTagNameLength = 40
)

// Tag is a free label of a workspace. Names are unique per workspace without
// regard to case.
type Tag struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	Name        string `json:"name"`
	Color       string `json:"color"`
}

// TransactionTag attaches a tag to a transaction. It is not synced: the wire
// carries the same fact as the tag_uids field of the transaction, which the
// service keeps equal to these rows in the same database transaction.
type TransactionTag struct {
	orm.Model

	WorkspaceID   int64 `json:"workspace_id"`
	TransactionID int64 `json:"transaction_id"`
	TagID         int64 `json:"tag_id"`
}
