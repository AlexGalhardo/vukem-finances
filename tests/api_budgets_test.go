package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestAPIBudgets(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	c().Get("/api/v1/budgets?month=2026-11").AssertOK().AssertJSONCount("data", 0)

	c().PostJSON("/api/v1/transactions", f.expense(12000, "2026-11-05")).AssertCreated()

	// Setting is an upsert on the category.
	set := c().PostJSON("/api/v1/budgets", map[string]any{"category_uid": f.food.UID, "limit_cents": 50000}).AssertOK()
	set.AssertJSONPath("limit_cents", float64(50000)).AssertJSONPath("category_name", "Groceries")
	uid := uidOf(t, set, "uid")

	c().PostJSON("/api/v1/budgets", map[string]any{"category_uid": f.food.UID, "limit_cents": 40000}).AssertOK().AssertJSONPath("uid", uid)

	list := c().Get("/api/v1/budgets?month=2026-11").AssertOK()
	list.AssertJSONCount("data", 1).AssertJSONPath("data.0.spent_cents", float64(12000)).AssertJSONPath("data.0.limit_cents", float64(40000)).
		AssertJSONPath("data.0.percent", float64(30)).AssertJSONPath("data.0.month", "2026-11")

	assertProblem(t, c().Get("/api/v1/budgets?month=2026-13"), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/budgets", map[string]any{"category_uid": f.food.UID, "limit_cents": 0}), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/budgets", map[string]any{"category_uid": "01ARZ3NDEKTSV4RRFFQ69G5FAV", "limit_cents": 100}), 404, "not_found")

	assertProblem(t, c().WithHeader("If-Match", "99").Delete("/api/v1/budgets/"+uid), 409, "version_conflict")
	c().Delete("/api/v1/budgets/" + uid).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/budgets?month=2026-11").AssertOK().AssertJSONCount("data", 0)
}

func TestAPIBudgetsNeedPro(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := fullToken(t, app, ana)

	cat := firstOf[models.Category](t, app, ana, "Groceries").UID
	assertProblem(t, apiClient(t, app, token).PostJSON("/api/v1/budgets", map[string]any{"category_uid": cat, "limit_cents": 100}), 402, "plan_limit")
}

func TestAPISummary(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	c().PostJSON("/api/v1/transactions", f.expense(1250, "2026-11-21")).AssertCreated()
	income := f.expense(5000, "2026-11-01")
	income["kind"], income["category_uid"], income["tags"] = "income", nil, nil
	c().PostJSON("/api/v1/transactions", income).AssertCreated()
	c().PostJSON("/api/v1/transactions", f.expense(999, "2026-10-01")).AssertCreated()

	res := c().Get("/api/v1/summary?month=2026-11").AssertOK()
	res.AssertJSONPath("month", "2026-11").AssertJSONPath("currency", "USD").AssertJSONPath("income_cents", float64(5000)).
		AssertJSONPath("expense_cents", float64(1250)).AssertJSONPath("net_cents", float64(3750)).
		AssertJSONPath("net_worth_cents", float64(5000-1250-999)).AssertJSONCount("accounts", 2)

	c().Get("/api/v1/summary").AssertOK()
	assertProblem(t, c().Get("/api/v1/summary?month=nov"), 422, "validation_failed")
}
