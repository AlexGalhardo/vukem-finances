package api

import (
	"net/http"
	"slices"
)

// BasePath is where the API lives.
const BasePath = "/api/v1"

// Operation is one operation of the API: its route, the scope a token needs
// and what it does to the data. The table below is the single declaration
// (decision D-017): the routes, the OpenAPI document and the scope matrix test
// all read it, and a route or a row without the other fails a test.
type Operation struct {
	// Name is the route name and the OpenAPI operationId.
	Name   string
	Method string
	// Path is relative to BasePath.
	Path string
	// Scope is the scope a token needs; "" for an operation of any valid token.
	Scope   string
	Summary string
	Tag     string
	// Status is the success status; 0 takes the framework default.
	Status int
}

// Read reports whether the operation only reads.
func (o Operation) Read() bool { return o.Method == http.MethodGet }

var operations = []Operation{ //nolint:gochecknoglobals // a constant table
	{Name: "me.show", Method: "GET", Path: "/me", Summary: "The token, its scopes and the workspace it works in", Tag: "Token"},

	{Name: "accounts.list", Method: "GET", Path: "/accounts", Scope: ScopeAccountsRead, Summary: "List the accounts with their balances", Tag: "Accounts"},
	{Name: "accounts.show", Method: "GET", Path: "/accounts/{uid}", Scope: ScopeAccountsRead, Summary: "Read an account", Tag: "Accounts"},
	{Name: "accounts.create", Method: "POST", Path: "/accounts", Scope: ScopeAccountsWrite, Summary: "Create an account", Tag: "Accounts", Status: http.StatusCreated},
	{Name: "accounts.update", Method: "PATCH", Path: "/accounts/{uid}", Scope: ScopeAccountsWrite, Summary: "Change the name, kind and opening balance of an account", Tag: "Accounts"},
	{Name: "accounts.archive", Method: "POST", Path: "/accounts/{uid}/archive", Scope: ScopeAccountsWrite, Summary: "Archive an account", Tag: "Accounts", Status: http.StatusOK},

	{Name: "categories.list", Method: "GET", Path: "/categories", Scope: ScopeCategoriesRead, Summary: "List the categories", Tag: "Categories"},
	{Name: "categories.show", Method: "GET", Path: "/categories/{uid}", Scope: ScopeCategoriesRead, Summary: "Read a category", Tag: "Categories"},
	{Name: "categories.create", Method: "POST", Path: "/categories", Scope: ScopeCategoriesWrite, Summary: "Create a category", Tag: "Categories", Status: http.StatusCreated},
	{Name: "categories.update", Method: "PATCH", Path: "/categories/{uid}", Scope: ScopeCategoriesWrite, Summary: "Change a category", Tag: "Categories"},
	{Name: "categories.archive", Method: "POST", Path: "/categories/{uid}/archive", Scope: ScopeCategoriesWrite, Summary: "Archive a category", Tag: "Categories", Status: http.StatusOK},

	{Name: "tags.list", Method: "GET", Path: "/tags", Scope: ScopeTagsRead, Summary: "List the tags", Tag: "Tags"},
	{Name: "tags.create", Method: "POST", Path: "/tags", Scope: ScopeTagsWrite, Summary: "Create a tag", Tag: "Tags", Status: http.StatusCreated},
	{Name: "tags.update", Method: "PATCH", Path: "/tags/{uid}", Scope: ScopeTagsWrite, Summary: "Rename or recolor a tag", Tag: "Tags"},
	{Name: "tags.delete", Method: "DELETE", Path: "/tags/{uid}", Scope: ScopeTagsWrite, Summary: "Delete a tag and take it off its transactions", Tag: "Tags"},

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
}

// Operations lists every operation of the API.
func Operations() []Operation { return slices.Clone(operations) }

// OperationNamed finds an operation by its route name.
func OperationNamed(name string) (Operation, bool) {
	for _, o := range operations {
		if o.Name == name {
			return o, true
		}
	}

	return Operation{}, false
}
