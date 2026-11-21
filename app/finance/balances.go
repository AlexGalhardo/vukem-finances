package finance

import (
	"context"
	"errors"
	"math"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// errBalanceRange answers a sum that no longer fits an int64.
var errBalanceRange = errors.New("finance: a balance is beyond the int64 range")

// AccountBalance is the balance of one account at a date. It is derived from
// the opening balance and the transactions, never stored.
type AccountBalance struct {
	AccountID     int64 `json:"accountId"`
	OpeningCents  int64 `json:"openingCents"`
	MovementCents int64 `json:"movementCents"`
	BalanceCents  int64 `json:"balanceCents"`
	Archived      bool  `json:"archived"`
	// Transactions counts the live transactions of the account, at any
	// date: what deleting the account would remove.
	Transactions int64 `json:"transactions"`
}

// movement is the signed sum of the transactions of one account and kind.
type movement struct {
	AccountID int64  `db:"account_id"`
	Kind      string `db:"kind"`
	Cents     int64  `db:"cents"`
}

func addCents(a, b int64) (int64, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, errBalanceRange
	}

	return a + b, nil
}

// AccountBalances returns one balance per live account (archived ones
// included, flagged) up to asOf, an ISO date taken inclusively; empty means
// today in the workspace timezone. Both legs of a transfer count, each on its
// own account: a transfer moves money between balances and changes none of
// the totals. The sums come from one grouped statement.
func (s *Service) AccountBalances(ctx context.Context, asOf string) ([]AccountBalance, error) {
	if asOf == "" {
		asOf = s.Today(ctx)
	}

	if !validDate(asOf) {
		return nil, errDate
	}

	accounts, err := s.Accounts(ctx)
	if err != nil {
		return nil, err
	}

	moves, err := workspaces.Scoped[models.Transaction](ctx, s.DB).
		Where("occurred_on", "<=", asOf).
		GroupBy("account_id", "kind").
		SelectAs("account_id", "account_id").
		SelectAs("kind", "kind").
		SelectExpr(orm.Sum("amount_cents"), "cents").
		Scan[movement](ctx)
	if err != nil {
		return nil, err
	}

	counts, err := workspaces.Scoped[models.Transaction](ctx, s.DB).GroupCount[int64](ctx, "account_id")
	if err != nil {
		return nil, err
	}

	signed := make(map[int64]int64, len(accounts))

	for _, m := range moves {
		delta := m.Cents
		if m.Kind != models.KindIncome {
			delta = -delta
		}

		if signed[m.AccountID], err = addCents(signed[m.AccountID], delta); err != nil {
			return nil, err
		}
	}

	out := make([]AccountBalance, len(accounts))

	for i, a := range accounts {
		balance, err := addCents(a.OpeningCents, signed[a.ID])
		if err != nil {
			return nil, err
		}

		out[i] = AccountBalance{
			AccountID: a.ID, OpeningCents: a.OpeningCents, MovementCents: signed[a.ID],
			BalanceCents: balance, Archived: a.ArchivedAt.Valid, Transactions: counts[a.ID],
		}
	}

	return out, nil
}

// NetWorth is the sum of the balances of the accounts that are not archived
// at asOf (empty means today). The dashboard shows this same number.
func (s *Service) NetWorth(ctx context.Context, asOf string) (int64, error) {
	balances, err := s.AccountBalances(ctx, asOf)
	if err != nil {
		return 0, err
	}

	return NetWorthOf(balances)
}

// NetWorthOf sums the balances that are not archived: the one rule behind the
// accounts page and the dashboard.
func NetWorthOf(balances []AccountBalance) (int64, error) {
	var total int64

	var err error

	for _, b := range balances {
		if b.Archived {
			continue
		}

		if total, err = addCents(total, b.BalanceCents); err != nil {
			return 0, err
		}
	}

	return total, nil
}
