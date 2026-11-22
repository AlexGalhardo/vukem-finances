package models

import (
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
)

// Budget is the monthly spending limit of one expense category, or, with no
// category, the overall limit of every non-transfer expense of the workspace.
// A budget on a parent category counts the spending of its subcategories.
type Budget struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64           `json:"workspace_id"`
	UID         string          `json:"uid"`
	Version     int64           `json:"version" orm:"readonly"`
	CategoryID  sql.Null[int64] `json:"category_id"`
	LimitCents  int64           `json:"limit_cents"`
	// Rollover adds what the previous month left unspent (never a negative
	// amount) to the amount available this month.
	Rollover bool `json:"rollover"`
}
