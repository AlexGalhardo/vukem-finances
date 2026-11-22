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

	{Name: "transactions.list", Method: "GET", Path: "/transactions", Scope: ScopeTransactionsRead, Summary: "List transactions, newest first, with filters", Tag: "Transactions"},
	{Name: "transactions.show", Method: "GET", Path: "/transactions/{uid}", Scope: ScopeTransactionsRead, Summary: "Read a transaction", Tag: "Transactions"},
	{Name: "transactions.create", Method: "POST", Path: "/transactions", Scope: ScopeTransactionsWrite, Summary: "Record a transaction", Tag: "Transactions", Status: http.StatusCreated},
	{Name: "transactions.update", Method: "PATCH", Path: "/transactions/{uid}", Scope: ScopeTransactionsWrite, Summary: "Change a transaction", Tag: "Transactions"},
	{Name: "transactions.delete", Method: "DELETE", Path: "/transactions/{uid}", Scope: ScopeTransactionsWrite, Summary: "Delete a transaction", Tag: "Transactions"},

	{Name: "transfers.create", Method: "POST", Path: "/transfers", Scope: ScopeTransfersWrite, Summary: "Move money between two accounts", Tag: "Transfers", Status: http.StatusCreated},
	{Name: "transfers.delete", Method: "DELETE", Path: "/transfers/{uid}", Scope: ScopeTransfersWrite, Summary: "Delete a transfer and both of its legs", Tag: "Transfers"},

	{Name: "budgets.list", Method: "GET", Path: "/budgets", Scope: ScopeBudgetsRead, Summary: "List the budgets with what the month spent", Tag: "Budgets"},
	{Name: "budgets.set", Method: "POST", Path: "/budgets", Scope: ScopeBudgetsWrite, Summary: "Set the monthly limit of an expense category", Tag: "Budgets", Status: http.StatusOK},
	{Name: "budgets.delete", Method: "DELETE", Path: "/budgets/{uid}", Scope: ScopeBudgetsWrite, Summary: "Remove a budget", Tag: "Budgets"},

	{Name: "summary.show", Method: "GET", Path: "/summary", Scope: ScopeSummaryRead, Summary: "Income, expense, net, account balances and net worth of a month", Tag: "Summary"},
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
