package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errBudgetCategory = errs.New(http.StatusUnprocessableEntity, "Choose one of your categories.")

// Budgets lists the budgets of this month against what was spent.
func (h *Handlers) Budgets(c *web.Ctx) error {
	month := h.Fin.Month(c.Context())

	// ?month= looks at another month; anything that is not a real month is
	// ignored rather than trusted.
	if m := c.Query("month"); finance.ValidMonth(m) {
		month = m
	}

	budgets, err := h.Fin.Budgets(c.Context(), month)
	if err != nil {
		return err
	}

	cats, err := h.Fin.Categories(c.Context())
	if err != nil {
		return err
	}

	lim, err := h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	return pages.Budgets.Render(c, pages.BudgetsProps{
		Budgets: budgets, Categories: cats, Month: month, Enabled: lim.Budgets,
	})
}

// StoreBudget sets the monthly limit of a category.
func (h *Handlers) StoreBudget(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.BudgetSchema)
	if err != nil {
		return err
	}

	// An empty category is the overall budget; a bad id must never become one.
	var category int64

	if in.CategoryID != "" {
		if category = id(in.CategoryID); category <= 0 {
			return back(c, "/budgets", errBudgetCategory)
		}
	}

	rollover := in.Rollover

	if err := h.Fin.SaveBudget(c.Context(), finance.BudgetInput{CategoryID: category, Limit: in.Limit, Rollover: &rollover}); err != nil {
		return back(c, "/budgets", err)
	}

	return done(c, "/budgets", "Budget saved.")
}

// DestroyBudget removes a budget.
func (h *Handlers) DestroyBudget(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	bid, err := c.ParamInt("budget")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteBudget(c.Context(), bid); err != nil {
		return err
	}

	return done(c, "/budgets", "Budget removed.")
}
