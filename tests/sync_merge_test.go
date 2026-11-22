package tests

import (
	"encoding/json"
	"testing"
	"time"
)

// rowOf pulls everything and returns the item of one row.
func rowOf(t *testing.T, d syncDeviceClient, uid string) pullItem {
	t.Helper()

	items, _ := d.pullAll(0)

	for _, it := range items {
		if it.UID == uid {
			return it
		}
	}

	t.Fatalf("pull has no row %s", uid)

	return pullItem{}
}

// Two devices that edit different fields of one row converge to both edits;
// the same field goes to the later arrival, and the loser is told.
func TestSyncPushMerge(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, tablet := f.device(t, "phone"), f.device(t, "tablet")
	_, _, txn := seedLedger(t, phone)

	base := rowOf(t, tablet, txn).Seq

	// The phone edits the description; the tablet, which has not seen that,
	// edits the amount from the same base.
	first := phone.one(mut("transaction", txn, "upsert", base, map[string]any{"description": "Dinner"}))
	if first.Status != "applied" || len(first.Conflicts) != 0 {
		t.Fatalf("first edit = %+v", first)
	}

	second := tablet.one(mut("transaction", txn, "upsert", base, map[string]any{"amount_cents": 900}))
	if second.Status != "merged" || len(second.Conflicts) != 0 {
		t.Fatalf("second edit = %+v, want merged without a conflict", second)
	}

	row := rowOf(t, phone, txn).Row
	if row["description"] != "Dinner" || row["amount_cents"] != float64(900) {
		t.Errorf("row = %+v, want both edits", row)
	}

	if second.Row["version"] != float64(3) || second.Version != 3 {
		t.Errorf("version after two edits = %v / %d, want 3", second.Row["version"], second.Version)
	}
}

func TestSyncConflictsReported(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, tablet := f.device(t, "phone"), f.device(t, "tablet")
	_, _, txn := seedLedger(t, phone)

	base := rowOf(t, tablet, txn).Seq

	phone.one(mut("transaction", txn, "upsert", base, map[string]any{"amount_cents": 700}))

	loser := tablet.one(mut("transaction", txn, "upsert", base, map[string]any{"amount_cents": 800}))
	if loser.Status != "merged" || len(loser.Conflicts) != 1 {
		t.Fatalf("same-field edit = %+v", loser)
	}

	c := loser.Conflicts[0]
	if c.Field != "amount_cents" || c.LostValue != float64(700) || c.AppliedValue != float64(800) || c.Kind != "field" {
		t.Errorf("conflict = %+v, want 700 lost to 800 on amount_cents", c)
	}

	if got := rowOf(t, phone, txn).Row["amount_cents"]; got != float64(800) {
		t.Errorf("amount = %v, want the later arrival (800)", got)
	}

	// The lost value is kept for 30 days.
	var kept int
	db, _ := f.app.DB()

	if err := db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sync_conflicts WHERE workspace_id = ? AND uid = ? AND field = 'amount_cents' AND lost_value = '700'",
		f.who.Workspace.ID, txn).Scan(&kept); err != nil || kept != 1 {
		t.Errorf("sync_conflicts rows = %d (%v), want 1", kept, err)
	}
}

// Delete wins over a concurrent edit, and only a restore brings the row back.
func TestSyncDeleteVersusEdit(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, tablet := f.device(t, "phone"), f.device(t, "tablet")
	_, _, txn := seedLedger(t, phone)
	seen := rowOf(t, tablet, txn).Seq

	if got := phone.one(mut("transaction", txn, "delete", seen, nil)); got.Status != "applied" || got.Row != nil {
		t.Fatalf("delete = %+v", got)
	}

	edit := tablet.one(mut("transaction", txn, "upsert", seen, map[string]any{"description": "late"}))
	if edit.Status != "rejected" || edit.code() != "edited_deleted_row" || len(edit.Conflicts) != 1 || edit.Conflicts[0].Kind != "edited_deleted_row" {
		t.Fatalf("edit of a deleted row = %+v", edit)
	}

	// The tombstone survives a round trip: pull says delete, with no row.
	it := rowOf(t, tablet, txn)
	if it.Op != "delete" || it.Row != nil || it.Version < 2 {
		t.Errorf("tombstone = %+v", it)
	}

	// Deleting again changes nothing.
	if got := phone.one(mut("transaction", txn, "delete", 0, nil)); got.Status != "duplicate" {
		t.Errorf("second delete = %+v", got)
	}

	restored := tablet.one(mut("transaction", txn, "restore", it.Seq, nil))
	if restored.Status != "applied" || restored.Row["description"] != "Lunch" {
		t.Fatalf("restore = %+v", restored)
	}

	if back := rowOf(t, phone, txn); back.Op != "upsert" || back.Row["description"] != "Lunch" {
		t.Errorf("after the restore = %+v", back)
	}

	// Restoring a live row is a no-op.
	if got := tablet.one(mut("transaction", txn, "restore", 0, nil)); got.Status != "duplicate" {
		t.Errorf("restore of a live row = %+v", got)
	}
}

