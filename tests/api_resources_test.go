package tests

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestAPIAccounts(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := fullToken(t, app, ana)
	c := func() *vukemtest.Client { return apiClient(t, app, token) }

	// The Free plan stops at one account: the web rule, with a stable code.
	assertProblem(t, c().PostJSON("/api/v1/accounts", map[string]any{"name": "Savings", "kind": "savings"}), 402, "plan_limit")

	makePro(t, app, ana)

	list := c().Get("/api/v1/accounts").AssertOK()
	list.AssertJSONCount("data", 1).AssertJSONPath("data.0.currency", "USD").AssertJSONPath("data.0.balance_cents", float64(0))

	res := c().PostJSON("/api/v1/accounts", map[string]any{"name": " Savings ", "kind": "savings", "opening_cents": 5000}).AssertCreated()
	res.AssertJSONPath("name", "Savings").AssertJSONPath("opening_cents", float64(5000)).AssertJSONPath("balance_cents", float64(5000))
	uid := uidOf(t, res, "uid")

	c().Get("/api/v1/accounts/"+uid).AssertOK().AssertJSONPath("kind", "savings")
	assertProblem(t, c().Get("/api/v1/accounts/01ARZ3NDEKTSV4RRFFQ69G5FAV"), 404, "not_found")

	// Money is an integer: a decimal is refused, and so is an unknown kind.
	assertProblem(t, c().PostJSON("/api/v1/accounts", map[string]any{"name": "X", "kind": "cash", "opening_cents": 12.5}), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/accounts", map[string]any{"name": "X", "kind": "bank"}), 422, "validation_failed")

	// A client uid is an idempotent creation.
	mine := models.NewUID()
	body := map[string]any{"uid": mine, "name": "Cash", "kind": "cash"}
	c().PostJSON("/api/v1/accounts", body).AssertCreated().AssertJSONPath("uid", mine)
	c().PostJSON("/api/v1/accounts", body).AssertOK().AssertJSONPath("uid", mine)

	if n := rowCount(t, app, "accounts"); n != 3 {
		t.Errorf("accounts = %d, want 3 after a repeated uid", n)
	}

	assertProblem(t, c().PostJSON("/api/v1/accounts", map[string]any{"uid": mine, "name": "Other", "kind": "cash"}), 409, "conflict")
	assertProblem(t, c().PostJSON("/api/v1/accounts", map[string]any{"uid": "abc", "name": "Y", "kind": "cash"}), 422, "validation_failed")

	// Edit with a version check, then archive.
	v, _ := res.JSONPath("version")
	cur, _ := v.(json.Number).Int64()

	stale := c().WithHeader("If-Match", "999").PatchJSON("/api/v1/accounts/"+uid, map[string]any{"name": "Nope"})
	assertProblem(t, stale, 409, "version_conflict")
	stale.AssertJSONPath("current.name", "Savings")

	ok := c().WithHeader("If-Match", strconv.FormatInt(cur, 10)).PatchJSON("/api/v1/accounts/"+uid, map[string]any{"name": "Reserve", "opening_cents": 7000}).AssertOK()
	ok.AssertJSONPath("name", "Reserve").AssertJSONPath("kind", "savings").AssertJSONPath("balance_cents", float64(7000))

	c().PostJSON("/api/v1/accounts/"+uid+"/archive", nil).AssertOK().AssertJSONPath("archived", true)
}

func TestAPICategories(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := fullToken(t, app, ana)
	c := func() *vukemtest.Client { return apiClient(t, app, token) }

	list := c().Get("/api/v1/categories").AssertOK()
	if _, ok := list.JSONPath("data.0.uid"); !ok {
		t.Fatalf("a new workspace has no categories: %s", list.Body())
	}

	parent := uidOf(t, c().PostJSON("/api/v1/categories", map[string]any{"name": "Pets", "kind": "expense", "color": "green", "icon": "paw"}).AssertCreated(), "uid")
	child := c().PostJSON("/api/v1/categories", map[string]any{"name": "Vet", "parent_uid": parent}).AssertCreated()
	child.AssertJSONPath("parent_uid", parent).AssertJSONPath("kind", "expense")

	assertProblem(t, c().PostJSON("/api/v1/categories", map[string]any{"name": "Pets", "kind": "expense"}), 409, "conflict")
	assertProblem(t, c().PostJSON("/api/v1/categories", map[string]any{"name": "Odd", "kind": "expense", "color": "#ff0000"}), 422, "validation_failed")
	assertProblem(t, c().PostJSON("/api/v1/categories", map[string]any{"name": "Orphan", "kind": "expense", "parent_uid": "01ARZ3NDEKTSV4RRFFQ69G5FAV"}), 404, "not_found")

	mine := models.NewUID()
	body := map[string]any{"uid": mine, "name": "Fuel", "kind": "expense"}
	c().PostJSON("/api/v1/categories", body).AssertCreated()
	c().PostJSON("/api/v1/categories", body).AssertOK().AssertJSONPath("uid", mine)

	c().PatchJSON("/api/v1/categories/"+parent, map[string]any{"color": "blue"}).AssertOK().
		AssertJSONPath("color", "blue").AssertJSONPath("icon", "paw").AssertJSONPath("name", "Pets")
	c().PostJSON("/api/v1/categories/"+mine+"/archive", nil).AssertOK().AssertJSONPath("archived", true)
	c().Get("/api/v1/categories/"+mine).AssertOK().AssertJSONPath("archived", true)
}

func TestAPITags(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := fullToken(t, app, ana)
	c := func() *vukemtest.Client { return apiClient(t, app, token) }

	mine := models.NewUID()
	body := map[string]any{"uid": mine, "name": "Trip", "color": "blue"}
	c().PostJSON("/api/v1/tags", body).AssertCreated().AssertJSONPath("name", "Trip")
	c().PostJSON("/api/v1/tags", body).AssertOK().AssertJSONPath("uid", mine)
	assertProblem(t, c().PostJSON("/api/v1/tags", map[string]any{"name": "trip"}), 409, "conflict")

	c().Get("/api/v1/tags").AssertOK().AssertJSONCount("data", 1)
	c().PatchJSON("/api/v1/tags/"+mine, map[string]any{"name": "Holiday"}).AssertOK().
		AssertJSONPath("name", "Holiday").AssertJSONPath("color", "blue")

	assertProblem(t, c().WithHeader("If-Match", "99").Delete("/api/v1/tags/"+mine), 409, "version_conflict")
	c().Delete("/api/v1/tags/" + mine).AssertStatus(http.StatusNoContent)
	c().Get("/api/v1/tags").AssertOK().AssertJSONCount("data", 0)
}
