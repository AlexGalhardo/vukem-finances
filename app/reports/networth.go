package reports

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// NetWorthPoint is the net worth at the end of a month ("2026-11").
type NetWorthPoint struct {
	Month        string `json:"month"`
	BalanceCents int64  `json:"balanceCents"`
}

// NetWorthReport is the trend of the net worth over the months of a period.
type NetWorthReport struct {
	Currency string          `json:"currency"`
	Period   Period          `json:"period"`
	Points   []NetWorthPoint `json:"points"`
}

// NetWorthSeries is the balance of all the accounts that are not archived at
// the last day of every month of the period: opening balances plus every
// transaction up to that day, transfer legs included (they only move money
// between accounts). A credit card account is a negative balance, so card
// debt lowers the net worth. The transactions are read once, in date order,
// and the months are cut from that single pass; the last point of a period
// that reaches into the future includes the future-dated rows of its month.
func (s *Service) NetWorthSeries(ctx context.Context, p Period) (NetWorthReport, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return NetWorthReport{}, err
	}

	out := NetWorthReport{Currency: ws.Currency, Period: p, Points: []NetWorthPoint{}}

	accounts, err := workspaces.Scoped[models.Account](ctx, s.DB).Get(ctx)
	if err != nil {
		return out, err
	}

	counted := make(map[int64]bool, len(accounts))

	var balance int64

	for _, a := range accounts {
		if a.ArchivedAt.Valid {
			continue
		}

		counted[a.ID] = true
		balance = add(balance, a.OpeningCents)
	}

	groups, err := workspaces.Scoped[models.Transaction](ctx, s.DB).
		Where("occurred_on", "<=", p.To).
		GroupBy("occurred_on", "account_id", "kind", "amount_cents").
		SelectAs("occurred_on", "occurred_on").
		SelectAs("account_id", "account_id").
		SelectAs("kind", "kind").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		OrderBy("occurred_on", orm.Asc).
		Scan[group](ctx)
	if err != nil {
		return out, err
	}

	next := 0

	for _, m := range p.Months() {
		first, _ := time.Parse("2006-01", m)
		cut := monthEnd(first).Format(isoDate)

		for next < len(groups) && groups[next].Date <= cut {
			g := groups[next]
			next++

			if !counted[g.AccountID] {
				continue
			}

			cents := mul(g.Amount, g.N)
			if g.Kind != models.KindIncome {
				cents = -cents
			}

			balance = add(balance, cents)
		}

		out.Points = append(out.Points, NetWorthPoint{Month: m, BalanceCents: balance})
	}

	return out, nil
}
