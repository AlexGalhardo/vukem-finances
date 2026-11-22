package tests

import (
	"fmt"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// TestEachExportRowReadsTheRowsOfEachTransaction is QA-503: the CSV reads plain
// columns in chunks, and must yield what the model reader yields, in the same
// order, across chunk edges that fall inside a run of rows of one date, without
// a trashed row or a row of another workspace.
func TestEachExportRowReadsTheRowsOfEachTransaction(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	acc, bobAcc := firstAccount(t, app, ana), firstAccount(t, app, bob)

	tx, err := db.SQL().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 2600 {
		ws, account, deleted := ana.Workspace.ID, acc, any(nil)

		switch {
		case i%13 == 0:
			deleted = "2026-01-01T00:00:00Z"
		case i%7 == 0:
			ws, account = bob.Workspace.ID, bobAcc
		}

		_, err := tx.ExecContext(t.Context(),
			"INSERT INTO transactions (workspace_id, account_id, kind, amount_cents, description, occurred_on, uid, tag_uids, deleted_at, created_at, updated_at) "+
				"VALUES (?, ?, 'expense', ?, ?, ?, ?, '[]', ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')",
			ws, account, 100+i, fmt.Sprintf("shop %d", i), fmt.Sprintf("2026-0%d-1%d", 1+i%3, i%2), fmt.Sprintf("uid%020d", i), deleted)
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	for _, f := range []finance.TxFilter{{}, {From: "2026-02-10", To: "2026-03-11"}} {
		var want, got []string

		for m, err := range fin.EachTransaction(ctx, f) {
			if err != nil {
				t.Fatal(err)
			}

			want = append(want, fmt.Sprint(m.ID, m.UID, m.OccurredOn, m.AmountCents, m.Description))
		}

		for r, err := range fin.EachExportRow(ctx, f) {
			if err != nil {
				t.Fatal(err)
			}

			got = append(got, fmt.Sprint(r.ID, r.UID, r.OccurredOn, r.AmountCents, r.Description))
		}

		if len(want) < 1000 {
			t.Fatalf("filter %+v matched %d rows: the test needs more than one chunk", f, len(want))
		}

		if !slices.Equal(got, want) {
			t.Errorf("filter %+v: %d export rows, %d model rows, or a different order", f, len(got), len(want))
		}
	}
}
