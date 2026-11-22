package tests

import (
	"testing"
)

func TestReportsPage(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	record(t, app, ana, acc, 0, "income", "100.00", "2026-01-05", "Pay")
	record(t, app, ana, acc, 0, "expense", "40.00", "2026-01-06", "Food")

	page := ana.Client.Inertia("").Get("/reports?period=custom&from=2026-01-01&to=2026-01-31").AssertOK().AssertInertia("Reports")
	page.PropEquals("period.from", "2026-01-01")
	page.PropEquals("cashFlow.totals.incomeCents", float64(10000))
	page.PropEquals("cashFlow.totals.expenseCents", float64(4000))
	page.PropEquals("expenseCategories.totalCents", float64(4000))
	page.PropEquals("compare.rows.0.notApplicable", true)

	ana.Client.Get("/reports?period=custom&from=2026-02-01&to=2026-01-01").AssertRedirect("/reports")

	vera := viewerOf(t, app, ana)
	vera.Client.Inertia("").Get("/reports").AssertOK().AssertInertia("Reports")
}
