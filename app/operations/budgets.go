package operations

import (
	"context"
	"regexp"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	monthShape = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
	errMonth   = errs.New(422, "The month must look like 2026-11.")
)

// MonthIn is the query of the month operations.
type MonthIn struct {
	Month string `json:"month" validate:"max=7" doc:"YYYY-MM; this month in the workspace timezone when left out" example:"2026-11"`
}

// BudgetSetIn is the body of budgets.set.
type BudgetSetIn struct {
	CategoryUID string `json:"category_uid" validate:"required,max=26" doc:"An expense category"`
	LimitCents  int64  `json:"limit_cents"  doc:"The monthly limit, a positive whole number of cents" example:"50000"`
}

type budgetOps struct {
	List    *Op[MonthIn, List[BudgetOut]]
	Set     *Op[BudgetSetIn, BudgetOut]
	Delete  *Op[UIDIn, Empty]
	Summary *Op[MonthIn, SummaryOut]
}

func (c *Catalog) defineBudgets() {
	c.Budgets = budgetOps{
		List: Define(c.Registry, Operation[MonthIn, List[BudgetOut]]{
			Name: "budgets.list", Summary: "List the budgets with what the month spent", Scope: ScopeBudgetsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in MonthIn) (List[BudgetOut], error) { return c.budgetsList(ctx, in) },
		}),
		Set: Define(c.Registry, Operation[BudgetSetIn, BudgetOut]{
			Name: "budgets.set", Summary: "Set the monthly limit of an expense category", Scope: ScopeBudgetsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in BudgetSetIn) (BudgetOut, error) { return c.budgetsSet(ctx, in) },
		}),
		Delete: Define(c.Registry, Operation[UIDIn, Empty]{
			Name: "budgets.delete", Summary: "Remove a budget", Scope: ScopeBudgetsWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (Empty, error) { return c.budgetsDelete(ctx, in) },
		}),
		Summary: Define(c.Registry, Operation[MonthIn, SummaryOut]{
			Name: "summary.show", Summary: "Income, expense, net, account balances and net worth of a month", Scope: ScopeSummaryRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in MonthIn) (SummaryOut, error) { return c.summaryShow(ctx, in) },
		}),
	}
}

// month reads and checks a month, today's month by default.
func (c *Catalog) month(ctx context.Context, in string) (string, error) {
	if in == "" {
		return c.Fin.Month(ctx), nil
	}

	if !monthShape.MatchString(in) || !finance.ValidMonth(in) {
		return "", errMonth
	}

	return in, nil
}

func (c *Catalog) budgetViews(ctx context.Context, month string) ([]BudgetOut, error) {
	status, err := c.Fin.Budgets(ctx, month)
	if err != nil {
		return nil, err
	}

	rows, err := workspaces.Scoped[models.Budget](ctx, c.WS.DB).Get(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Budget, len(rows))
	for _, b := range rows {
		byID[b.ID] = b
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]BudgetOut, 0, len(status))
	for _, s := range status {
		if s.Overall {
			continue // the overall budget has no category; the API lists category budgets only
		}

		b := byID[s.ID]
		out = append(out, BudgetOut{
			UID: b.UID, Version: b.Version, CategoryUID: l.categories[s.CategoryID], CategoryName: s.Category, Month: month,
			LimitCents: s.LimitCents, SpentCents: s.SpentCents, Percent: s.Percent, Currency: l.currency,
		})
	}

	return out, nil
}

func (c *Catalog) budgetsList(ctx context.Context, in MonthIn) (List[BudgetOut], error) {
	month, err := c.month(ctx, in.Month)
	if err != nil {
		return List[BudgetOut]{}, err
	}

	data, err := c.budgetViews(ctx, month)

	return List[BudgetOut]{Data: data}, err
}

func (c *Catalog) budgetsSet(ctx context.Context, in BudgetSetIn) (BudgetOut, error) {
	cat, err := c.categoryByUID(ctx, in.CategoryUID)
	if err != nil {
		return BudgetOut{}, err
	}

	if err := c.Fin.SetBudget(ctx, cat.ID, centsText(in.LimitCents)); err != nil {
		return BudgetOut{}, err
	}

	set, err := workspaces.Scoped[models.Budget](ctx, c.WS.DB).Where("category_id", cat.ID).First(ctx)
	if err != nil {
		return BudgetOut{}, err
	}

	views, err := c.budgetViews(ctx, c.Fin.Month(ctx))
	if err != nil {
		return BudgetOut{}, err
	}

	for _, v := range views {
		if v.UID == set.UID {
			return v, nil
		}
	}

	return BudgetOut{}, errs.ErrNotFound
}

func (c *Catalog) budgetsDelete(ctx context.Context, in UIDIn) (Empty, error) {
	b, err := c.budgetByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	if want, err := precondition(ctx); err != nil || (want >= 0 && want != b.Version) {
		if err != nil {
			return Empty{}, err
		}

		views, verr := c.budgetViews(ctx, c.Fin.Month(ctx))
		if verr != nil {
			return Empty{}, verr
		}

		for _, v := range views {
			if v.UID == b.UID {
				return Empty{}, checkVersion(ctx, b.Version, v)
			}
		}
	}

	return Empty{}, c.Fin.DeleteBudget(ctx, b.ID)
}

func (c *Catalog) summaryShow(ctx context.Context, in MonthIn) (SummaryOut, error) {
	month, err := c.month(ctx, in.Month)
	if err != nil {
		return SummaryOut{}, err
	}

	d, err := c.Fin.Dashboard(ctx, month)
	if err != nil {
		return SummaryOut{}, err
	}

	out := SummaryOut{Month: month, Currency: d.Currency, NetWorthCents: d.BalanceCents, Accounts: []SummaryAccountOut{}}

	for _, p := range d.Months {
		if p.Month == month {
			out.IncomeCents, out.ExpenseCents = p.IncomeCents, p.ExpenseCents
		}
	}

	out.NetCents = out.IncomeCents - out.ExpenseCents

	accounts, err := c.Fin.Accounts(ctx)
	if err != nil {
		return SummaryOut{}, err
	}

	balances, err := c.Fin.AccountBalances(ctx, "")
	if err != nil {
		return SummaryOut{}, err
	}

	byID := make(map[int64]int64, len(balances))
	for _, b := range balances {
		byID[b.AccountID] = b.BalanceCents
	}

	for _, x := range accounts {
		out.Accounts = append(out.Accounts, SummaryAccountOut{
			UID: x.UID, Name: x.Name, BalanceCents: byID[x.ID], Archived: x.ArchivedAt.Valid,
		})
	}

	return out, nil
}
