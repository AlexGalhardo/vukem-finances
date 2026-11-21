package tests

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// crossTenantCase is one id-taking route, called by a user of workspace B
// with an id owned by workspace A.
type crossTenantCase struct {
	method string
	// body is sent as JSON; it is valid for the route so only the foreign id can be the reason to refuse.
	body func(ids map[string]int64) map[string]any
	// table and param name the row that must survive the call.
	table, param string
}

// crossTenantExempt are the workspace routes with a path parameter that name
// no row of another workspace; each says why.
var crossTenantExempt = map[string]string{
	"workspaces.switch": "answers a redirect and the next request re-checks the membership; see TestTenancyIsolatesWorkspaces",
	"billing.checkout":  "the parameter is a plan name, not an id",
	"billing.pix.show":  "the parameter is a ULID reference, not a row id; TestPixPaymentOfAnotherWorkspaceIsNotFound proves the 404",
}

var crossTenantCases = map[string]crossTenantCase{
	"transactions.update": {
		method: http.MethodPatch, table: "transactions", param: "transaction",
		body: func(ids map[string]int64) map[string]any {
			return map[string]any{
				"accountId": strconv.FormatInt(ids["bobAccount"], 10), "kind": "expense", "amount": "1.00", "date": "2026-01-10",
			}
		},
	},
	"transactions.destroy":     {method: http.MethodDelete, table: "transactions", param: "transaction"},
	"transactions.tags.attach": {method: http.MethodPost, table: "transaction_tags", param: "transaction"},
	"transactions.tags.detach": {method: http.MethodDelete, table: "transaction_tags", param: "transaction"},
	"categories.update": {
		method: http.MethodPatch, table: "categories", param: "category",
		body: func(map[string]int64) map[string]any { return map[string]any{"name": "Renamed", "color": "blue"} },
	},
	"categories.archive":   {method: http.MethodPost, table: "categories", param: "category"},
	"categories.unarchive": {method: http.MethodPost, table: "categories", param: "category"},
	"categories.destroy": {
		method: http.MethodDelete, table: "categories", param: "category",
		body: func(ids map[string]int64) map[string]any {
			return map[string]any{"reassignTo": strconv.FormatInt(ids["bobCategory"], 10)}
		},
	},
	"tags.update": {
		method: http.MethodPatch, table: "tags", param: "tag",
		body: func(map[string]int64) map[string]any { return map[string]any{"name": "Renamed", "color": "red"} },
	},
	"tags.destroy":     {method: http.MethodDelete, table: "tags", param: "tag"},
	"accounts.destroy": {method: http.MethodDelete, table: "accounts", param: "account"},
	"accounts.update": {
		method: http.MethodPatch, table: "accounts", param: "account",
		body: func(map[string]int64) map[string]any {
			return map[string]any{"name": "Renamed", "kind": "savings", "opening": "1.00"}
		},
	},
	"accounts.archive":   {method: http.MethodPost, table: "accounts", param: "account"},
	"accounts.unarchive": {method: http.MethodPost, table: "accounts", param: "account"},
	"transfers.update": {
		method: http.MethodPatch, table: "transactions", param: "transaction",
		body: func(ids map[string]int64) map[string]any {
			return map[string]any{
				"fromAccountId": strconv.FormatInt(ids["bobAccount"], 10), "toAccountId": strconv.FormatInt(ids["bobAccount"], 10),
				"amount": "1.00", "date": "2026-01-10",
			}
		},
	},
	"transfers.destroy": {method: http.MethodDelete, table: "transactions", param: "transaction"},
	"cards.show":        {method: http.MethodGet, table: "credit_cards", param: "card"},
	"cards.update": {
		method: http.MethodPatch, table: "credit_cards", param: "card",
		body: func(map[string]int64) map[string]any {
			return map[string]any{"name": "Renamed", "limit": "10.00", "closingDay": "5", "dueDay": "15"}
		},
	},
	"cards.archive":   {method: http.MethodPost, table: "credit_cards", param: "card"},
	"cards.unarchive": {method: http.MethodPost, table: "credit_cards", param: "card"},
	"cards.destroy":   {method: http.MethodDelete, table: "credit_cards", param: "card"},
	"cards.pay": {
		method: http.MethodPost, table: "transactions", param: "card",
		body: func(ids map[string]int64) map[string]any {
			return map[string]any{"fromAccountId": strconv.FormatInt(ids["bobAccount"], 10), "amount": "1.00"}
		},
	},
	"budgets.destroy": {method: http.MethodDelete, table: "budgets", param: "budget"},
	"import.confirm": {
		method: http.MethodPost, table: "imports", param: "import",
		body: func(map[string]int64) map[string]any {
			return map[string]any{
				"date": "Date", "description": "Description", "amount": "Amount", "dateFormat": "2006-01-02",
			}
		},
	},
	"import.preview": {
		method: http.MethodPost, table: "imports", param: "import",
		body: func(map[string]int64) map[string]any {
			return map[string]any{
				"date": "Date", "description": "Description", "amount": "Amount", "dateFormat": "2006-01-02",
			}
		},
	},
	"ai.key.destroy":            {method: http.MethodDelete, table: "ai_credentials", param: "credential"},
	"tokens.destroy":            {method: http.MethodDelete, table: "access_tokens", param: "token"},
	"workspace.members.destroy": {method: http.MethodDelete, table: "memberships", param: "user"},
	"workspace.members.role": {
		method: http.MethodPatch, table: "memberships", param: "user",
		body: func(map[string]int64) map[string]any { return map[string]any{"role": "viewer"} },
	},
}

