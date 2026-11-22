package reports

import "context"

// Comparison keys.
const (
	CompareIncome  = "income"
	CompareExpense = "expense"
	CompareNet     = "net"
)

// CompareRow sets one total next to the same total of the previous period.
// DiffCents is Current - Previous. ChangeBps is the difference as a
// percentage of the previous value in basis points (1250 is 12.50 %),
// computed with integers and rounded half away from zero; NotApplicable is
// set, and ChangeBps is 0, when the previous value is zero.
type CompareRow struct {
	Key           string `json:"key"`
	CurrentCents  int64  `json:"currentCents"`
	PreviousCents int64  `json:"previousCents"`
	DiffCents     int64  `json:"diffCents"`
	ChangeBps     int64  `json:"changeBps"`
	NotApplicable bool   `json:"notApplicable"`
}

// CompareReport compares a period with the one right before it.
type CompareReport struct {
	Currency string       `json:"currency"`
	Period   Period       `json:"period"`
	Previous Period       `json:"previous"`
	Rows     []CompareRow `json:"rows"`
}

// changeBps is the change from previous to current in basis points; ok is
// false when previous is zero.
func changeBps(current, previous int64) (int64, bool) {
	if previous == 0 {
		return 0, false
	}

	diff := current - previous
	mag := previous

	if mag < 0 {
		mag = -mag
	}

	if diff < 0 {
		return -ratio(-diff, mag), true
	}

	return ratio(diff, mag), true
}

func compareRow(key string, current, previous int64) CompareRow {
	r := CompareRow{Key: key, CurrentCents: current, PreviousCents: previous, DiffCents: current - previous}
	bp, ok := changeBps(current, previous)
	r.ChangeBps, r.NotApplicable = bp, !ok

	return r
}

// Compare sets the income, expense and net total of the period next to those
// of the previous period (same length; for whole months, the previous months).
func (s *Service) Compare(ctx context.Context, p Period) (CompareReport, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return CompareReport{}, err
	}

	prev := p.Previous()
	out := CompareReport{Currency: ws.Currency, Period: p, Previous: prev}

	cur, err := s.totals(ctx, p)
	if err != nil {
		return out, err
	}

	old, err := s.totals(ctx, prev)
	if err != nil {
		return out, err
	}

	out.Rows = []CompareRow{
		compareRow(CompareIncome, cur.IncomeCents, old.IncomeCents),
		compareRow(CompareExpense, cur.ExpenseCents, old.ExpenseCents),
		compareRow(CompareNet, cur.NetCents, old.NetCents),
	}

	return out, nil
}
