package privacy_test

import (
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"
)

func TestSchema(t *testing.T) {
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

	want := map[string][]string{
		"workspaces":      {"state", "purge_at", "deletion_requested_at", "deletion_requested_by", "purge_error"},
		"users":           {"deletion_requested_at", "purge_at"},
		"privacy_audit":   {"event", "subject_ref", "created_at"},
		"payment_records": {"subject_ref", "provider", "provider_ref", "amount_cents", "currency", "paid_at", "retain_until"},
		"purged_tokens":   {"token_id", "created_at"},
	}

	for table, cols := range want {
		var tbl schema.TableInfo

		found := false

		for _, tb := range snap.Tables {
			if tb.Name == table {
				tbl, found = tb, true
			}
		}

		if !found {
			t.Errorf("table %s does not exist", table)
			continue
		}

		for _, c := range cols {
			if !slices.ContainsFunc(tbl.Columns, func(ci schema.ColumnInfo) bool { return ci.Name == c }) {
				t.Errorf("%s has no column %s", table, c)
			}
		}
	}

	t.Run("the audit and payment tables hold no workspace_id", func(t *testing.T) {
		for _, tb := range snap.Tables {
			if tb.Name != "privacy_audit" && tb.Name != "payment_records" && tb.Name != "purged_tokens" {
				continue
			}

			if slices.ContainsFunc(tb.Columns, func(ci schema.ColumnInfo) bool { return ci.Name == "workspace_id" }) {
				t.Errorf("%s must not reference a workspace", tb.Name)
			}
		}
	})
}

func TestSubjectRefIsStableAndOpaque(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	svc := newService(t, app)

	a, b := svc.Ref("workspace", 7), svc.Ref("workspace", 7)
	if a != b || len(a) != 64 {
		t.Fatalf("ref is not stable: %q %q", a, b)
	}

	if a == svc.Ref("user", 7) || a == svc.Ref("workspace", 8) {
		t.Fatal("different subjects share a reference")
	}

	if a == itoa(7) {
		t.Fatal("reference is the id")
	}
}
