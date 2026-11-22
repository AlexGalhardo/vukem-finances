package tests

import (
	"sync"
	"testing"
)

func legFields(account, kind string, cents int, on string, transfer string) map[string]any {
	return map[string]any{
		"account_uid": account, "kind": kind, "amount_cents": cents, "occurred_on": on, "transfer_uid": transfer,
	}
}

// pushTransfer sends both legs of a transfer.
func pushTransfer(d syncDeviceClient, from, to string) (transfer, out, in string, res []pushResult) {
	transfer, out, in = newULID(), newULID(), newULID()
	res = d.pushOK(
		mut("transaction", out, "upsert", 0, legFields(from, "expense", 300, "2026-11-20", transfer)),
		mut("transaction", in, "upsert", 0, legFields(to, "income", 300, "2026-11-20", transfer)),
	)

	return transfer, out, in, res
}

func twoAccounts(t *testing.T, d syncDeviceClient) (string, string) {
	t.Helper()

	a, b := newULID(), newULID()
	for _, r := range d.pushOK(
		mut("account", a, "upsert", 0, map[string]any{"name": "From " + a, "kind": "checking"}),
		mut("account", b, "upsert", 0, map[string]any{"name": "To " + b, "kind": "savings"}),
	) {
		if r.Status != "applied" {
			t.Fatalf("account: %+v", r)
		}
	}

	return a, b
}

func TestSyncTransfers(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d, other := f.device(t, "phone"), f.device(t, "tablet")
	from, to := twoAccounts(t, d)

	transfer, out, in, res := pushTransfer(d, from, to)
	if res[0].Status != "applied" || res[1].Status != "applied" {
		t.Fatalf("both legs = %+v", res)
	}

	// Both legs come back together in one page, even when the page is tiny.
	all, _ := other.pullAll(0)
	if len(only(all, out, in)) != 2 {
		t.Fatalf("pull holds %d legs, want 2", len(only(all, out, in)))
	}

	// One leg alone is refused, whatever the order.
	lone := d.one(mut("transaction", newULID(), "upsert", 0, legFields(from, "expense", 5, "2026-11-20", newULID())))
	if lone.Status != "rejected" || lone.code() != "transfer_incomplete" {
		t.Errorf("a lone leg = %+v", lone)
	}

	// Both legs of a broken pair are refused together and nothing is written.
	t1, a1, b1 := newULID(), newULID(), newULID()
	bad := d.pushOK(
		mut("transaction", a1, "upsert", 0, legFields(from, "expense", 5, "2026-11-20", t1)),
		mut("transaction", b1, "upsert", 0, legFields(to, "income", 6, "2026-11-20", t1)),
	)

	if bad[0].code() != "transfer_invalid" || bad[1].code() != "transfer_invalid" {
		t.Errorf("legs with different amounts = %+v", bad)
	}

	// Three legs of one transfer are refused.
	t2 := newULID()
	three := d.pushOK(
		mut("transaction", newULID(), "upsert", 0, legFields(from, "expense", 5, "2026-11-20", t2)),
		mut("transaction", newULID(), "upsert", 0, legFields(to, "income", 5, "2026-11-20", t2)),
		mut("transaction", newULID(), "upsert", 0, legFields(to, "income", 5, "2026-11-20", t2)),
	)

	for _, r := range three {
		if r.Status != "rejected" {
			t.Errorf("three legs: %+v", r)
		}
	}

	after, _ := other.pullAll(0)
	if len(only(after, a1, b1)) != 0 {
		t.Error("a refused pair was written")
	}

	// A leg that changes its transfer_uid away leaves its partner alone: refused.
	left := d.one(mut("transaction", out, "upsert", 0, map[string]any{"transfer_uid": nil}))
	if left.Status != "rejected" || left.code() != "transfer_incomplete" {
		t.Errorf("one leg leaving the transfer = %+v", left)
	}

	// Deleting one leg deletes the other and reports both changes.
	del := d.one(mut("transaction", out, "delete", 0, nil))
	if del.Status != "applied" || len(del.Effects) != 1 || del.Effects[0].UID != in || del.Effects[0].Op != "delete" {
		t.Fatalf("delete of a leg = %+v", del)
	}

	gone, _ := other.pullAll(0)
	for _, it := range only(gone, out, in) {
		if it.Op != "delete" {
			t.Errorf("leg %s after the delete = %+v", it.UID, it)
		}
	}

	// Restoring one leg brings the pair back, within the window.
	back := d.one(mut("transaction", in, "restore", 0, nil))
	if back.Status == "rejected" {
		t.Fatalf("restore of a leg = %+v", back)
	}

	live, _ := other.pullAll(0)
	for _, it := range only(live, out, in) {
		if it.Op != "upsert" || it.Row["transfer_uid"] != transfer {
			t.Errorf("leg %s after the restore = %+v", it.UID, it)
		}
	}
}

