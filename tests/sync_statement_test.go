package tests

import (
	"database/sql"
	"maps"
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

// A restore and the legs of a transfer are held to the same rule, and a leg
// cannot leave its transfer by deleting the other one in the same batch.
func TestSyncStatementOnRestoreAndTransfers(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	fin := softService(t, f.app)
	ctx := wsCtx(t, f.who)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	db := mustDB(t, f.app).SQL()

	var cardUID, plainUID string

	if err := db.QueryRowContext(t.Context(), "SELECT uid FROM accounts WHERE id = ?", card.AccountID).Scan(&cardUID); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRowContext(t.Context(), "SELECT uid FROM accounts WHERE workspace_id = ? AND id <> ? LIMIT 1",
		f.who.Workspace.ID, card.AccountID).Scan(&plainUID); err != nil {
		t.Fatal(err)
	}

	stored := func(uid string) string {
		t.Helper()

		var ref sql.NullString
		if err := db.QueryRowContext(t.Context(), "SELECT statement_ref FROM transactions WHERE uid = ?", uid).Scan(&ref); err != nil {
			t.Fatal(err)
		}

		return ref.String
	}

	const statement = "2026-12-10"

	buy := newULID()
	bought := d.one(mut("transaction", buy, "upsert", 0, map[string]any{
		"account_uid": cardUID, "kind": "expense", "amount_cents": 500, "occurred_on": "2026-11-20",
	}))
	gone := d.one(mut("transaction", buy, "delete", bought.Seq, nil))
	d.one(mut("transaction", buy, "restore", gone.Seq, map[string]any{"statement_ref": "2020-01-10"}))

	if got := stored(buy); got != statement {
		t.Errorf("after a restore that names a statement: %q, want %q", got, statement)
	}

	legs := func(ref string) (out, in string, results []pushResult) {
		out, in = newULID(), newULID()
		group := newULID()
		base := map[string]any{"amount_cents": 100, "occurred_on": "2026-11-20", "transfer_uid": group}

		expense := map[string]any{"account_uid": plainUID, "kind": "expense", "statement_ref": ref}
		income := map[string]any{"account_uid": cardUID, "kind": "income", "statement_ref": ref}
		maps.Copy(expense, base)
		maps.Copy(income, base)

		return out, in, d.pushOK(mut("transaction", out, "upsert", 0, expense), mut("transaction", in, "upsert", 0, income))
	}

	out, in, _ := legs("2020-01-10")
	if stored(out) != "" || stored(in) != "" {
		t.Errorf("a forged payment: expense %q, income %q, want neither on a statement", stored(out), stored(in))
	}

	out, in, res := legs(statement)
	if stored(out) != "" || stored(in) != statement {
		t.Errorf("a real payment: expense %q, income %q, want only the income on %s", stored(out), stored(in), statement)
	}

	batch := d.pushOK(
		mut("transaction", in, "upsert", res[1].Seq, map[string]any{"transfer_uid": nil}),
		mut("transaction", out, "delete", res[0].Seq, nil),
	)
	if batch[0].Status != "rejected" {
		t.Errorf("a leg left its transfer while the other was deleted: %+v", batch[0])
	}
}
