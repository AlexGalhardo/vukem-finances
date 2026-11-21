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

	// TransferUID is the ULID both legs of a transfer share; null on an
	// ordinary income or expense.
	TransferUID sql.Null[string] `json:"transfer_uid"`

	// TagUIDs is a JSON array of the uids of the tags of the transaction: the
	// synced image of the transaction_tags rows, written with them.
	TagUIDs string `db:"tag_uids" json:"tag_uids"`

	// ImportHash is the duplicate key of an imported row; server only.
	ImportHash sql.Null[string] `json:"-"`
}
