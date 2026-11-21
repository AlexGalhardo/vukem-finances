package tests

import (
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"
)

// Billing data is server-only: a uid column would make the sync guard treat the
// table as a synced entity, so none has one, and each is still covered by the
// isolation guard through workspace_id and a leading index.
func TestBillingTablesNotSynced(t *testing.T) {
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

	for _, name := range []string{"pix_payments", "pix_entitlements", "billing_events"} {
		tbl, ok := tableOf(snap, name)
		if !ok {
			t.Errorf("%s: table does not exist", name)
			continue
		}

		if hasColumns(tbl, "uid") {
			t.Errorf("%s: a billing table must not have a uid column", name)
		}

		if !hasColumns(tbl, "workspace_id") || !hasTenantIndex(tbl) {
			t.Errorf("%s: needs workspace_id with a leading index", name)
		}
	}

	sub, _ := tableOf(snap, "workspace_subscriptions")
	if !hasColumns(sub, "access_revoked_at", "revoked_reason", "revoked_ref") || slices.ContainsFunc(sub.Columns, func(c schema.ColumnInfo) bool { return c.Name == "uid" }) {
		t.Error("workspace_subscriptions: revocation columns missing or uid present")
	}
}