// A restore is allowed 30 days after the delete and refused later; the delete
// itself and the tombstone survive the round trip either way.
func TestSyncRestoreWindow(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone := f.device(t, "phone")
	_, _, txn := seedLedger(t, phone)
	phone.one(mut("transaction", txn, "delete", 0, nil))

	db, _ := f.app.DB()
	age := func(days int) {
		t.Helper()

		at := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		if _, err := db.SQL().ExecContext(t.Context(), "UPDATE transactions SET deleted_at = ? WHERE uid = ?", at, txn); err != nil {
			t.Fatal(err)
		}
	}

	age(31)

	late := phone.one(mut("transaction", txn, "restore", 0, nil))
	if late.Status != "rejected" || late.code() != "restore_window_expired" {
		t.Fatalf("a restore after 31 days = %+v", late)
	}

	if it := rowOf(t, phone, txn); it.Op != "delete" {
		t.Errorf("the row after a refused restore = %+v, want it still deleted", it)
	}

	age(29)

	if ok := phone.one(mut("transaction", txn, "restore", 0, nil)); ok.Status != "applied" && ok.Status != "merged" {
		t.Errorf("a restore after 29 days = %+v", ok)
	}
}

// The same op_id ten times changes the row once and returns the same result.
func TestSyncPushRetryIsIdempotent(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	uid := newULID()

	create := mut("account", uid, "upsert", 0, map[string]any{"name": "Once", "kind": "cash"})
	first := d.one(create)

	if first.Status != "applied" || first.Version != 1 {
		t.Fatalf("first push = %+v", first)
	}

	want, _ := json.Marshal(first)

	for range 9 {
		again := d.one(create)

		got, _ := json.Marshal(again)
		if string(got) != string(want) {
			t.Fatalf("a retry returned %s, want %s", got, want)
		}
	}

	// An edit retried the same way is applied once: the version moves once.
	edit := mut("account", uid, "upsert", first.Seq, map[string]any{"name": "Twice"})
	for range 10 {
		if got := d.one(edit); got.Status != "applied" || got.Version != 2 {
			t.Fatalf("edit retry = %+v", got)
		}
	}

	if it := rowOf(t, d, uid); it.Version != 2 || it.Row["name"] != "Twice" {
		t.Errorf("row = %+v, want version 2", it)
	}

	// A rejection is stored too: the retry gets the same answer.
	bad := mut("account", newULID(), "upsert", 0, map[string]any{"name": "", "kind": "cash"})
	a, b := d.one(bad), d.one(bad)

	if a.code() != "invalid_field" || b.code() != a.code() {
		t.Errorf("rejection retry = %+v / %+v", a, b)
	}

	// The same op_id twice in one batch also runs once.
	both := d.pushOK(edit, edit)
	if both[0].Status != both[1].Status || both[0].Version != 2 {
		t.Errorf("a batch with a repeated op_id = %+v", both)
	}
}

// Creating a uid that exists with the same fields is a retry, not an error and
// not a second row.
func TestSyncCreateOfExistingUID(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	uid := newULID()
	fields := map[string]any{"name": "Same", "kind": "cash", "opening_cents": 5}

	d.one(mut("account", uid, "upsert", 0, fields))

	// A new op_id (the client lost the first answer and rebuilt the mutation).
	if got := d.one(mut("account", uid, "upsert", 0, fields)); got.Status != "duplicate" || got.Version != 1 {
		t.Errorf("re-create with equal fields = %+v", got)
	}
}

// generated_through never moves back and never past today: a stale device
// cannot make a deleted occurrence generate again, and a far-future value
// cannot stop a rule.
func TestSyncWatermarkClamp(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	account, _, _ := seedLedger(t, d)
	rule := newULID()

	create := d.one(mut("recurrence", rule, "upsert", 0, map[string]any{
		"account_uid": account, "kind": "expense", "amount_cents": 100, "frequency": "monthly",
		"start_date": "2026-01-15", "generated_through": "2026-03-15",
	}))
	if create.Status != "applied" {
		t.Fatalf("rule = %+v", create)
	}

	push := func(through string, base int64) pushResult {
		return d.one(mut("recurrence", rule, "upsert", base, map[string]any{"generated_through": through}))
	}

	if got := push("2026-02-15", 0); got.Status != "duplicate" || got.Row["generated_through"] != "2026-03-15" || len(got.Conflicts) != 0 {
		t.Errorf("an earlier watermark = %+v, want it ignored with no conflict", got)
	}

	if got := push("2026-04-15", 0); got.Status == "rejected" || got.Row["generated_through"] != "2026-04-15" || len(got.Conflicts) != 0 {
		t.Errorf("a later watermark = %+v, want it applied with no conflict", got)
	}

	future := time.Now().UTC().AddDate(5, 0, 0).Format("2006-01-02")
	if got := push(future, 0); got.Status != "duplicate" || got.Row["generated_through"] != "2026-04-15" {
		t.Errorf("a watermark in the future = %+v, want it ignored", got)
	}

	if got := push("not-a-date", 0); got.Status != "rejected" || got.code() != "invalid_date" {
		t.Errorf("a malformed watermark = %+v", got)
	}
}
