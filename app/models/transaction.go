package models

import (
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
)

// Transaction kinds.
const (
	KindIncome  = "income"
	KindExpense = "expense"
)

// Transaction is one movement. AmountCents is always positive: Kind says
// whether it adds or subtracts. OccurredOn is an ISO date, "2026-11-12".
type Transaction struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64           `json:"workspace_id"`
	UID         string          `json:"uid"`
	Version     int64           `json:"version" orm:"readonly"`
	AccountID   int64           `json:"account_id"`
	CategoryID  sql.Null[int64] `json:"category_id"`
	Kind        string          `json:"kind"`
	AmountCents int64           `json:"amount_cents"`
	Description string          `json:"description"`
	OccurredOn  string          `json:"occurred_on"`
	ImportID    sql.Null[int64] `json:"-"`
}
