package operations

import (
	"slices"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// The scopes a token can hold. A scope is a ceiling: what a request may do is
// the scope intersected with the ability of the role of the person behind it.
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
	// ScopeSync is the sync protocol (docs/specs/sync-protocol.md). It needs
	// only the read ability: what a push may change is checked per row, against
	// the role of the token's person.
	ScopeSync = "sync"
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
	{ScopeSync, workspaces.AbilityRead},
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
