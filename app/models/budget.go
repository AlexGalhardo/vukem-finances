package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Budget is the monthly spending limit of one expense category.
type Budget struct {
	orm.Model

	WorkspaceID int64 `json:"workspace_id"`
	CategoryID  int64 `json:"category_id"`
	LimitCents  int64 `json:"limit_cents"`
}
