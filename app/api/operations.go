package api

import (
	"net/http"
	"sync"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// BasePath is where the API lives.
const BasePath = "/api/v1"

// Operation is one operation of the API: its route, the scope a token needs
// and what it does to the data. The table below declares the REST side of each
// operation (decision D-017): the method, the path and the status. An operation
// of the registry (app/operations) takes its scope and summary from there, so
// they are declared once; the few operations that are REST only (the token and
// the sync protocol) carry their own. The routes, the OpenAPI document and the
// scope matrix test all read it, and a route or a row without the other fails
// a test.
type Operation struct {
	// Name is the route name and the OpenAPI operationId.
	Name   string
	Method string
	// Path is relative to BasePath.
	Path string
	// Scope is the scope a token needs; "" for an operation of any valid token.
	// Left out for an operation of the registry.
	Scope string
	// Registered is set by Operations for an operation of the registry.
	Registered bool
	Summary    string
	Tag        string
	// Status is the success status; 0 takes the framework default.
	Status int
	// MaxBody caps the request body in bytes; 0 leaves the server default.
	MaxBody int64
}

// Read reports whether the operation only reads.
func (o Operation) Read() bool { return o.Method == http.MethodGet }

var table = []Operation{ //nolint:gochecknoglobals // a constant table
	{Name: "me.show", Method: "GET", Path: "/me", Summary: "The token, its scopes and the workspace it works in", Tag: "Token"},

	{Name: "accounts.list", Method: "GET", Path: "/accounts", Tag: "Accounts"},
	{Name: "accounts.show", Method: "GET", Path: "/accounts/{uid}", Tag: "Accounts"},
	{Name: "accounts.create", Method: "POST", Path: "/accounts", Tag: "Accounts", Status: http.StatusCreated},
	{Name: "accounts.update", Method: "PATCH", Path: "/accounts/{uid}", Tag: "Accounts"},
	{Name: "accounts.archive", Method: "POST", Path: "/accounts/{uid}/archive", Tag: "Accounts", Status: http.StatusOK},

	{Name: "categories.list", Method: "GET", Path: "/categories", Tag: "Categories"},
	{Name: "categories.show", Method: "GET", Path: "/categories/{uid}", Tag: "Categories"},
	{Name: "categories.create", Method: "POST", Path: "/categories", Tag: "Categories", Status: http.StatusCreated},
	{Name: "categories.update", Method: "PATCH", Path: "/categories/{uid}", Tag: "Categories"},
	{Name: "categories.archive", Method: "POST", Path: "/categories/{uid}/archive", Tag: "Categories", Status: http.StatusOK},

	{Name: "tags.list", Method: "GET", Path: "/tags", Tag: "Tags"},
	{Name: "tags.create", Method: "POST", Path: "/tags", Tag: "Tags", Status: http.StatusCreated},
	{Name: "tags.update", Method: "PATCH", Path: "/tags/{uid}", Tag: "Tags"},
	{Name: "tags.delete", Method: "DELETE", Path: "/tags/{uid}", Tag: "Tags"},

	{Name: "transactions.list", Method: "GET", Path: "/transactions", Tag: "Transactions"},
	{Name: "transactions.show", Method: "GET", Path: "/transactions/{uid}", Tag: "Transactions"},
	{Name: "transactions.create", Method: "POST", Path: "/transactions", Tag: "Transactions", Status: http.StatusCreated},
	{Name: "transactions.update", Method: "PATCH", Path: "/transactions/{uid}", Tag: "Transactions"},
	{Name: "transactions.delete", Method: "DELETE", Path: "/transactions/{uid}", Tag: "Transactions"},

	{Name: "transfers.create", Method: "POST", Path: "/transfers", Tag: "Transfers", Status: http.StatusCreated},
	{Name: "transfers.delete", Method: "DELETE", Path: "/transfers/{uid}", Tag: "Transfers"},

	{Name: "budgets.list", Method: "GET", Path: "/budgets", Tag: "Budgets"},
	{Name: "budgets.set", Method: "POST", Path: "/budgets", Tag: "Budgets", Status: http.StatusOK},
	{Name: "budgets.delete", Method: "DELETE", Path: "/budgets/{uid}", Tag: "Budgets"},

	{Name: "summary.show", Method: "GET", Path: "/summary", Tag: "Summary"},

	{Name: "cards.list", Method: "GET", Path: "/cards", Tag: "Cards"},
	{Name: "cards.show", Method: "GET", Path: "/cards/{uid}", Tag: "Cards"},
	{Name: "cards.create", Method: "POST", Path: "/cards", Tag: "Cards", Status: http.StatusCreated},
	{Name: "cards.update", Method: "PATCH", Path: "/cards/{uid}", Tag: "Cards"},
	{Name: "cards.archive", Method: "POST", Path: "/cards/{uid}/archive", Tag: "Cards", Status: http.StatusOK},
	{Name: "cards.unarchive", Method: "POST", Path: "/cards/{uid}/unarchive", Tag: "Cards", Status: http.StatusOK},

	{Name: "statements.list", Method: "GET", Path: "/cards/{uid}/statements", Tag: "Statements"},
	{Name: "statements.show", Method: "GET", Path: "/cards/{uid}/statements/{ref}", Tag: "Statements"},
	{Name: "statements.pay", Method: "POST", Path: "/cards/{uid}/statements/{ref}/pay", Tag: "Statements", Status: http.StatusCreated},

	{Name: "recurrences.list", Method: "GET", Path: "/recurrences", Tag: "Recurrences"},
	{Name: "recurrences.create", Method: "POST", Path: "/recurrences", Tag: "Recurrences", Status: http.StatusCreated},
	{Name: "recurrences.update", Method: "PATCH", Path: "/recurrences/{uid}", Tag: "Recurrences"},
	{Name: "recurrences.delete", Method: "DELETE", Path: "/recurrences/{uid}", Tag: "Recurrences"},

	{Name: "goals.list", Method: "GET", Path: "/goals", Tag: "Goals"},
	{Name: "goals.create", Method: "POST", Path: "/goals", Tag: "Goals", Status: http.StatusCreated},
	{Name: "goals.update", Method: "PATCH", Path: "/goals/{uid}", Tag: "Goals"},
	{Name: "goals.archive", Method: "POST", Path: "/goals/{uid}/archive", Tag: "Goals", Status: http.StatusOK},
	{Name: "goals.unarchive", Method: "POST", Path: "/goals/{uid}/unarchive", Tag: "Goals", Status: http.StatusOK},
	{Name: "goals.delete", Method: "DELETE", Path: "/goals/{uid}", Tag: "Goals"},
	{Name: "goals.contributions.add", Method: "POST", Path: "/goals/{uid}/contributions", Tag: "Goals", Status: http.StatusCreated},
	{Name: "goals.contributions.delete", Method: "DELETE", Path: "/goals/{uid}/contributions/{contribution_uid}", Tag: "Goals"},

	{Name: "reports.cashflow", Method: "GET", Path: "/reports/cash-flow", Tag: "Reports"},
	{Name: "reports.categories", Method: "GET", Path: "/reports/categories", Tag: "Reports"},
	{Name: "reports.accounts", Method: "GET", Path: "/reports/accounts", Tag: "Reports"},
	{Name: "reports.tags", Method: "GET", Path: "/reports/tags", Tag: "Reports"},
	{Name: "reports.networth", Method: "GET", Path: "/reports/net-worth", Tag: "Reports"},
	{Name: "reports.compare", Method: "GET", Path: "/reports/compare", Tag: "Reports"},

	{Name: "export.transactions", Method: "GET", Path: "/export/transactions", Tag: "Export"},
	{Name: "export.workspace", Method: "GET", Path: "/export/workspace", Tag: "Export"},
	{Name: "sync.info", Method: "GET", Path: "/sync/info", Scope: ScopeSync, Summary: "The sync protocol versions, the entities and fields this server knows, and its limits", Tag: "Sync"},
	{Name: "sync.devices.create", Method: "POST", Path: "/sync/devices", Scope: ScopeSync, Summary: "Register a device for the calling token", Tag: "Sync", Status: http.StatusCreated},
	{Name: "sync.devices.list", Method: "GET", Path: "/sync/devices", Scope: ScopeSync, Summary: "List the devices the caller registered in this workspace", Tag: "Sync"},
	{Name: "sync.pull", Method: "GET", Path: "/sync/pull", Scope: ScopeSync, Summary: "Read the changes after a cursor, parents first", Tag: "Sync"},
	{Name: "sync.snapshot", Method: "GET", Path: "/sync/snapshot", Scope: ScopeSync, Summary: "Read the live rows of one entity to rebuild a device", Tag: "Sync"},
	{Name: "sync.push", Method: "POST", Path: "/sync/push", Scope: ScopeSync, Summary: "Send the mutations a device made offline", Tag: "Sync", Status: http.StatusOK, MaxBody: 1 << 20},
}

// described is the registry with no services: it only describes the operations.
var described = sync.OnceValue(func() *operations.Catalog { //nolint:gochecknoglobals // built once
	return operations.NewCatalog(operations.Config{}, operations.Services{})
})

// Describe returns the registry that names every operation: the REST table
// reads its scopes and summaries, the inventory command its whole metadata.
func Describe() *operations.Catalog { return described() }

func resolve(o Operation) Operation {
	if m, ok := described().Named(o.Name); ok {
		o.Scope, o.Summary, o.Registered = m.Scope, m.Summary, true
	}

	return o
}

// Operations lists every operation of the API.
func Operations() []Operation {
	out := make([]Operation, 0, len(table))
	for _, o := range table {
		out = append(out, resolve(o))
	}

	return out
}

// OperationNamed finds an operation by its route name.
func OperationNamed(name string) (Operation, bool) {
	for _, o := range table {
		if o.Name == name {
			return resolve(o), true
		}
	}

	return Operation{}, false
}
