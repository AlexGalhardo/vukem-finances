package assistant

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const labelMax = 100

// Figure is one number of a table: a label and an amount in cents.
type Figure struct {
	Label string `json:"label"`
	Cents int64  `json:"cents"`
}

// Table is the result of one query, computed by the server. It is the
// authoritative answer: the figures are never read back from the model.
type Table struct {
	Report string   `json:"report"`
	Title  string   `json:"title"`
	From   string   `json:"from"`
	To     string   `json:"to"`
	Rows   []Figure `json:"rows"`
	// Truncated says rows were cut at MaxRows.
	Truncated bool `json:"truncated"`
}

// Execute runs resolved queries with the report functions, inside the
// workspace of ctx. It never builds a query from the model's text: the only
// inputs are the validated dates, kind and category.
func (s *Service) Execute(ctx context.Context, queries []Resolved) ([]Table, error) {
	tables := make([]Table, 0, len(queries))

	for _, q := range queries {
		t, err := s.run(ctx, q)
		if err != nil {
			return nil, err
		}

		if len(t.Rows) > MaxRows {
			t.Rows, t.Truncated = t.Rows[:MaxRows], true
		}

		tables = append(tables, t)
	}

	return tables, nil
}

func (s *Service) run(ctx context.Context, q Resolved) (Table, error) {
	t := Table{Report: q.Report, Title: q.Describe(), From: q.From, To: q.To, Rows: []Figure{}}

	if q.Report == ReportTransactions {
		return s.transactions(ctx, q, t)
	}

	p, err := reports.NewPeriod(reports.Custom, q.From, q.To, s.Fin.Today(ctx))
	if err != nil {
		return t, ErrInvalidPlan
	}

	switch q.Report {
	case ReportByCategory:
		rep, err := s.Rep.ByCategory(ctx, p, q.Kind)
		if err != nil {
			return t, err
		}

		for _, r := range rep.Rows {
			label := r.Name
			if r.Uncategorized {
				label = "Uncategorized"
			}

			t.Rows = append(t.Rows, Figure{Label: clean(label), Cents: r.Cents})
		}

		t.Rows = append(t.Rows, Figure{Label: "Total", Cents: rep.TotalCents})
	case ReportCashFlow:
		rep, err := s.Rep.CashFlow(ctx, p)
		if err != nil {
			return t, err
		}

		for _, m := range rep.Months {
			t.Rows = append(t.Rows,
				Figure{Label: m.Month + " income", Cents: m.IncomeCents},
				Figure{Label: m.Month + " expense", Cents: m.ExpenseCents},
				Figure{Label: m.Month + " net", Cents: m.NetCents})
		}

		t.Rows = append(t.Rows,
			Figure{Label: "Total income", Cents: rep.Totals.IncomeCents},
			Figure{Label: "Total expense", Cents: rep.Totals.ExpenseCents},
			Figure{Label: "Total net", Cents: rep.Totals.NetCents})
	case ReportByAccount:
		rep, err := s.Rep.ByAccount(ctx, p)
		if err != nil {
			return t, err
		}

		for _, r := range rep.Rows {
			t.Rows = append(t.Rows,
				Figure{Label: clean(r.Name) + " income", Cents: r.IncomeCents},
				Figure{Label: clean(r.Name) + " expense", Cents: r.ExpenseCents})
		}
	case ReportByTag:
		rep, err := s.Rep.ByTag(ctx, p)
		if err != nil {
			return t, err
		}

		for _, r := range rep.Rows {
			t.Rows = append(t.Rows,
				Figure{Label: clean(r.Name) + " income", Cents: r.IncomeCents},
				Figure{Label: clean(r.Name) + " expense", Cents: r.ExpenseCents})
		}
	case ReportNetWorth:
		rep, err := s.Rep.NetWorthSeries(ctx, p)
		if err != nil {
			return t, err
		}

		for _, pt := range rep.Points {
			t.Rows = append(t.Rows, Figure{Label: pt.Month, Cents: pt.BalanceCents})
		}
	case ReportCompare:
		rep, err := s.Rep.Compare(ctx, p)
		if err != nil {
			return t, err
		}

		t.Title += " (previous: " + rep.Previous.From + " to " + rep.Previous.To + ")"

		for _, r := range rep.Rows {
			t.Rows = append(t.Rows,
				Figure{Label: r.Key + ", this period", Cents: r.CurrentCents},
				Figure{Label: r.Key + ", previous period", Cents: r.PreviousCents},
				Figure{Label: r.Key + ", difference", Cents: r.DiffCents})
		}
	}

	return t, nil
}

// transactions lists the newest rows of the range, never a transfer leg. The
// description is redacted and clipped: it goes to the model with the figures.
func (s *Service) transactions(ctx context.Context, q Resolved, t Table) (Table, error) {
	query := workspaces.Scoped[models.Transaction](ctx, s.DB).WhereNull("transfer_uid").
		WhereBetween("occurred_on", q.From, q.To)

	if q.Kind != "" {
		query = query.Where("kind", q.Kind)
	}

	if q.CategoryID > 0 {
		kids, err := workspaces.Scoped[models.Category](ctx, s.DB).Where("parent_id", q.CategoryID).Get(ctx)
		if err != nil {
			return t, err
		}

		ids := []any{q.CategoryID}
		for _, k := range kids {
			ids = append(ids, k.ID)
		}

		query = query.WhereIn("category_id", ids...)
	}

	rows, err := query.OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).Limit(q.Limit).Get(ctx)
	if err != nil {
		return t, err
	}

	for _, r := range rows {
		cents := r.AmountCents
		if r.Kind == models.KindExpense {
			cents = -cents
		}

		t.Rows = append(t.Rows, Figure{Label: r.OccurredOn + " " + clean(r.Description), Cents: cents})
	}

	return t, nil
}

// clean makes text fit to send to the model: control characters out, redacted,
// clipped. It is applied to every name and description of a result.
func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, s)

	s = strings.Join(strings.Fields(ai.Redact(s)), " ")
	if utf8.RuneCountInString(s) > labelMax {
		s = string([]rune(s)[:labelMax])
	}

	return s
}
