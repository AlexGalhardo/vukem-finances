package reports

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
)

// MonthFlow is the income and expense of one month ("2026-11").
type MonthFlow struct {
	Month string `json:"month"`
	Sum
}

// CashFlowReport is income against expense per month of the period.
type CashFlowReport struct {
	Currency string      `json:"currency"`
	Period   Period      `json:"period"`
	Months   []MonthFlow `json:"months"`
	Totals   Sum         `json:"totals"`
}

// CashFlow sums income and expense by calendar month. A month the period
// only touches counts the dates inside the period.
func (s *Service) CashFlow(ctx context.Context, p Period) (CashFlowReport, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return CashFlowReport{}, err
	}

	out := CashFlowReport{Currency: ws.Currency, Period: p}

	index := map[string]int{}

	for i, m := range p.Months() {
		index[m] = i
		out.Months = append(out.Months, MonthFlow{Month: m})
	}

	groups, err := s.ordinary(ctx, p).
		GroupBy("occurred_on", "kind", "amount_cents").
		SelectAs("occurred_on", "occurred_on").
		SelectAs("kind", "kind").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[group](ctx)
	if err != nil {
		return out, err
	}

	for _, g := range groups {
		if i, ok := index[g.Date[:7]]; ok {
			out.Months[i].put(g)
		}

		out.Totals.put(g)
	}

	return out, nil
}

// totals sums the period: the numbers Compare sets side by side.
func (s *Service) totals(ctx context.Context, p Period) (Sum, error) {
	var sum Sum

	groups, err := s.ordinary(ctx, p).
		GroupBy("kind", "amount_cents").
		SelectAs("kind", "kind").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[group](ctx)
	if err != nil {
		return sum, err
	}

	for _, g := range groups {
		sum.put(g)
	}

	return sum, nil
}
