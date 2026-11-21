package tests

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func ruleForm(account int64, start string) url.Values {
	return url.Values{
		"accountId": {strconv.FormatInt(account, 10)}, "kind": {"expense"}, "amount": {"10.00"}, "description": {"Rent"},
		"frequency": {"monthly"}, "interval": {"1"}, "startDate": {start}, "backfill": {"true"}, "active": {"true"},
	}
}

func TestRecurrencesHTTP(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc := firstAccount(t, app, ana)
	vera := viewerOf(t, app, ana)

	// A viewer cannot make a rule or an installment purchase.
	vera.Client.PostForm("/recurrences", ruleForm(acc, "2027-01-31")).AssertForbidden()
	vera.Client.PostForm("/installments", url.Values{"accountId": {strconv.FormatInt(acc, 10)}}).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "recurrences", 0)

	ana.Client.PostForm("/recurrences", ruleForm(acc, "2027-01-31")).AssertRedirect("/recurrences")
	vukemtest.AssertDatabaseCount(t, app, "recurrences", 1)
	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)

	ana.Client.Inertia("").Get("/recurrences").AssertInertia("Recurrences").
		PropEquals("recurrences.0.description", "Rent").PropEquals("today", "2027-03-31")
	ana.Client.Inertia("").Get("/dashboard").AssertInertia("Dashboard")

	rule := firstOf[models.Recurrence](t, app, ana, "")

	ana.Client.PatchJSON("/recurrences/"+strconv.FormatInt(rule.ID, 10), map[string]any{
		"accountId": strconv.FormatInt(acc, 10), "kind": "expense", "amount": "12.50", "description": "Rent",
		"frequency": "monthly", "interval": "1", "startDate": "2027-01-31", "active": true,
	}).AssertRedirect("/recurrences")

	vukemtest.AssertDatabaseHas(t, app, "recurrences", map[string]any{"amount_cents": 1250})

	// Generated transactions keep their value.
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"amount_cents": 1000, "occurrence_on": "2027-02-28"})

	ana.Client.PostForm("/recurrences", url.Values{
		"accountId": {strconv.FormatInt(acc, 10)}, "kind": {"expense"}, "amount": {"1.00"}, "frequency": {"hourly"},
		"interval": {"1"}, "startDate": {"2027-04-01"},
	}).AssertRedirect("/recurrences")
	vukemtest.AssertDatabaseCount(t, app, "recurrences", 1)

	ana.Client.PostForm("/installments", url.Values{
		"accountId": {strconv.FormatInt(acc, 10)}, "total": {"100.00"}, "parts": {"3"}, "firstDate": {"2027-04-30"}, "description": {"Sofa"},
	}).AssertRedirect("/transactions")
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{
		"description": "Sofa", "amount_cents": 3334, "installment_no": 1, "installment_total": 3, "occurred_on": "2027-04-30",
	})

	part := firstOfWhere[models.Transaction](t, app, ana, "installment_no", 2)
	ana.Client.Delete("/transactions/" + strconv.FormatInt(part.ID, 10) + "/installments").AssertRedirect("/transactions")

	// This part and the following are deleted; the first one stays.
	if n := rowCountWhere(t, app, "transactions", "description = 'Sofa' AND deleted_at IS NULL"); n != 1 {
		t.Errorf("%d Sofa parts live, want 1", n)
	}

	ana.Client.Delete("/recurrences/" + strconv.FormatInt(rule.ID, 10)).AssertRedirect("/recurrences")
	ana.Client.Inertia("").Get("/recurrences").AssertInertia("Recurrences").PropEquals("recurrences", []any{})
}
