package api

import (
	"time"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// The wire format. Money is an integer *_cents in the workspace currency, a
// date is YYYY-MM-DD, a timestamp is RFC 3339 in UTC and a row is named by its
// uid; the numeric ids of the database never leave the server.

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func stampNull(t orm.NullTime) *string {
	if !t.Valid {
		return nil
	}

	s := stamp(t.V)

	return &s
}

// List is the answer of a list that comes whole, and of a page of a paged list
// (with meta and links).
type List[R any] = vukemapi.Envelope[R]

// WorkspaceOut is the workspace a token works in.
type WorkspaceOut struct {
	ID       int64  `json:"id"       example:"7"`
	Name     string `json:"name"     example:"Home"`
	Currency string `json:"currency" example:"USD"`
	Timezone string `json:"timezone" example:"America/Sao_Paulo"`
}

// MeOut is the token that made the request.
type MeOut struct {
	TokenName string       `json:"token_name" example:"ci"`
	Scopes    []string     `json:"scopes"`
	ExpiresAt *string      `json:"expires_at" example:"2027-02-19T12:00:00Z"`
	Role      string       `json:"role"       example:"member"`
	Workspace WorkspaceOut `json:"workspace"`
}

// AccountOut is an account with its balance as of today.
type AccountOut = operations.AccountOut

// CategoryOut is a category.
type CategoryOut = operations.CategoryOut

// TagOut is a tag.
type TagOut = operations.TagOut

// TagRefOut is a tag on a transaction.
type TagRefOut = operations.TagRefOut

// TransactionOut is a transaction. A transfer leg carries the transfer_uid
// both legs share.
type TransactionOut = operations.TransactionOut

// TransferOut is a transfer: the expense leg on the source account and the
// income leg on the destination.
type TransferOut = operations.TransferOut

// BudgetOut is a budget with what the month spent against it.
type BudgetOut = operations.BudgetOut

// SummaryAccountOut is the balance of one account in the summary.
type SummaryAccountOut = operations.SummaryAccountOut

// SummaryOut is a month of the workspace. Transfers are neither income nor
// expense; the balances are as of today in the workspace timezone.
type SummaryOut = operations.SummaryOut
