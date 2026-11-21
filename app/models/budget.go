package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Budget is the monthly spending limit of one expense category.
type Budget struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	CategoryID  int64  `json:"category_id"`
	LimitCents  int64  `json:"limit_cents"`
}
