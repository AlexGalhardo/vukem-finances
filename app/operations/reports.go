package operations

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

// PeriodIn is the query of the report operations: a preset or a custom range.
type PeriodIn struct {
	Period string `json:"period" validate:"max=20" doc:"this_month (the default), last_month, last_3_months, last_12_months, this_year or custom" example:"this_month"`
	From   string `json:"from"   validate:"max=10" doc:"First date of a custom period, YYYY-MM-DD" example:"2026-11-01"`
	To     string `json:"to"     validate:"max=10" doc:"Last date of a custom period, YYYY-MM-DD" example:"2026-11-30"`
}

// CategoryReportIn is the query of reports.categories.
type CategoryReportIn struct {
	Period string `json:"period" validate:"max=20" doc:"this_month (the default), last_month, last_3_months, last_12_months, this_year or custom" example:"this_month"`
	From   string `json:"from"   validate:"max=10" doc:"First date of a custom period, YYYY-MM-DD" example:"2026-11-01"`
	To     string `json:"to"     validate:"max=10" doc:"Last date of a custom period, YYYY-MM-DD" example:"2026-11-30"`
	Kind   string `json:"kind"   validate:"max=20" doc:"expense (the default) or income" example:"expense"`
}

// PeriodOut is the closed range of workspace-local dates a report covers.
type PeriodOut struct {
	Preset string `json:"preset" example:"this_month"`
	From   string `json:"from"   example:"2026-11-01"`
	To     string `json:"to"     example:"2026-11-30"`
}

func periodOut(p reports.Period) PeriodOut {
	return PeriodOut{Preset: p.Preset, From: p.From, To: p.To}
}

// SumOut is income, expense and their difference, in cents.
type SumOut struct {
	IncomeCents  int64 `json:"income_cents"  example:"500000"`
	ExpenseCents int64 `json:"expense_cents" example:"320000"`
	NetCents     int64 `json:"net_cents"     example:"180000"`
}

func sumOut(s reports.Sum) SumOut {
	return SumOut{IncomeCents: s.IncomeCents, ExpenseCents: s.ExpenseCents, NetCents: s.NetCents}
}

// MonthFlowOut is the income and expense of one month.
type MonthFlowOut struct {
	Month string `json:"month" example:"2026-11"`
	SumOut
}

// CashFlowOut is income against expense per month of the period.
type CashFlowOut struct {
	Currency string         `json:"currency" example:"USD"`
	Period   PeriodOut      `json:"period"`
	Months   []MonthFlowOut `json:"months"`
	Totals   SumOut         `json:"totals"`
}

// CategoryChildOut is what a subcategory took in the period.
type CategoryChildOut struct {
	UID           string `json:"uid"           example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name          string `json:"name"          example:"Restaurants"`
	Uncategorized bool   `json:"uncategorized"`
	Cents         int64  `json:"cents"         example:"12000"`
	OwnCents      int64  `json:"own_cents"     example:"12000"`
	ShareBps      int64  `json:"share_bps"     doc:"Its part of the total in basis points (1250 is 12.50 %)" example:"1250"`
}

// CategoryRowOut is what a category took in the period: its own transactions
// plus those of its subcategories, listed in children. The row of the
// transactions with no category has uncategorized set, no uid and no name.
type CategoryRowOut struct {
	UID           string             `json:"uid"           example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name          string             `json:"name"          example:"Food"`
	Uncategorized bool               `json:"uncategorized"`
	Cents         int64              `json:"cents"         example:"32000"`
	OwnCents      int64              `json:"own_cents"     example:"20000"`
	ShareBps      int64              `json:"share_bps"     doc:"Its part of the total in basis points (1250 is 12.50 %)" example:"3000"`
	Children      []CategoryChildOut `json:"children"`
}

// CategoryReportOut is the spending (or income) of the period by category.
type CategoryReportOut struct {
	Currency   string           `json:"currency" example:"USD"`
	Period     PeriodOut        `json:"period"`
	Kind       string           `json:"kind"     example:"expense"`
	TotalCents int64            `json:"total_cents" example:"320000"`
	Rows       []CategoryRowOut `json:"rows"`
}

// AccountRowOut is the income and expense of one account in the period.
type AccountRowOut struct {
	UID  string `json:"uid"  example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name string `json:"name" example:"Checking"`
	Kind string `json:"kind" example:"checking"`
	SumOut
}

// AccountReportOut is the period by account; only accounts with movement are listed.
type AccountReportOut struct {
	Currency string          `json:"currency" example:"USD"`
	Period   PeriodOut       `json:"period"`
	Rows     []AccountRowOut `json:"rows"`
}

// TagRowOut is the income and expense of the transactions that carry a tag.
type TagRowOut struct {
	UID  string `json:"uid"  example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name string `json:"name" example:"work"`
	SumOut
}

