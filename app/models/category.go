package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Category groups transactions of one kind ("income" or "expense").
type Category struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
}