// tenantMiddleware is how Router.Routes names the workspace middleware: a route
// without it is outside the workspace and has its own tests.
const tenantMiddleware = "workspaces.(*Service).Middleware"

var pathParam = regexp.MustCompile(`\{([a-z_]+)\}`)

func TestCrossTenantMatrix(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	cara := newVerifiedUser(t, app, "Cara", "cara@example.com")
	join(t, app, ana, cara, models.RoleMember)
	makePro(t, app, ana)

	accAna := firstAccount(t, app, ana)
	addTransaction(t, ana, accAna, 0, "expense", "11.00", "2026-01-10").AssertRedirect("/transactions")
	ana.Client.PostForm("/budgets", map[string][]string{
		"categoryId": {strconv.FormatInt(categoryID(t, app, ana, "Groceries"), 10)}, "limit": {"100.00"},
	}).AssertRedirect("/budgets")
	upload(t, ana, accAna, statement).AssertOK()
	ana.Client.PostJSON("/tags", map[string]any{"name": "Trip"}).AssertStatus(http.StatusCreated)
	ana.Client.PostForm("/cards", map[string][]string{
		"name": {"Visa"}, "limit": {"1000.00"}, "closingDay": {"10"}, "dueDay": {"20"},
	}).AssertRedirect("/cards")
	ana.Client.PostForm("/ai/key", map[string][]string{"provider": {"anthropic"}, "apiSecret": {"sk-placeholder-matrix-0123456789"}}).
		AssertRedirect("/ai")

	ids := map[string]int64{
		"transaction": firstOf[models.Transaction](t, app, ana, "").ID,
		"account":     accAna,
		"budget":      firstOf[models.Budget](t, app, ana, "").ID,
		"import":      firstOf[models.Import](t, app, ana, "").ID,
		"credential":  firstOf[models.AICredential](t, app, ana, "").ID,
		"user":        cara.ID,
		"card":        firstOf[models.CreditCard](t, app, ana, "").ID,
		"bobAccount":  firstAccount(t, app, bob),
		"category":    categoryID(t, app, ana, "Groceries"),
		"bobCategory": categoryID(t, app, bob, "Groceries"),
		"tag":         firstOf[models.Tag](t, app, ana, "Trip").ID,
	}

	// Token ids are strings; the row that must survive is checked after the loop (revoking keeps the row).
	anaToken := makeToken(t, app, ana, 30, "accounts:read")
	strIDs := map[string]string{"token": tokenID(anaToken)}

	// Control: the same edit is fine for the owner, so only the id is wrong for Bob.
	own := crossTenantCases["transactions.update"].body(ids)
	own["accountId"] = strconv.FormatInt(accAna, 10)
	ana.Client.PatchJSON("/transactions/"+strconv.FormatInt(ids["transaction"], 10), own).AssertOK()

	tables := map[string]int64{}
	for table := range map[string]bool{"transactions": true, "accounts": true, "budgets": true, "imports": true, "memberships": true, "ai_credentials": true, "categories": true, "tags": true, "transaction_tags": true, "credit_cards": true, "access_tokens": true} {
		tables[table] = rowCount(t, app, table)
	}

	covered := map[string]bool{}

	for _, route := range app.Router.Routes() {
		if !pathParam.MatchString(route.Path) || !slices.Contains(route.Middleware, tenantMiddleware) {
			continue
		}

		covered[route.Name] = true

		if _, exempt := crossTenantExempt[route.Name]; exempt {
			continue
		}

		tc, ok := crossTenantCases[route.Name]
		if !ok {
			t.Errorf("route %s %s (%s) takes an id and has no case in crossTenantCases (or crossTenantExempt)",
				route.Method, route.Path, route.Name)
			continue
		}

		t.Run(route.Name, func(t *testing.T) {
			if route.Method != tc.method {
				t.Fatalf("the route is %s, the case calls %s", route.Method, tc.method)
			}

			target := pathParam.ReplaceAllStringFunc(route.Path, func(p string) string {
				if p == "{ref}" {
					return "2026-11-10"
				}

				if v, ok := strIDs[p[1:len(p)-1]]; ok {
					return v
				}

				return strconv.FormatInt(ids[p[1:len(p)-1]], 10)
			})

			var body map[string]any
			if tc.body != nil {
				body = tc.body(ids)
			}

			var res *vukemtest.Response

			switch tc.method {
			case http.MethodGet:
				res = bob.Client.Get(target)
			case http.MethodDelete:
				res = bob.Client.Delete(target)
			case http.MethodPatch:
				res = bob.Client.PatchJSON(target, body)
			case http.MethodPut:
				res = bob.Client.PutJSON(target, body)
			default:
				res = bob.Client.PostJSON(target, body)
			}

			res.AssertNotFound()

			if got := rowCount(t, app, tc.table); got != tables[tc.table] {
				t.Errorf("%s has %d rows after the call, want %d", tc.table, got, tables[tc.table])
			}
		})
	}

	if n := rowCountWhere(t, app, "access_tokens", "id = '"+strIDs["token"]+"' AND revoked_at IS NOT NULL"); n != 0 {
		t.Error("Bob revoked Ana's token")
	}

	for name := range crossTenantCases {
		if !covered[name] {
			t.Errorf("case %s matches no route: remove it", name)
		}
	}
}

func rowCount(t *testing.T, app *vukemtest.App, table string) int64 {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int64
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func rowCountWhere(t *testing.T, app *vukemtest.App, table, where string) int64 {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int64
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE "+where).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func idString(id int64) string { return strconv.FormatInt(id, 10) }
