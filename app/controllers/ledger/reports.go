package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
)

// Reports renders every report of a period: ?period= is a preset (this_month,
// last_month, last_3_months, last_12_months, this_year) or custom with ?from=
// and ?to= (YYYY-MM-DD, the workspace's calendar). Any member may read them.
func (h *Handlers) Reports(c *web.Ctx) error {
	ctx := c.Context()

	p, err := h.Rep.Period(ctx, c.Query("period"), c.Query("from"), c.Query("to"))
	if err != nil {
		return back(c, "/reports", err)
	}

	props := pages.ReportsProps{Period: p}

	if props.CashFlow, err = h.Rep.CashFlow(ctx, p); err != nil {
		return err
	}

	props.Currency = props.CashFlow.Currency

	if props.ExpenseCategories, err = h.Rep.ByCategory(ctx, p, models.KindExpense); err != nil {
		return err
	}

	if props.IncomeCategories, err = h.Rep.ByCategory(ctx, p, models.KindIncome); err != nil {
		return err
	}

	if props.Accounts, err = h.Rep.ByAccount(ctx, p); err != nil {
		return err
	}

	if props.Tags, err = h.Rep.ByTag(ctx, p); err != nil {
		return err
	}

	if props.NetWorth, err = h.Rep.NetWorthSeries(ctx, p); err != nil {
		return err
	}

	if props.Compare, err = h.Rep.Compare(ctx, p); err != nil {
		return err
	}

	return pages.Reports.Render(c, props)
}
