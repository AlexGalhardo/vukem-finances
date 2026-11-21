// Package api is the public JSON API: scoped tokens, the middleware chain and
// the REST adapter over the finance services (docs/api.md). A handler binds
// its input, calls a service of app/finance or app/workspaces and returns the
// output; no business rule lives here.
package api

import (
	"slices"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// The scopes a token can hold. A scope is a ceiling: what a request may do is
// the scope intersected with the ability of the role of the token's person.
const (
	ScopeAccountsRead      = "accounts:read"
	ScopeAccountsWrite     = "accounts:write"
	ScopeCategoriesRead    = "categories:read"
	ScopeCategoriesWrite   = "categories:write"
	ScopeTagsRead          = "tags:read"
	ScopeTagsWrite         = "tags:write"
	ScopeTransactionsRead  = "transactions:read"
	ScopeTransactionsWrite = "transactions:write"
	ScopeTransfersWrite    = "transfers:write"
	ScopeBudgetsRead       = "budgets:read"
	ScopeBudgetsWrite      = "budgets:write"
	ScopeSummaryRead       = "summary:read"
)

// Scope is one scope and the role ability it needs.
type Scope struct {
	Name    string
	Ability string
}

var scopes = []Scope{ //nolint:gochecknoglobals // a constant table
	{ScopeAccountsRead, workspaces.AbilityRead},
	{ScopeAccountsWrite, workspaces.AbilityManageData},
	{ScopeCategoriesRead, workspaces.AbilityRead},
	{ScopeCategoriesWrite, workspaces.AbilityManageData},
	{ScopeTagsRead, workspaces.AbilityRead},
	{ScopeTagsWrite, workspaces.AbilityManageData},
	{ScopeTransactionsRead, workspaces.AbilityRead},
	{ScopeTransactionsWrite, workspaces.AbilityRecord},
	{ScopeTransfersWrite, workspaces.AbilityRecord},
	{ScopeBudgetsRead, workspaces.AbilityRead},
	{ScopeBudgetsWrite, workspaces.AbilityManageData},
	{ScopeSummaryRead, workspaces.AbilityRead},
}

// Scopes lists every scope in display order.
func Scopes() []Scope { return slices.Clone(scopes) }

// ValidScope reports whether name is a scope a token can be issued with.
func ValidScope(name string) bool {
	return slices.ContainsFunc(scopes, func(s Scope) bool { return s.Name == name })
}

// AbilityOf is the role ability a scope needs; "" for an unknown scope.
func AbilityOf(scope string) string {
	for _, s := range scopes {
		if s.Name == scope {
			return s.Ability
		}
	}

	return ""
}
