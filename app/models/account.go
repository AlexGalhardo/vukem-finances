package models

import (
	"database/sql"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// Account kinds. credit_card is reserved for the cards epic and refused.
const (
	AccountChecking   = "checking"
	AccountSavings    = "savings"
	AccountCash       = "cash"
	AccountInvestment = "investment"
	AccountWallet     = "wallet"
)

// AccountKinds lists the kinds an account may have, in the order a picker shows them.
func AccountKinds() []string {
	return []string{AccountChecking, AccountSavings, AccountCash, AccountInvestment, AccountWallet}
}

// Account is a place money sits: a bank account, cash, a wallet. Its balance
// is never stored: it is derived from OpeningCents and the transactions.
type Account struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID  int64  `json:"workspace_id"`
	UID          string `json:"uid"`
	Version      int64  `json:"version" orm:"readonly"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	OpeningCents int64  `json:"opening_cents"`

	// ArchivedAt is set while the account is archived: hidden from pickers and
	// out of the plan quota, history and balance kept.
	ArchivedAt sql.Null[time.Time] `json:"archived_at"`
	Position   int                 `json:"position"`
}
