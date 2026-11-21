package tests

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// apiFixture is a Pro workspace with a token of every scope, two accounts and a category.
type apiFixture struct {
	t        *testing.T
	app      *vukemtest.App
	who      who
	token    string
	checking models.Account
	savings  models.Account
	food     models.Category
}

func newAPIFixture(t *testing.T) apiFixture {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	f := apiFixture{t: t, app: app, who: ana, token: fullToken(t, app, ana), checking: firstOf[models.Account](t, app, ana, "")}
	f.client().PostJSON("/api/v1/accounts", map[string]any{"name": "Savings", "kind": "savings"}).AssertCreated()
	f.savings = firstOf[models.Account](t, app, ana, "Savings")
	f.food = firstOf[models.Category](t, app, ana, "Groceries")

	return f
}

// client is a fresh client that sends the fixture token and nothing else.
func (f apiFixture) client() *vukemtest.Client { return apiClient(f.t, f.app, f.token) }

func (f apiFixture) expense(cents int, date string) map[string]any {
	return map[string]any{
		"account_uid": f.checking.UID, "category_uid": f.food.UID, "kind": "expense", "amount_cents": cents,
		"date": date, "description": "Lunch", "tags": []string{"work", "Food"},
	}
}

func TestAPITransactions(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	res := c().PostJSON("/api/v1/transactions", f.expense(1250, "2026-11-21")).AssertCreated()
	res.AssertJSONPath("amount_cents", float64(1250)).AssertJSONPath("currency", "USD").AssertJSONPath("date", "2026-11-21").
		AssertJSONPath("account_uid", f.checking.UID).AssertJSONPath("category_uid", f.food.UID).AssertJSONCount("tags", 2)
	uid := uidOf(t, res, "uid")

	c().PostJSON("/api/v1/transactions", f.expense(300, "2026-10-02")).AssertCreated()

	income := f.expense(5000, "2026-11-01")
	income["kind"], income["category_uid"], income["tags"] = "income", nil, nil
	c().PostJSON("/api/v1/transactions", income).AssertCreated()

	// The balance follows.
	c().Get("/api/v1/accounts/"+f.checking.UID).AssertOK().AssertJSONPath("balance_cents", float64(5000-1250-300))

	// Filters and the page envelope.
	list := c().Get("/api/v1/transactions").AssertOK()
	list.AssertJSONCount("data", 3).AssertJSONPath("meta.total", float64(3)).AssertJSONPath("meta.page", float64(1)).
		AssertJSONPath("meta.per_page", float64(50)).AssertJSONPath("data.0.date", "2026-11-21")
	c().Get("/api/v1/transactions?kind=expense&from=2026-11-01").AssertOK().AssertJSONCount("data", 1)
	c().Get("/api/v1/transactions?q=lunch&min_cents=1000").AssertOK().AssertJSONCount("data", 2)
	c().Get("/api/v1/transactions?account="+f.savings.UID).AssertOK().AssertJSONCount("data", 0)
	c().Get("/api/v1/transactions?category="+f.food.UID+"&max_cents=500").AssertOK().AssertJSONCount("data", 1)
	assertProblem(t, c().Get("/api/v1/transactions?account=01ARZ3NDEKTSV4RRFFQ69G5FAV"), 404, "not_found")
	c().Get("/api/v1/transactions?page=2").AssertOK().AssertJSONCount("data", 0)

	// Edit one field; the rest stays, tags are kept unless named.
	c().PatchJSON("/api/v1/transactions/"+uid, map[string]any{"amount_cents": 2000}).AssertOK().
		AssertJSONPath("amount_cents", float64(2000)).AssertJSONPath("description", "Lunch").AssertJSONCount("tags", 2)
	c().PatchJSON("/api/v1/transactions/"+uid, map[string]any{"tags": []string{}, "category_uid": ""}).AssertOK().
		AssertJSONCount("tags", 0).AssertJSONPath("category_uid", nil)

	// The web rules apply: an amount is positive, a date is a date, a category has the kind of the entry.
	assertProblem(t, c().PostJSON("/api/v1/transactions", f.expense(0, "2026-11-21")), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/transactions", f.expense(100, "21/11/2026")), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/transactions", map[string]any{
		"account_uid": f.checking.UID, "kind": "income", "amount_cents": 1, "date": "2026-11-21", "category_uid": f.food.UID,
	}), 422, "validation_failed")

	// An archived account takes no new entry.
	c().PostJSON("/api/v1/accounts/"+f.savings.UID+"/archive", nil).AssertOK()

	toSavings := f.expense(100, "2026-11-21")
	toSavings["account_uid"] = f.savings.UID
	assertProblem(t, c().PostJSON("/api/v1/transactions", toSavings), 422, "validation_failed")

	// A client uid is an idempotent creation.
	mine := models.NewUID()
	body := f.expense(700, "2026-11-20")
	body["uid"] = mine
	c().PostJSON("/api/v1/transactions", body).AssertCreated()
	c().PostJSON("/api/v1/transactions", body).AssertOK().AssertJSONPath("uid", mine)

	body["amount_cents"] = 701
	assertProblem(t, c().PostJSON("/api/v1/transactions", body), 409, "conflict")

	if n := rowCount(t, f.app, "transactions"); n != 4 {
		t.Errorf("transactions = %d, want 4", n)
	}

	c().Delete("/api/v1/transactions/" + uid).AssertStatus(http.StatusNoContent)
	assertProblem(t, c().Get("/api/v1/transactions/"+uid), 404, "not_found")

	again := map[string]any{"uid": uid, "account_uid": f.checking.UID, "kind": "expense", "amount_cents": 1, "date": "2026-11-21"}
	assertProblem(t, c().PostJSON("/api/v1/transactions", again), 409, "conflict")
}

