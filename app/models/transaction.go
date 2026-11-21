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

	// StatementRef is the closing date (ISO) of the card statement the row
	// belongs to: set on a purchase made on a credit card account and on the
	// card leg of a statement payment, null everywhere else.
	StatementRef sql.Null[string] `json:"statement_ref"`

	// RecurrenceUID and OccurrenceOn are the natural key of a transaction
	// generated from a recurrence; both are null on any other row.
	RecurrenceUID sql.Null[string] `json:"recurrence_uid"`
	OccurrenceOn  sql.Null[string] `json:"occurrence_on"`

	// InstallmentGroupUID ties the parts of an installment purchase;
	// InstallmentNo (1-based) and InstallmentTotal say which part this is.
	InstallmentGroupUID sql.Null[string] `json:"installment_group_uid"`
	InstallmentNo       sql.Null[int64]  `json:"installment_no"`
	InstallmentTotal    sql.Null[int64]  `json:"installment_total"`

	// ImportHash is the duplicate key of an imported row; server only.
	ImportHash sql.Null[string] `json:"-"`
}
