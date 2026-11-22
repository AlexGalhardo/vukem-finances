package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestAPICards(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	res := c().PostJSON("/api/v1/cards", map[string]any{
		"name": "Visa", "limit_cents": 150000, "closing_day": 10, "due_day": 20, "brand": "visa", "last4": "4242",
		"payment_account_uid": f.checking.UID,
	}).AssertCreated()
	res.AssertJSONPath("name", "Visa").AssertJSONPath("limit_cents", float64(150000)).AssertJSONPath("available_cents", float64(150000)).
		AssertJSONPath("payment_account_uid", f.checking.UID).AssertJSONPath("currency", "USD").AssertJSONPath("archived", false)
	uid := uidOf(t, res, "uid")
	accountUID := uidOf(t, res, "account_uid")

	// A card is also an account of its own, and only its last four digits are kept.
	c().Get("/api/v1/accounts/"+accountUID).AssertOK().AssertJSONPath("kind", "credit_card")
	c().Get("/api/v1/cards").AssertOK().AssertJSONCount("data", 1)
	c().Get("/api/v1/cards/"+uid).AssertOK().AssertJSONPath("last4", "4242")

	assertProblem(t, c().PostJSON("/api/v1/cards", map[string]any{"name": "Bad", "limit_cents": 0, "closing_day": 10, "due_day": 20}), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/cards", map[string]any{"name": "Bad", "limit_cents": 100, "closing_day": 10, "due_day": 20, "last4": "12345"}), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/cards", map[string]any{"name": "Bad", "limit_cents": 100, "closing_day": 32, "due_day": 20}), 422, "validation_failed")
	assertProblem(t, c().Get("/api/v1/cards/01ARZ3NDEKTSV4RRFFQ69G5FAV"), 404, "not_found")

	// A field left out stays; payment_account_uid "" removes the default.
	c().PatchJSON("/api/v1/cards/"+uid, map[string]any{"name": "Gold", "payment_account_uid": ""}).AssertOK().
		AssertJSONPath("name", "Gold").AssertJSONPath("limit_cents", float64(150000)).AssertJSONPath("payment_account_uid", nil)

	// A purchase on the card opens a statement; the statement shows it and can be paid.
	c().PostJSON("/api/v1/transactions", map[string]any{
		"account_uid": accountUID, "kind": "expense", "amount_cents": 12000, "date": "2026-11-15", "description": "Shoes",
	}).AssertCreated()

	list := c().Get("/api/v1/cards/" + uid + "/statements").AssertOK()
	ref := uidOf(t, list, "data.0.ref")
	list.AssertJSONPath("data.0.total_cents", float64(12000))

	detail := c().Get("/api/v1/cards/" + uid + "/statements/" + ref).AssertOK()
	detail.AssertJSONPath("statement.remaining_cents", float64(12000)).AssertJSONCount("rows", 1)
	assertProblem(t, c().Get("/api/v1/cards/"+uid+"/statements/1999-01-01"), 404, "not_found")

	pay := c().PostJSON("/api/v1/cards/"+uid+"/statements/"+ref+"/pay", map[string]any{"from_account_uid": f.checking.UID, "date": "2026-11-20"}).AssertCreated()
	pay.AssertJSONPath("from.amount_cents", float64(12000)).AssertJSONPath("to.account_uid", accountUID)
	c().Get("/api/v1/cards/"+uid+"/statements/"+ref).AssertOK().AssertJSONPath("statement.remaining_cents", float64(0)).
		AssertJSONPath("statement.paid_cents", float64(12000))
	assertProblem(t, c().PostJSON("/api/v1/cards/"+uid+"/statements/"+ref+"/pay",
		map[string]any{"from_account_uid": accountUID}), 422, "validation_failed")

	// Archive and bring it back.
	c().PostJSON("/api/v1/cards/"+uid+"/archive", nil).AssertOK().AssertJSONPath("archived", true)
	c().PostJSON("/api/v1/cards/"+uid+"/unarchive", nil).AssertOK().AssertJSONPath("archived", false)
}

