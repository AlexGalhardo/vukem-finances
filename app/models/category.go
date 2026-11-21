package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Category groups transactions of one kind ("income" or "expense").
type Category struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
}