func TestAPIIdempotency(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	body := f.expense(999, "2026-11-21")

	var wg sync.WaitGroup

	statuses := make([]int, 10)

	for i := range statuses {
		wg.Add(1)

		go func() {
			defer wg.Done()

			statuses[i] = f.client().WithHeader("Idempotency-Key", "key-1").PostJSON("/api/v1/transactions", body).Status()
		}()
	}

	wg.Wait()

	for i, s := range statuses {
		if s != http.StatusCreated && s != http.StatusConflict {
			t.Errorf("request %d answered %d, want 201 or a 409 while the first runs", i, s)
		}
	}

	if n := rowCount(t, f.app, "transactions"); n != 1 {
		t.Fatalf("10 parallel calls with one key made %d transactions, want 1", n)
	}

	// After it finished, a repeat is the same answer, flagged.
	f.client().WithHeader("Idempotency-Key", "key-1").PostJSON("/api/v1/transactions", body).AssertCreated().
		AssertHeader("Idempotent-Replayed", "true")

	// Another request under the same key is refused; another token does not share the key.
	f.client().WithHeader("Idempotency-Key", "key-1").PostJSON("/api/v1/transactions", f.expense(1, "2026-11-21")).
		AssertStatus(http.StatusUnprocessableEntity)

	second := fullToken(t, f.app, f.who)
	apiClient(t, f.app, second).WithHeader("Idempotency-Key", "key-1").PostJSON("/api/v1/transactions", body).AssertCreated()

	if n := rowCount(t, f.app, "transactions"); n != 2 {
		t.Errorf("transactions = %d, want 2", n)
	}
}

func TestAPIIfMatch(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	res := c().PostJSON("/api/v1/transactions", f.expense(1250, "2026-11-21")).AssertCreated()
	uid := uidOf(t, res, "uid")
	v, _ := res.JSONPath("version")
	version, _ := v.(json.Number).Int64()

	stale := strconv.FormatInt(version+5, 10)
	r := c().WithHeader("If-Match", stale).PatchJSON("/api/v1/transactions/"+uid, map[string]any{"amount_cents": 1})
	assertProblem(t, r, 409, "version_conflict")
	r.AssertJSONPath("current.amount_cents", float64(1250)).AssertJSONPath("current.uid", uid)
	assertProblem(t, c().WithHeader("If-Match", stale).Delete("/api/v1/transactions/"+uid), 409, "version_conflict")
	c().WithHeader("If-Match", "abc").PatchJSON("/api/v1/transactions/"+uid, map[string]any{"amount_cents": 1}).AssertStatus(http.StatusBadRequest)
	c().Get("/api/v1/transactions/"+uid).AssertOK().AssertJSONPath("amount_cents", float64(1250))

	// The right version, quoted or not, goes through and the version moves on.
	quoted := "\"" + strconv.FormatInt(version, 10) + "\""
	ok := c().WithHeader("If-Match", quoted).PatchJSON("/api/v1/transactions/"+uid, map[string]any{"amount_cents": 1300}).AssertOK()
	nv, _ := ok.JSONPath("version")

	if got, _ := nv.(json.Number).Int64(); got <= version {
		t.Errorf("version stayed at %d after a change", got)
	}

	c().WithHeader("If-Match", strconv.FormatInt(version, 10)).PatchJSON("/api/v1/transactions/"+uid, map[string]any{"amount_cents": 1400}).
		AssertStatus(http.StatusConflict)
}

func TestAPITransfers(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)
	c := f.client

	body := map[string]any{
		"from_account_uid": f.checking.UID, "to_account_uid": f.savings.UID, "amount_cents": 50000, "date": "2026-11-21", "description": "Move",
	}
	res := c().PostJSON("/api/v1/transfers", body).AssertCreated()
	uid := uidOf(t, res, "uid")
	res.AssertJSONPath("from.kind", "expense").AssertJSONPath("to.kind", "income").AssertJSONPath("from.account_uid", f.checking.UID).
		AssertJSONPath("to.account_uid", f.savings.UID).AssertJSONPath("from.transfer_uid", uid)

	c().Get("/api/v1/accounts/"+f.savings.UID).AssertOK().AssertJSONPath("balance_cents", float64(50000))

	// A transfer is neither income nor expense, and not an ordinary entry to edit.
	//SUMMARY
	legUID := uidOf(t, res, "from.uid")
	assertProblem(t, c().PatchJSON("/api/v1/transactions/"+legUID, map[string]any{"amount_cents": 1}), 422, "validation_failed")

	same := map[string]any{"from_account_uid": f.checking.UID, "to_account_uid": f.checking.UID, "amount_cents": 1, "date": "2026-11-21"}
	assertProblem(t, c().PostJSON("/api/v1/transfers", same), 422, "validation_failed")

	c().Delete("/api/v1/transfers/" + uid).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/accounts/"+f.savings.UID).AssertOK().AssertJSONPath("balance_cents", float64(0))
	assertProblem(t, c().Delete("/api/v1/transfers/"+uid), 404, "not_found")
}
