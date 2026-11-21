package finance

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// MonthPoint is the income and expense of one month ("2026-11").
type MonthPoint struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"incomeCents"`
	ExpenseCents int64  `json:"expenseCents"`
}

// CategorySpend is what a category cost in the month.
type CategorySpend struct {
	Category string `json:"category"`
	Cents    int64  `json:"cents"`
}

// Dashboard is what the home page charts.
type Dashboard struct {
	Currency     string          `json:"currency"`
	Month        string          `json:"month"`
	BalanceCents int64           `json:"balanceCents"`
	Months       []MonthPoint    `json:"months"`
	Categories   []CategorySpend `json:"categories"`
	Budgets      []BudgetStatus  `json:"budgets"`
}

// monthsBack lists the n months ending at month, oldest first.
func monthsBack(month string, n int) []string {
	t, _ := time.Parse("2006-01", month)

	out := make([]string, n)
	for i := range n {
		out[n-1-i] = timeutil.AddMonths(t, -i).Format("2006-01")
	}

	return out
}

// Dashboard computes the balance, the last six months and the spending by
// category of month.
func (s *Service) Dashboard(ctx context.Context, month string) (Dashboard, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return Dashboard{}, err
	}

	d := Dashboard{Currency: ws.Currency, Month: month, Categories: []CategorySpend{}}

	opening, err := workspaces.Scoped[models.Account](ctx, s.DB).Sum[int64](ctx, "opening_cents")
	if err != nil {
		return d, err
	}

	txs := workspaces.Scoped[models.Transaction](ctx, s.DB)

	income, err := txs.Where("kind", models.KindIncome).Sum[int64](ctx, "amount_cents")
	if err != nil {
		return d, err
	}

	expense, err := txs.Where("kind", models.KindExpense).Sum[int64](ctx, "amount_cents")
	if err != nil {
		return d, err
	}

	d.BalanceCents = opening + income - expense

	months := monthsBack(month, 6)
	points := map[string]*MonthPoint{}

	d.Months = make([]MonthPoint, len(months))
	for i, m := range months {
		d.Months[i] = MonthPoint{Month: m}
		points[m] = &d.Months[i]
	}

	cats, err := s.Categories(ctx)
	if err != nil {
		return d, err
	}

	names := map[int64]string{}
	for _, c := range cats {
		names[c.ID] = c.Name
	}

	byCategory := map[string]int64{}
	from, _ := monthBounds(months[0])
	_, to := monthBounds(month)

	for t, err := range workspaces.Scoped[models.Transaction](ctx, s.DB).WhereBetween("occurred_on", from, to).Each(ctx) {
		if err != nil {
			return d, err
		}

		p := points[MonthOf(t.OccurredOn)]
		if p == nil {
			continue
		}

		if t.Kind == models.KindIncome {
			p.IncomeCents += t.AmountCents
			continue
		}

		p.ExpenseCents += t.AmountCents

		if MonthOf(t.OccurredOn) == month {
			name := "Uncategorized"
			if t.CategoryID.Valid {
				name = names[t.CategoryID.V]
			}

			byCategory[name] += t.AmountCents
		}
	}

	for name, cents := range byCategory {
		d.Categories = append(d.Categories, CategorySpend{Category: name, Cents: cents})
	}

	slices.SortFunc(d.Categories, func(a, b CategorySpend) int { return int(b.Cents - a.Cents) })

	d.Budgets, err = s.Budgets(ctx, month)

	return d, err
}

// SendMonthlySummary emails the owners of the workspace of ctx how month
// went; a month without transactions sends nothing.
func (s *Service) SendMonthlySummary(ctx context.Context, month string) error {
	d, err := s.Dashboard(ctx, month)
	if err != nil {
		return err
	}

	last := d.Months[len(d.Months)-1]
	if last.IncomeCents == 0 && last.ExpenseCents == 0 {
		return nil
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	owners, err := s.WS.OwnerEmails(ctx, ws.ID)
	if err != nil {
		return err
	}

	top := make([]appmail.Line, 0, 3)
	for _, c := range d.Categories[:min(3, len(d.Categories))] {
		top = append(top, appmail.Line{Category: c.Category, Amount: FormatAmount(c.Cents, ws.Currency)})
	}

	for _, to := range owners {
		err := s.WS.Notify(ctx, appmail.MonthlySummary{
			To: to, Workspace: ws.Name, Month: month,
			Income:  FormatAmount(last.IncomeCents, ws.Currency),
			Expense: FormatAmount(last.ExpenseCents, ws.Currency),
			Net:     FormatAmount(last.IncomeCents-last.ExpenseCents, ws.Currency),
			URL:     s.URL + "/dashboard", Top: top,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// summaryHour is the local hour, on the first day of a month, from which the
// summary of the month that just ended goes out. The scheduler runs hourly,
// so a run missed at 08:00 is made up later that day; the sent marker keeps
// every workspace to one summary per month.
const summaryHour = 8

// SendDueMonthlySummaries sends the summary of the month that just ended to
// every workspace for which it is the first day of a month, from 08:00 on, in
// its own timezone, and whose summary was not sent yet. It is the one place
// that reads across tenants on purpose: each workspace is handled under its
// own tenant context. Running it twice sends nothing the second time.
func (s *Service) SendDueMonthlySummaries(ctx context.Context, now time.Time) error {
	// shortcut: loads every workspace at once (the rows stay small), and an
	// open cursor would hold SQLite's single connection while each summary
	// queries; page with Chunk when there are many thousands.
	all, err := orm.QueryOf[models.Workspace](s.DB).Get(ctx)
	if err != nil {
		return err
	}

	// One workspace that cannot be mailed must not starve the ones after it.
	var failed error

	for _, ws := range all {
		loc, err := workspaces.LoadZone(ws.Timezone)
		if err != nil {
			loc = time.UTC
		}

		local := now.In(loc)
		if local.Day() != 1 || local.Hour() < summaryHour {
			continue
		}

		if err := s.sendOnce(workspaces.WithTenant(ctx, ws.ID), PreviousMonth(local)); err != nil {
			failed = errors.Join(failed, fmt.Errorf("monthly summary of workspace %d: %w", ws.ID, err))
		}
	}

	return failed
}

// sendOnce claims the marker of month for the workspace of ctx and sends its
// summary; a workspace that already holds the marker is skipped. A failed
// send releases the marker so the next run tries again.
func (s *Service) sendOnce(ctx context.Context, month string) error {
	err := orm.Create(ctx, s.DB, &models.MonthlySummarySent{Month: month})
	if err != nil && (s.DB.UniqueViolation(err) || errors.Is(err, orm.ErrConflict)) {
		return nil
	}

	if err != nil {
		return err
	}

	if err := s.SendMonthlySummary(ctx, month); err != nil {
		_, delErr := workspaces.Scoped[models.MonthlySummarySent](ctx, s.DB).Where("month", month).Delete(ctx)

		return errors.Join(err, delErr)
	}

	return nil
}

// PreviousMonth is the "YYYY-MM" before the month of t.
func PreviousMonth(t time.Time) string {
	return timeutil.AddMonths(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), -1).Format("2006-01")
}
