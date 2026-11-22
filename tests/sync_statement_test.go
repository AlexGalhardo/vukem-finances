package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// The statement of a purchase follows from its card and its date, as it does
// on every other path: a device cannot file one on another statement.
func TestSyncStatementIsDerivedByTheServer(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	fin := softService(t, f.app)
	ctx := wsCtx(t, f.who)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	var cardUID, plainUID string

	db := mustDB(t, f.app).SQL()
	if err := db.QueryRowContext(t.Context(), "SELECT uid FROM accounts WHERE id = ?", card.AccountID).Scan(&cardUID); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRowContext(t.Context(), "SELECT uid FROM accounts WHERE workspace_id = ? AND id <> ? LIMIT 1",
		f.who.Workspace.ID, card.AccountID).Scan(&plainUID); err != nil {
		t.Fatal(err)
	}

	uid := newULID()

	got := d.one(mut("transaction", uid, "upsert", 0, map[string]any{
		"account_uid": cardUID, "kind": "expense", "amount_cents": 500, "occurred_on": "2026-11-20",
		"statement_ref": "2020-01-10",
	}))
	if got.Status != "applied" || got.Row["statement_ref"] != "2026-12-10" {
		t.Fatalf("a forged statement = %+v, want the row on 2026-12-10", got)
	}

	moved := d.one(mut("transaction", uid, "upsert", got.Seq, map[string]any{"occurred_on": "2026-11-05"}))
	if moved.Row["statement_ref"] != "2026-11-10" {
		t.Fatalf("after a new date the statement = %v, want 2026-11-10", moved.Row["statement_ref"])
	}

	plain := d.one(mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": plainUID, "kind": "expense", "amount_cents": 500, "occurred_on": "2026-11-20",
		"statement_ref": "2026-12-10",
	}))
	if plain.Status != "applied" || plain.Row["statement_ref"] != nil {
		t.Fatalf("an entry outside a card = %+v, want no statement", plain)
	}
}
