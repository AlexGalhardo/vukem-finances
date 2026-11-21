package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Account is a place money sits: a bank account, a card, cash.
type Account struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID  int64  `json:"workspace_id"`
	UID          string `json:"uid"`
	Version      int64  `json:"version" orm:"readonly"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	OpeningCents int64  `json:"opening_cents"`
}
