package api

import (
	"regexp"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

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

// month reads and checks ?month=, today's month by default.
func (a *API) month(c *web.Ctx, in string) (string, error) {
	if in == "" {
		return a.Fin.Month(c.Context()), nil
	}

	if !monthShape.MatchString(in) || !finance.ValidMonth(in) {
		return "", errMonth
	}

	return in, nil
}

func (a *API) budgetViews(c *web.Ctx, month string) ([]BudgetOut, error) {
	ctx := c.Context()

	status, err := a.Fin.Budgets(ctx, month)
	if err != nil {
		return nil, err
	}

	rows, err := workspaces.Scoped[models.Budget](ctx, a.WS.DB).Get(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Budget, len(rows))
	for _, b := range rows {
		byID[b.ID] = b
	}

	l, err := a.lookups(ctx)
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

func (a *API) budgetsList(c *web.Ctx, in MonthIn) (List[BudgetOut], error) {
	month, err := a.month(c, in.Month)
	if err != nil {
		return List[BudgetOut]{}, err
	}

	data, err := a.budgetViews(c, month)

	return List[BudgetOut]{Data: data}, err
}

func (a *API) budgetsSet(c *web.Ctx, in BudgetSetIn) (BudgetOut, error) {
	ctx := c.Context()

	cat, err := a.categoryByUID(ctx, in.CategoryUID)
	if err != nil {
		return BudgetOut{}, err
	}

	if err := a.Fin.SetBudget(ctx, cat.ID, centsText(in.LimitCents)); err != nil {
		return BudgetOut{}, err
	}

	set, err := workspaces.Scoped[models.Budget](ctx, a.WS.DB).Where("category_id", cat.ID).First(ctx)
	if err != nil {
		return BudgetOut{}, err
	}

	views, err := a.budgetViews(c, a.Fin.Month(ctx))
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

func (a *API) budgetsDelete(c *web.Ctx, in UIDIn) (Empty, error) {
	ctx := c.Context()

	b, err := a.budgetByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	if want, err := ifMatch(c); err != nil || (want >= 0 && want != b.Version) {
		if err != nil {
			return Empty{}, err
		}

		views, verr := a.budgetViews(c, a.Fin.Month(ctx))
		if verr != nil {
			return Empty{}, verr
		}

		for _, v := range views {
			if v.UID == b.UID {
				return Empty{}, checkVersion(c, b.Version, v)
			}
		}
	}

	return Empty{}, a.Fin.DeleteBudget(ctx, b.ID)
}

func (a *API) summaryShow(c *web.Ctx, in MonthIn) (SummaryOut, error) {
	ctx := c.Context()

	month, err := a.month(c, in.Month)
	if err != nil {
		return SummaryOut{}, err
	}

	d, err := a.Fin.Dashboard(ctx, month)
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

	accounts, err := a.Fin.Accounts(ctx)
	if err != nil {
		return SummaryOut{}, err
	}

	balances, err := a.Fin.AccountBalances(ctx, "")
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