// A create that hits the natural key of a recurrence occurrence is the same
// record: the client adopts the existing uid.
func TestSyncRecurrenceDuplicate(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, tablet := f.device(t, "phone"), f.device(t, "tablet")
	account, _, _ := seedLedger(t, phone)
	rule := newULID()

	phone.one(mut("recurrence", rule, "upsert", 0, map[string]any{
		"account_uid": account, "kind": "expense", "amount_cents": 100, "frequency": "monthly", "start_date": "2026-01-15",
	}))

	occurrence := func() map[string]any {
		return mut("transaction", newULID(), "upsert", 0, map[string]any{
			"account_uid": account, "kind": "expense", "amount_cents": 100, "occurred_on": "2026-02-15",
			"recurrence_uid": rule, "occurrence_on": "2026-02-15",
		})
	}

	first := phone.one(occurrence())
	if first.Status != "applied" {
		t.Fatalf("first occurrence = %+v", first)
	}

	second := tablet.one(occurrence())
	if second.Status != "duplicate" || second.ReplacedByUID != first.UID || second.Row["uid"] != first.UID {
		t.Fatalf("second occurrence = %+v, want a duplicate of %s", second, first.UID)
	}

	// An occurrence the person deleted is never created again.
	phone.one(mut("transaction", first.UID, "delete", 0, nil))

	third := tablet.one(occurrence())
	if third.Status != "duplicate" || third.ReplacedByUID != first.UID || third.Row != nil {
		t.Errorf("occurrence of a deleted one = %+v", third)
	}

	items, _ := phone.pullAll(0)

	n := 0

	for _, it := range items {
		if it.Entity == "transaction" && it.Row != nil && it.Row["recurrence_uid"] == rule {
			n++
		}
	}

	if n != 0 {
		t.Errorf("%d live occurrences, want none after the delete", n)
	}
}

// Deleting an account soft-deletes its transactions, each logged, and they
// come back through pull.
func TestSyncAccountCascade(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, tablet := f.device(t, "phone"), f.device(t, "tablet")
	account, _, txn := seedLedger(t, phone)
	_, cursor := tablet.pullAll(0)

	del := phone.one(mut("account", account, "delete", 0, nil))
	if del.Status != "applied" {
		t.Fatalf("delete = %+v", del)
	}

	found := false

	for _, e := range del.Effects {
		found = found || (e.UID == txn && e.Op == "delete")
	}

	if !found {
		t.Errorf("effects = %+v, want the transaction deleted", del.Effects)
	}

	items, _ := tablet.pullAll(cursor)
	if got := only(items, txn); len(got) != 1 || got[0].Op != "delete" {
		t.Errorf("the transaction after its account was deleted = %+v", got)
	}

	// A transaction cannot be added to the deleted account.
	late := tablet.one(mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": account, "kind": "expense", "amount_cents": 1, "occurred_on": "2026-11-20",
	}))
	if late.code() != "parent_deleted" {
		t.Errorf("a transaction on a deleted account = %+v", late)
	}

	// Restoring the account brings its transactions back (they were deleted with it).
	phone.one(mut("account", account, "restore", 0, nil))

	back, _ := tablet.pullAll(cursor)
	if got := only(back, txn); len(got) != 1 || got[0].Op != "upsert" {
		t.Errorf("the transaction after the account came back = %+v", got)
	}
}

// The Free plan stops at its monthly quota, atomically: concurrent pushes
// cannot both pass the count, and a refusal does not block the other rows.
func TestSyncQuotaAtomic(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com") // Free: 100 transactions a month
	token := rawToken(t, app, ana.User, ana.Workspace.ID, "sync")
	a, b := newSyncDevice(t, app, token, "a"), newSyncDevice(t, app, token, "b")

	// The workspace starts with its one account.
	var account string

	items, _ := a.pullAll(0)
	for _, it := range items {
		if it.Entity == "account" {
			account = it.UID
		}
	}

	batch := func(n int) []map[string]any {
		out := make([]map[string]any, n)
		for i := range out {
			out[i] = mut("transaction", newULID(), "upsert", 0, map[string]any{
				"account_uid": account, "kind": "expense", "amount_cents": 1 + i, "occurred_on": "2026-11-10",
			})
		}

		return out
	}

	// Two devices race 60 each into a quota of 100.
	var (
		wg      sync.WaitGroup
		results [2][]pushResult
	)

	for i, d := range []syncDeviceClient{a, b} {
		wg.Add(1)

		go func() {
			defer wg.Done()

			results[i] = d.pushOK(batch(60)...)
		}()
	}

	wg.Wait()

	applied, refused := 0, 0

	for _, rs := range results {
		for _, r := range rs {
			switch {
			case r.Status == "applied":
				applied++
			case r.code() == "quota_exceeded":
				refused++
			default:
				t.Fatalf("unexpected result %+v (%+v)", r, r.Error)
			}
		}
	}

	if applied != 100 || refused != 20 {
		t.Errorf("applied %d and refused %d, want exactly 100 and 20", applied, refused)
	}

	// Another month has its own quota, so the refusal blocked nothing else.
	other := a.one(mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": account, "kind": "expense", "amount_cents": 1, "occurred_on": "2026-10-10",
	}))
	if other.Status != "applied" {
		t.Errorf("a transaction in another month = %+v", other)
	}

	// Free also keeps one live account: a second account is refused.
	if got := a.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Second", "kind": "cash"})); got.code() != "quota_exceeded" {
		t.Errorf("a second account on Free = %+v", got)
	}

	// An archived one does not count.
	if got := a.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Old", "kind": "cash", "archived_at": "2026-01-01T00:00:00Z"})); got.Status != "applied" {
		t.Errorf("an archived account on Free = %+v", got)
	}
}