// TagReportOut is the period by tag. A transaction with several tags counts
// once under each of them, so the rows can add up to more than the totals.
type TagReportOut struct {
	Currency string      `json:"currency" example:"USD"`
	Period   PeriodOut   `json:"period"`
	Rows     []TagRowOut `json:"rows"`
}

// NetWorthPointOut is the net worth at the end of a month.
type NetWorthPointOut struct {
	Month        string `json:"month"         example:"2026-11"`
	BalanceCents int64  `json:"balance_cents" example:"1234500"`
}

// NetWorthReportOut is the trend of the net worth over the months of a period.
type NetWorthReportOut struct {
	Currency string             `json:"currency" example:"USD"`
	Period   PeriodOut          `json:"period"`
	Points   []NetWorthPointOut `json:"points"`
}

// CompareRowOut sets one total next to the same total of the previous period.
type CompareRowOut struct {
	Key           string `json:"key"            doc:"income, expense or net" example:"expense"`
	CurrentCents  int64  `json:"current_cents"  example:"320000"`
	PreviousCents int64  `json:"previous_cents" example:"300000"`
	DiffCents     int64  `json:"diff_cents"     example:"20000"`
	ChangeBps     int64  `json:"change_bps"     doc:"The change as a part of the previous value in basis points; 0 when not_applicable" example:"667"`
	NotApplicable bool   `json:"not_applicable" doc:"The previous value is zero, so there is no percentage"`
}

// CompareReportOut compares a period with the one right before it.
type CompareReportOut struct {
	Currency string          `json:"currency" example:"USD"`
	Period   PeriodOut       `json:"period"`
	Previous PeriodOut       `json:"previous"`
	Rows     []CompareRowOut `json:"rows"`
}

type reportOps struct {
	CashFlow   *Op[PeriodIn, CashFlowOut]
	Categories *Op[CategoryReportIn, CategoryReportOut]
	Accounts   *Op[PeriodIn, AccountReportOut]
	Tags       *Op[PeriodIn, TagReportOut]
	NetWorth   *Op[PeriodIn, NetWorthReportOut]
	Compare    *Op[PeriodIn, CompareReportOut]
}

func (c *Catalog) defineReports() {
	c.Reports = reportOps{
		CashFlow: Define(c.Registry, Operation[PeriodIn, CashFlowOut]{
			Name: "reports.cashflow", Summary: "Income against expense per month of a period", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in PeriodIn) (CashFlowOut, error) { return c.reportCashFlow(ctx, in) },
		}),
		Categories: Define(c.Registry, Operation[CategoryReportIn, CategoryReportOut]{
			Name: "reports.categories", Summary: "Spending or income of a period by category", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in CategoryReportIn) (CategoryReportOut, error) {
				return c.reportCategories(ctx, in)
			},
		}),
		Accounts: Define(c.Registry, Operation[PeriodIn, AccountReportOut]{
			Name: "reports.accounts", Summary: "Income and expense of a period by account", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in PeriodIn) (AccountReportOut, error) {
				return c.reportAccounts(ctx, in)
			},
		}),
		Tags: Define(c.Registry, Operation[PeriodIn, TagReportOut]{
			Name: "reports.tags", Summary: "Income and expense of a period by tag", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in PeriodIn) (TagReportOut, error) { return c.reportTags(ctx, in) },
		}),
		NetWorth: Define(c.Registry, Operation[PeriodIn, NetWorthReportOut]{
			Name: "reports.networth", Summary: "The net worth at the end of each month of a period", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in PeriodIn) (NetWorthReportOut, error) {
				return c.reportNetWorth(ctx, in)
			},
		}),
		Compare: Define(c.Registry, Operation[PeriodIn, CompareReportOut]{
			Name: "reports.compare", Summary: "A period against the one right before it", Scope: ScopeReportsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in PeriodIn) (CompareReportOut, error) {
				return c.reportCompare(ctx, in)
			},
		}),
	}
}

func (c *Catalog) period(ctx context.Context, in PeriodIn) (reports.Period, error) {
	return c.Rep.Period(ctx, in.Period, in.From, in.To)
}

func (c *Catalog) reportCashFlow(ctx context.Context, in PeriodIn) (CashFlowOut, error) {
	p, err := c.period(ctx, in)
	if err != nil {
		return CashFlowOut{}, err
	}

	r, err := c.Rep.CashFlow(ctx, p)
	if err != nil {
		return CashFlowOut{}, err
	}

	out := CashFlowOut{Currency: r.Currency, Period: periodOut(r.Period), Months: make([]MonthFlowOut, 0, len(r.Months)), Totals: sumOut(r.Totals)}
	for _, m := range r.Months {
		out.Months = append(out.Months, MonthFlowOut{Month: m.Month, SumOut: sumOut(m.Sum)})
	}

	return out, nil
}