func TestAPIRecurrences(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	body := map[string]any{
		"account_uid": f.checking.UID, "category_uid": f.food.UID, "kind": "expense", "amount_cents": 4990,
		"description": "Streaming", "frequency": "monthly", "interval": 1, "start_date": "2099-01-15",
	}

	res := c().PostJSON("/api/v1/recurrences", body).AssertCreated()
	res.AssertJSONPath("amount_cents", float64(4990)).AssertJSONPath("frequency", "monthly").AssertJSONPath("active", true).
		AssertJSONPath("account_uid", f.checking.UID).AssertJSONPath("category_uid", f.food.UID)
	uid := uidOf(t, res, "uid")

	c().Get("/api/v1/recurrences").AssertOK().AssertJSONCount("data", 1)

	bad := map[string]any{}
	for k, v := range body {
		bad[k] = v
	}

	bad["frequency"] = "hourly"
	assertProblem(t, c().PostJSON("/api/v1/recurrences", bad), 422, "validation_failed")

	bad["frequency"], bad["interval"] = "monthly", 0
	assertProblem(t, c().PostJSON("/api/v1/recurrences", bad), 422, "validation_failed")

	bad["interval"], bad["account_uid"] = 1, "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	assertProblem(t, c().PostJSON("/api/v1/recurrences", bad), 404, "not_found")

	// A field left out stays; category_uid "" removes the category; active false pauses.
	c().PatchJSON("/api/v1/recurrences/"+uid, map[string]any{"description": "Music", "category_uid": "", "active": false}).AssertOK().
		AssertJSONPath("description", "Music").AssertJSONPath("category_uid", nil).AssertJSONPath("active", false).
		AssertJSONPath("amount_cents", float64(4990))

	c().Delete("/api/v1/recurrences/" + uid).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/recurrences").AssertOK().AssertJSONCount("data", 0)
	assertProblem(t, c().Delete("/api/v1/recurrences/"+uid), 404, "not_found")
}

func TestAPIGoals(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	res := c().PostJSON("/api/v1/goals", map[string]any{"name": "Trip", "target_cents": 100000, "target_date": "2099-12-31"}).AssertCreated()
	res.AssertJSONPath("name", "Trip").AssertJSONPath("saved_cents", float64(0)).AssertJSONPath("remaining_cents", float64(100000))
	uid := uidOf(t, res, "uid")

	assertProblem(t, c().PostJSON("/api/v1/goals", map[string]any{"name": "Bad", "target_cents": 0}), 422, "validation_failed")

	add := c().PostJSON("/api/v1/goals/"+uid+"/contributions", map[string]any{"amount_cents": 25000, "date": "2026-11-01", "note": "Bonus"}).AssertCreated()
	add.AssertJSONPath("amount_cents", float64(25000)).AssertJSONPath("goal_uid", uid)
	part := uidOf(t, add, "uid")

	assertProblem(t, c().PostJSON("/api/v1/goals/"+uid+"/contributions", map[string]any{"amount_cents": 0}), 422, "validation_failed")

	list := c().Get("/api/v1/goals").AssertOK()
	list.AssertJSONPath("data.0.saved_cents", float64(25000)).AssertJSONPath("data.0.percent", float64(25)).AssertJSONPath("data.0.contributions", float64(1))

	c().PatchJSON("/api/v1/goals/"+uid, map[string]any{"target_cents": 50000, "target_date": ""}).AssertOK().
		AssertJSONPath("target_cents", float64(50000)).AssertJSONPath("name", "Trip").AssertJSONPath("target_date", nil).
		AssertJSONPath("percent", float64(50))

	c().Delete("/api/v1/goals/" + uid + "/contributions/" + part).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/goals").AssertOK().AssertJSONPath("data.0.contributions", float64(0))
	assertProblem(t, c().Delete("/api/v1/goals/"+uid+"/contributions/"+part), 404, "not_found")

	// An archived goal takes no contribution until it is brought back.
	c().PostJSON("/api/v1/goals/"+uid+"/archive", nil).AssertOK().AssertJSONPath("archived", true)
	assertProblem(t, c().PostJSON("/api/v1/goals/"+uid+"/contributions", map[string]any{"amount_cents": 100}), 409, "conflict")
	c().PostJSON("/api/v1/goals/"+uid+"/unarchive", nil).AssertOK().AssertJSONPath("archived", false)

	c().Delete("/api/v1/goals/" + uid).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/goals").AssertOK().AssertJSONCount("data", 0)
}

