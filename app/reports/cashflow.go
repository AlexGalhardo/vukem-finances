package reports

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
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

	rows, err := s.ordinaryTotals(ctx, p, finance.ByMonth)
	if err != nil {
		return out, err
	}

	var all bucket

	per := make([]bucket, len(out.Months))

	for _, r := range rows {
		if i, ok := index[r.Month]; ok {
			per[i].add(r)
		}

		all.add(r)
	}

	for i := range out.Months {
		out.Months[i].Sum = per[i].sum()
	}

	out.Totals = all.sum()

	return out, nil
}

// totals sums the period: the numbers Compare sets side by side.
func (s *Service) totals(ctx context.Context, p Period) (Sum, error) {
	var sum Sum

	rows, err := s.ordinaryTotals(ctx, p, 0)
	if err != nil {
		return sum, err
	}

	var all bucket

	for _, r := range rows {
		all.add(r)
	}

	return all.sum(), nil
}