func (c *Catalog) reportCategories(ctx context.Context, in CategoryReportIn) (CategoryReportOut, error) {
	p, err := c.period(ctx, PeriodIn{Period: in.Period, From: in.From, To: in.To})
	if err != nil {
		return CategoryReportOut{}, err
	}

	kind := in.Kind
	if kind != models.KindIncome {
		kind = models.KindExpense
	}

	r, err := c.Rep.ByCategory(ctx, p, kind)
	if err != nil {
		return CategoryReportOut{}, err
	}

	out := CategoryReportOut{
		Currency: r.Currency, Period: periodOut(r.Period), Kind: r.Kind, TotalCents: r.TotalCents,
		Rows: make([]CategoryRowOut, 0, len(r.Rows)),
	}

	for _, row := range r.Rows {
		o := CategoryRowOut{
			UID: row.UID, Name: row.Name, Uncategorized: row.Uncategorized, Cents: row.Cents, OwnCents: row.OwnCents,
			ShareBps: row.ShareBps, Children: make([]CategoryChildOut, 0, len(row.Children)),
		}

		for _, ch := range row.Children {
			o.Children = append(o.Children, CategoryChildOut{
				UID: ch.UID, Name: ch.Name, Uncategorized: ch.Uncategorized, Cents: ch.Cents, OwnCents: ch.OwnCents, ShareBps: ch.ShareBps,
			})
		}

		out.Rows = append(out.Rows, o)
	}

	return out, nil
}

func (c *Catalog) reportAccounts(ctx context.Context, in PeriodIn) (AccountReportOut, error) {
	p, err := c.period(ctx, in)
	if err != nil {
		return AccountReportOut{}, err
	}

	r, err := c.Rep.ByAccount(ctx, p)
	if err != nil {
		return AccountReportOut{}, err
	}

	out := AccountReportOut{Currency: r.Currency, Period: periodOut(r.Period), Rows: make([]AccountRowOut, 0, len(r.Rows))}
	for _, row := range r.Rows {
		out.Rows = append(out.Rows, AccountRowOut{UID: row.UID, Name: row.Name, Kind: row.Kind, SumOut: sumOut(row.Sum)})
	}

	return out, nil
}

func (c *Catalog) reportTags(ctx context.Context, in PeriodIn) (TagReportOut, error) {
	p, err := c.period(ctx, in)
	if err != nil {
		return TagReportOut{}, err
	}

	r, err := c.Rep.ByTag(ctx, p)
	if err != nil {
		return TagReportOut{}, err
	}

	out := TagReportOut{Currency: r.Currency, Period: periodOut(r.Period), Rows: make([]TagRowOut, 0, len(r.Rows))}
	for _, row := range r.Rows {
		out.Rows = append(out.Rows, TagRowOut{UID: row.UID, Name: row.Name, SumOut: sumOut(row.Sum)})
	}

	return out, nil
}

func (c *Catalog) reportNetWorth(ctx context.Context, in PeriodIn) (NetWorthReportOut, error) {
	p, err := c.period(ctx, in)
	if err != nil {
		return NetWorthReportOut{}, err
	}

	r, err := c.Rep.NetWorthSeries(ctx, p)
	if err != nil {
		return NetWorthReportOut{}, err
	}

	out := NetWorthReportOut{Currency: r.Currency, Period: periodOut(r.Period), Points: make([]NetWorthPointOut, 0, len(r.Points))}
	for _, pt := range r.Points {
		out.Points = append(out.Points, NetWorthPointOut{Month: pt.Month, BalanceCents: pt.BalanceCents})
	}

	return out, nil
}

func (c *Catalog) reportCompare(ctx context.Context, in PeriodIn) (CompareReportOut, error) {
	p, err := c.period(ctx, in)
	if err != nil {
		return CompareReportOut{}, err
	}

	r, err := c.Rep.Compare(ctx, p)
	if err != nil {
		return CompareReportOut{}, err
	}

	out := CompareReportOut{
		Currency: r.Currency, Period: periodOut(r.Period), Previous: periodOut(r.Previous), Rows: make([]CompareRowOut, 0, len(r.Rows)),
	}

	for _, row := range r.Rows {
		out.Rows = append(out.Rows, CompareRowOut{
			Key: row.Key, CurrentCents: row.CurrentCents, PreviousCents: row.PreviousCents, DiffCents: row.DiffCents,
			ChangeBps: row.ChangeBps, NotApplicable: row.NotApplicable,
		})
	}

	return out, nil
}
