package assistant

import (
	"context"
	"regexp"
	"strconv"
	"strings"

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

	// model is what the provider sees of Rows: the same figures, with the names
	// of accounts and tags replaced by opaque labels ("Account 1", "Tag 2").
	// It is never serialized or stored.
	model []Figure
	book  *aliasBook
}

// aliasBook hands out the opaque labels of one question: the same name gets
// the same label in every table, and the labels map back to names locally.
type aliasBook struct {
	n    map[string]int
	real map[string]string
}

func newBook() *aliasBook { return &aliasBook{n: map[string]int{}, real: map[string]string{}} }

// label returns "<noun> <n>" for name, numbering in order of first sight.
func (b *aliasBook) label(noun, name string) string {
	key := noun + "|" + name

	i, ok := b.n[key]
	if !ok {
		i = 1
		for k := range b.n {
			if strings.HasPrefix(k, noun+"|") {
				i++
			}
		}

		b.n[key] = i
	}

	alias := noun + " " + strconv.Itoa(i)
	b.real[alias] = name

	return alias
}

var aliasRe = regexp.MustCompile(`\b(Account|Tag) (\d+)\b`)

// restore puts the real names back into text the model wrote, locally: the
// provider never learns them.
func (b *aliasBook) restore(text string) string {
	return aliasRe.ReplaceAllStringFunc(text, func(m string) string {
		if name, ok := b.real[m]; ok {
			return name
		}

		return m
	})
}

// add appends a row the provider sees under the same label as the person.
func (t *Table) add(label string, cents int64) { t.addAs(label, label, cents) }

// addAs appends a row with a real label for the person and an opaque one for the provider.
func (t *Table) addAs(label, modelLabel string, cents int64) {
	t.Rows = append(t.Rows, Figure{Label: label, Cents: cents})
	t.model = append(t.model, Figure{Label: modelLabel, Cents: cents})
}

// Execute runs resolved queries with the report functions, inside the
// workspace of ctx. It never builds a query from the model's text: the only
// inputs are the validated dates, kind and category.
func (s *Service) Execute(ctx context.Context, queries []Resolved) ([]Table, error) {
	tables := make([]Table, 0, len(queries))
	book := newBook()

	for _, q := range queries {
		t, err := s.run(ctx, q, book)
		if err != nil {
			return nil, err
		}

		if len(t.Rows) > MaxRows {
			t.Rows, t.model, t.Truncated = t.Rows[:MaxRows], t.model[:MaxRows], true
		}

		tables = append(tables, t)
	}

	return tables, nil
}

func (s *Service) run(ctx context.Context, q Resolved, book *aliasBook) (Table, error) {
	t := Table{Report: q.Report, Title: q.Describe(), From: q.From, To: q.To, Rows: []Figure{}, model: []Figure{}, book: book}

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

			t.add(ai.Sanitize(label, labelMax), r.Cents)
		}

		t.add("Total", rep.TotalCents)
	case ReportCashFlow:
		rep, err := s.Rep.CashFlow(ctx, p)
		if err != nil {
			return t, err
		}

		for _, m := range rep.Months {
			t.add(m.Month+" income", m.IncomeCents)
			t.add(m.Month+" expense", m.ExpenseCents)
			t.add(m.Month+" net", m.NetCents)
		}

		t.add("Total income", rep.Totals.IncomeCents)
		t.add("Total expense", rep.Totals.ExpenseCents)
		t.add("Total net", rep.Totals.NetCents)
	case ReportByAccount:
		rep, err := s.Rep.ByAccount(ctx, p)
		if err != nil {
			return t, err
		}

		for _, r := range rep.Rows {
			name, alias := ai.Sanitize(r.Name, labelMax), book.label("Account", r.Name)
			t.addAs(name+" income", alias+" income", r.IncomeCents)
			t.addAs(name+" expense", alias+" expense", r.ExpenseCents)
		}
	case ReportByTag:
		rep, err := s.Rep.ByTag(ctx, p)
		if err != nil {
			return t, err
		}

		for _, r := range rep.Rows {
			name, alias := ai.Sanitize(r.Name, labelMax), book.label("Tag", r.Name)
			t.addAs(name+" income", alias+" income", r.IncomeCents)
			t.addAs(name+" expense", alias+" expense", r.ExpenseCents)
		}
	case ReportNetWorth:
		rep, err := s.Rep.NetWorthSeries(ctx, p)
		if err != nil {
			return t, err
		}

		for _, pt := range rep.Points {
			t.add(pt.Month, pt.BalanceCents)
		}
	case ReportCompare:
		rep, err := s.Rep.Compare(ctx, p)
		if err != nil {
			return t, err
		}

		t.Title += " (previous: " + rep.Previous.From + " to " + rep.Previous.To + ")"

		for _, r := range rep.Rows {
			t.add(r.Key+", this period", r.CurrentCents)
			t.add(r.Key+", previous period", r.PreviousCents)
			t.add(r.Key+", difference", r.DiffCents)
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

		t.add(r.OccurredOn+" "+ai.Sanitize(r.Description, labelMax), cents)
	}

	return t, nil
}
