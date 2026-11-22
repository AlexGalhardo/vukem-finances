// Package api is the public JSON API: scoped tokens, the middleware chain and
// the REST adapter over the operation registry (docs/api.md). A handler binds
// its input, calls operations.Invoke and returns the output; no business rule
// lives here.
package api

import "github.com/AlexGalhardo/vukem-finances/app/operations"

// The scopes a token can hold (declared with the operations, see
// operations/scopes.go). A scope is a ceiling: what a request may do is the
// scope intersected with the ability of the role of the token's person.
const (
	ScopeAccountsRead      = operations.ScopeAccountsRead
	ScopeAccountsWrite     = operations.ScopeAccountsWrite
	ScopeCategoriesRead    = operations.ScopeCategoriesRead
	ScopeCategoriesWrite   = operations.ScopeCategoriesWrite
	ScopeTagsRead          = operations.ScopeTagsRead
	ScopeTagsWrite         = operations.ScopeTagsWrite
	ScopeTransactionsRead  = operations.ScopeTransactionsRead
	ScopeTransactionsWrite = operations.ScopeTransactionsWrite
	ScopeTransfersWrite    = operations.ScopeTransfersWrite
	ScopeBudgetsRead       = operations.ScopeBudgetsRead
	ScopeBudgetsWrite      = operations.ScopeBudgetsWrite
	ScopeSummaryRead       = operations.ScopeSummaryRead
	ScopeCardsRead         = operations.ScopeCardsRead
	ScopeCardsWrite        = operations.ScopeCardsWrite
	ScopeStatementsRead    = operations.ScopeStatementsRead
	ScopeStatementsWrite   = operations.ScopeStatementsWrite
	ScopeRecurrencesRead   = operations.ScopeRecurrencesRead
	ScopeRecurrencesWrite  = operations.ScopeRecurrencesWrite
	ScopeGoalsRead         = operations.ScopeGoalsRead
	ScopeGoalsWrite        = operations.ScopeGoalsWrite
	ScopeReportsRead       = operations.ScopeReportsRead
	ScopeExportRead        = operations.ScopeExportRead
	ScopeSync              = operations.ScopeSync
)

// Scope is one scope and the role ability it needs.
type Scope = operations.Scope

// Scopes lists every scope in display order.
func Scopes() []Scope { return operations.Scopes() }

// ValidScope reports whether name is a scope a token can be issued with.
func ValidScope(name string) bool { return operations.ValidScope(name) }

// AbilityOf is the role ability a scope needs; "" for an unknown scope.
func AbilityOf(scope string) string { return operations.AbilityOf(scope) }
