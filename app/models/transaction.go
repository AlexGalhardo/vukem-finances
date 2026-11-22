package models

import (
	"database/sql"
	"math/big"

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

// TransactionMonthTotal is one bucket of the per-workspace summary that
// triggers keep over the live transactions (QA-503): the count and the cents
// of a month, account, category (0 for none), kind and transfer flag. The
// cents are split in AmountHi (cents >> 32) and AmountLo (the rest, below
// 2^32) so a sum never overflows; Cents joins them. The table is written by
// the database only: the app reads it.
type TransactionMonthTotal struct {
	ID          int64  `db:"id"           json:"-"`
	WorkspaceID int64  `db:"workspace_id" json:"-"`
	Month       string `db:"month"        json:"month"`
	AccountID   int64  `db:"account_id"   json:"account_id"`
	CategoryID  int64  `db:"category_id"  json:"category_id"`
	Kind        string `db:"kind"         json:"kind"`
	Transfer    int64  `db:"transfer"     json:"transfer"`
	N           int64  `db:"n"            json:"n"`
	AmountHi    int64  `db:"amount_hi"    json:"-"`
	AmountLo    int64  `db:"amount_lo"    json:"-"`
}

// Table maps TransactionMonthTotal to transaction_month_totals.
func (TransactionMonthTotal) Table() string { return "transaction_month_totals" }

// Sum is the exact cents of the bucket: AmountHi * 2^32 + AmountLo.
func (t TransactionMonthTotal) Sum() *big.Int {
	v := new(big.Int).Lsh(big.NewInt(t.AmountHi), 32)

	return v.Add(v, big.NewInt(t.AmountLo))
}
