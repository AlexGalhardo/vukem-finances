package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// Account is a place money sits: a bank account, a card, cash.
type Account struct {
	orm.Model

	WorkspaceID  int64  `json:"workspace_id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	OpeningCents int64  `json:"opening_cents"`
}
