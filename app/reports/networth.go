package reports

import (
	"context"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
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
// debt lowers the net worth. The sums come from the monthly summary (whole
// months) and the rows of a month the period only touches; the last point of a period
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

	rows, err := s.Fin.MonthTotals(ctx, finance.FirstSummaryDate, p.To, finance.ByMonth|finance.ByAccount)
	if err != nil {
		return out, err
	}

	slices.SortFunc(rows, func(a, b models.TransactionMonthTotal) int { return strings.Compare(a.Month, b.Month) })

	next := 0

	for _, m := range p.Months() {
		for next < len(rows) && rows[next].Month <= m {
			r := rows[next]
			next++

			if !counted[r.AccountID] {
				continue
			}

			cents := finance.ClampCents(r.Sum())
			if r.Kind != models.KindIncome {
				cents = -cents
			}

			balance = add(balance, cents)
		}

		out.Points = append(out.Points, NetWorthPoint{Month: m, BalanceCents: balance})
	}

	return out, nil
}