// A member records money but does not manage data: on the goals page a member
// adds contributions and cannot make goals, and the token keeps that split.
func TestAPIGoalsKeepTheRoleSplit(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)

	uid := uidOf(t, f.client().PostJSON("/api/v1/goals", map[string]any{"name": "Trip", "target_cents": 100000}).AssertCreated(), "uid")

	member := newVerifiedUser(t, f.app, "Mia", "mia@example.com")
	join(t, f.app, f.who, member, models.RoleMember)

	c := apiClient(t, f.app, rawToken(t, f.app, member, f.who.Workspace.ID, "goals:write", "goals:read"))

	part := uidOf(t, c.PostJSON("/api/v1/goals/"+uid+"/contributions", map[string]any{"amount_cents": 500}).AssertCreated(), "uid")
	c.Delete("/api/v1/goals/" + uid + "/contributions/" + part).AssertStatus(http.StatusNoContent)

	assertProblem(t, c.PostJSON("/api/v1/goals", map[string]any{"name": "Mine", "target_cents": 100}), 403, "forbidden")
	assertProblem(t, c.PostJSON("/api/v1/goals/"+uid+"/archive", nil), 403, "forbidden")
}

func TestAPIReports(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	c().PostJSON("/api/v1/transactions", f.expense(2000, "2026-11-10")).AssertCreated()

	income := f.expense(9000, "2026-11-02")
	income["kind"], income["category_uid"], income["tags"] = "income", nil, nil
	c().PostJSON("/api/v1/transactions", income).AssertCreated()

	period := "?period=custom&from=2026-11-01&to=2026-11-30"

	flow := c().Get("/api/v1/reports/cash-flow" + period).AssertOK()
	flow.AssertJSONPath("currency", "USD").AssertJSONPath("totals.income_cents", float64(9000)).
		AssertJSONPath("totals.expense_cents", float64(2000)).AssertJSONPath("totals.net_cents", float64(7000)).
		AssertJSONPath("months.0.month", "2026-11")

	cats := c().Get("/api/v1/reports/categories" + period).AssertOK()
	cats.AssertJSONPath("kind", "expense").AssertJSONPath("total_cents", float64(2000)).AssertJSONPath("rows.0.uid", f.food.UID)
	c().Get("/api/v1/reports/categories"+period+"&kind=income").AssertOK().AssertJSONPath("total_cents", float64(9000))

	c().Get("/api/v1/reports/accounts"+period).AssertOK().AssertJSONPath("rows.0.uid", f.checking.UID).
		AssertJSONPath("rows.0.net_cents", float64(7000))
	c().Get("/api/v1/reports/tags"+period).AssertOK().AssertJSONCount("rows", 2)
	c().Get("/api/v1/reports/net-worth"+period).AssertOK().AssertJSONPath("points.0.month", "2026-11")
	c().Get("/api/v1/reports/compare"+period).AssertOK().AssertJSONPath("rows.0.key", "income").AssertJSONPath("previous.to", "2026-10-31")

	assertProblem(t, c().Get("/api/v1/reports/cash-flow?period=custom&from=2026-12-01&to=2026-11-01"), 422, "validation_failed")
	assertProblem(t, c().Get("/api/v1/reports/cash-flow?period=forever"), 422, "validation_failed")
}

func TestAPIExport(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	c().PostJSON("/api/v1/transactions", map[string]any{
		"account_uid": f.checking.UID, "kind": "expense", "amount_cents": 1250, "date": "2026-11-10", "description": "=SUM(A1)",
	}).AssertCreated()

	csv := c().Get("/api/v1/export/transactions?from=2026-11-01&to=2026-11-30&delimiter=semicolon&bom=true").AssertOK().
		AssertHeaderContains("Content-Type", "text/csv").AssertHeaderContains("Content-Disposition", "attachment")

	body := csv.Body()
	if !strings.HasPrefix(body, "\xEF\xBB\xBF") || !strings.Contains(body, "uid;date;account") || !strings.Contains(body, "'=SUM(A1)") {
		t.Errorf("the CSV is not the one of the web export (BOM, semicolons, spreadsheet-safe cells):\n%s", body)
	}

	assertProblem(t, c().Get("/api/v1/export/transactions?delimiter=tab"), 422, "validation_failed")
	assertProblem(t, c().Get("/api/v1/export/transactions?account=01ARZ3NDEKTSV4RRFFQ69G5FAV"), 404, "not_found")

	doc := c().Get("/api/v1/export/workspace").AssertOK().AssertHeaderContains("Content-Type", "application/json")
	if _, ok := doc.JSONPath("transactions"); !ok {
		t.Errorf("the workspace export has no transactions: %.200s", doc.Body())
	}

	// The downloads of a workspace share one limit with the web exports.
	limited := 0

	for range 12 {
		if c().Get("/api/v1/export/transactions").Status() == http.StatusTooManyRequests {
			limited++
		}
	}

	if limited == 0 {
		t.Error("twelve more downloads were never limited")
	}
}
