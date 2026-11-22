package mcp_test

import (
	"slices"
	"strings"
	"testing"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// listSize is the serialized size of the tool list a client would put in its
// context.
func (f *fixture) listSize(t *testing.T) int {
	t.Helper()

	rep := f.client.Call("tools/list", nil)
	if rep.Error != nil {
		t.Fatal(rep.Error)
	}

	return len(rep.Result)
}

func TestToolsets(t *testing.T) {
	t.Parallel()

	// The budgets are what a client may have to hold in context for each
	// toolset; a tool added to a group has to fit, or the budget is raised on
	// purpose.
	cases := []struct {
		name        string
		cfg         config.MCP
		budget      int
		has, hasNot []string
	}{
		{"read", readOnly(), 43_000,
			[]string{"accounts_list", "transactions_list", "reports_cashflow", "summary_show"},
			[]string{"transactions_create", "accounts_create", "budgets_set", "transactions_delete", "export_transactions"}},
		{"transactions", config.MCP{Toolsets: []string{"transactions"}}, 12_000,
			[]string{"transactions_list", "transactions_create", "transactions_update", "transfers_create"},
			[]string{"budgets_set", "accounts_create", "transactions_delete", "transfers_delete"}},
		{"transactions with destructive", config.MCP{Toolsets: []string{"transactions"}, AllowDestructive: true}, 13_500,
			[]string{"transactions_create", "transactions_delete", "transfers_delete"},
			[]string{"budgets_set", "budgets_delete", "accounts_archive"}},
		{"planning", config.MCP{Toolsets: []string{"planning"}}, 46_000,
			[]string{"budgets_set", "goals_create", "recurrences_create", "summary_show", "cards_create"},
			[]string{"transactions_create", "accounts_create", "goals_delete", "budgets_delete"}},
		{"read and transactions", config.MCP{Toolsets: []string{"read", "transactions"}}, 50_000,
			[]string{"accounts_list", "transactions_create"},
			[]string{"accounts_create", "budgets_set"}},
		{"all", config.MCP{Toolsets: []string{"all"}}, 85_000,
			[]string{"accounts_create", "budgets_set", "transactions_create", "goals_create"},
			[]string{"transactions_delete", "goals_delete", "tags_delete"}},
		{"all with destructive", everything(true), 96_000,
			[]string{"transactions_delete", "goals_delete", "tags_delete", "accounts_archive"},
			[]string{"export_transactions", "export_workspace"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t, tc.cfg, nil)
			names := f.toolNames(t)

			for _, n := range tc.has {
				if !slices.Contains(names, n) {
					t.Errorf("missing %s in %v", n, names)
				}
			}

			for _, n := range tc.hasNot {
				if slices.Contains(names, n) {
					t.Errorf("%s is offered", n)
				}
			}

			if size := f.listSize(t); size > tc.budget {
				t.Errorf("the %s tool list is %d bytes, over its budget of %d", tc.name, size, tc.budget)
			} else {
				t.Logf("%s: %d tools, %d bytes (budget %d)", tc.name, len(names), size, tc.budget)
			}
		})
	}

	t.Run("the default is read only", func(t *testing.T) {
		t.Parallel()

		def, err := config.LoadMCP(mapEnv{})
		if err != nil {
			t.Fatal(err)
		}

		f := newFixture(t, def, nil)

		for _, n := range f.toolNames(t) {
			m, ok := f.cat.Named(strings.ReplaceAll(n, "_", "."))
			if ok && m.Kind != operations.KindRead {
				t.Errorf("%s is %s and offered by default", n, m.Kind)
			}
		}

		for _, s := range appmcp.Scopes(f.cat, def) {
			if strings.HasSuffix(s, ":write") {
				t.Errorf("the default actor holds %s", s)
			}
		}
	})
}

type mapEnv map[string]string

func (m mapEnv) Lookup(k string) (string, bool) { v, ok := m[k]; return v, ok }
