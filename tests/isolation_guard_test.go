package tests

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"
	"github.com/AlexGalhardo/vukem/tenancy"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// workspaceTables hold a workspace_id but are not read through the tenant
// scope: memberships and subscriptions are keyed by the workspace on purpose
// (the tenant middleware and the Stripe webhook run before any tenant exists).
var workspaceTables = []string{"memberships", "workspace_subscriptions"}

// sharedTables carry no workspace_id by design: the tenant itself, people and
// their credentials, framework tables and address-level data.
var sharedTables = []string{
	"workspaces", "users", "email_suppressions", "admin_audit_logs", "social_accounts",
	"remember_tokens", "access_tokens", "password_resets",
	// Framework tables (queue, cache, sessions, locks, migrations).
	"migrations", "jobs", "failed_jobs", "cache", "cache_locks", "sessions",
}

// guardProblems checks a schema against the isolation rules and returns one
// line per violation, each naming the table.
func guardProblems(snap schema.Snapshot) []string {
	var problems []string

	scoped := map[string]bool{}
	for _, m := range models.TenantModels() {
		scoped[m.Table] = true
	}

	for _, tbl := range snap.Tables {
		has := slices.ContainsFunc(tbl.Columns, func(c schema.ColumnInfo) bool { return c.Name == "workspace_id" })

		switch {
		case !has && (scoped[tbl.Name] || slices.Contains(workspaceTables, tbl.Name)):
			problems = append(problems, tbl.Name+": the table is tenant data but has no workspace_id column")
		case !has && slices.Contains(sharedTables, tbl.Name):
		case !has:
			problems = append(problems, tbl.Name+": no workspace_id column and not on the shared-tables allow-list")
		case !scoped[tbl.Name] && !slices.Contains(workspaceTables, tbl.Name):
			problems = append(problems, tbl.Name+": has workspace_id but its model is not in models.TenantModels")
		case !hasTenantIndex(tbl):
			problems = append(problems, tbl.Name+": no index starts with workspace_id and no unique index includes it")
		}
	}

	return problems
}

func hasTenantIndex(tbl schema.TableInfo) bool {
	return slices.ContainsFunc(tbl.Indexes, func(ix schema.IndexInfo) bool {
		return (len(ix.Columns) > 0 && ix.Columns[0] == "workspace_id") ||
			(ix.Unique && slices.Contains(ix.Columns, "workspace_id"))
	})
}

func TestIsolationGuard(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range guardProblems(snap) {
		t.Error(p)
	}

	t.Run("models stamp the tenant", func(t *testing.T) {
		t.Parallel()

		for _, m := range models.TenantModels() {
			hook, ok := m.New().(interface{ BeforeCreate(context.Context) error })
			if !ok {
				t.Errorf("%s: the model has no BeforeCreate hook", m.Table)
				continue
			}

			if err := hook.BeforeCreate(t.Context()); !errors.Is(err, tenancy.ErrNoTenant) {
				t.Errorf("%s: BeforeCreate without a tenant = %v, want tenancy.ErrNoTenant", m.Table, err)
			}

			if err := hook.BeforeCreate(tenancy.With(t.Context(), "42")); err != nil {
				t.Errorf("%s: BeforeCreate with a tenant: %v", m.Table, err)
			}
		}
	})

	t.Run("every model with a WorkspaceID is registered", func(t *testing.T) {
		t.Parallel()

		registered := map[string]bool{}
		for _, m := range models.TenantModels() {
			registered[fmt.Sprintf("%T", m.New())] = true
		}

		files, err := filepath.Glob("../app/models/*.go")
		if err != nil {
			t.Fatal(err)
		}

		for _, name := range structsWithWorkspaceID(t, files) {
			if registered["*models."+name] || name == "Membership" || name == "WorkspaceSubscription" {
				continue
			}

			t.Errorf("model %s has a WorkspaceID field but is not in models.TenantModels", name)
		}
	})

	t.Run("a table without workspace_id is named", func(t *testing.T) {
		t.Parallel()

		scratch := schema.Snapshot{Tables: []schema.TableInfo{
			{Name: "scratch_notes", Columns: []schema.ColumnInfo{{Name: "id"}, {Name: "body"}}},
			{
				Name:    "scratch_unindexed",
				Columns: []schema.ColumnInfo{{Name: "id"}, {Name: "workspace_id"}},
			},
		}}

		got := strings.Join(guardProblems(scratch), "\n")
		for _, want := range []string{"scratch_notes", "scratch_unindexed"} {
			if !strings.Contains(got, want) {
				t.Errorf("the guard did not name %s: %q", want, got)
			}
		}
	})
}

func structsWithWorkspaceID(t *testing.T, files []string) []string {
	t.Helper()

	var out []string

	for _, file := range files {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			st, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			for _, fld := range st.Fields.List {
				for _, name := range fld.Names {
					if name.Name == "WorkspaceID" {
						out = append(out, spec.Name.Name)
					}
				}
			}

			return true
		})
	}

	return out
}
