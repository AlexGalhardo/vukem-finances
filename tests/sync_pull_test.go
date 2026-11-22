package tests

import (
	"slices"
	"testing"
)

// seedLedger pushes an account, a category and a transaction from d and
// returns their uids.
func seedLedger(t *testing.T, d syncDeviceClient) (account, category, txn string) {
	t.Helper()

	account, category, txn = newULID(), newULID(), newULID()

	res := d.pushOK(
		mut("account", account, "upsert", 0, map[string]any{"name": "Wallet", "kind": "cash", "opening_cents": 1000}),
		mut("category", category, "upsert", 0, map[string]any{"name": "Food", "kind": "expense", "color": "green"}),
		mut("transaction", txn, "upsert", 0, map[string]any{
			"account_uid": account, "category_uid": category, "kind": "expense", "amount_cents": 250,
			"description": "Lunch", "occurred_on": "2026-11-20",
		}),
	)

	for _, r := range res {
		if r.Status != "applied" {
			t.Fatalf("seed: %+v", r)
		}
	}

	return account, category, txn
}

// only keeps the items of the named rows, in pull order.
func only(items []pullItem, uids ...string) []pullItem {
	var out []pullItem

	for _, it := range items {
		if slices.Contains(uids, it.UID) {
			out = append(out, it)
		}
	}

	return out
}

func TestSyncInfo(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	c := apiClient(t, f.app, f.token).WithHeader("X-Sync-Protocol", "1")

	res := c.Get("/api/v1/sync/info").AssertOK()
	res.AssertJSONPath("protocol.min", float64(1)).AssertJSONPath("protocol.max", float64(1)).
		AssertJSONPath("workspace.currency", "USD").AssertJSONPath("limits.max_mutations", float64(200))

	// Every registered entity is listed, parents first, each field typed.
	res.AssertJSONPath("entities.0.name", "account")
	count, _ := res.JSONPath("entities")
	if n := len(count.([]any)); n < 9 {
		t.Errorf("entities = %d, want every registered entity", n)
	}
}

func TestSyncProtocolVersion(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	c := apiClient(t, f.app, f.token)

	assertProblem(t, c.Get("/api/v1/sync/info"), 400, "bad_request")
	assertProblem(t, c.WithHeader("X-Sync-Protocol", "2").Get("/api/v1/sync/info"), 426, "protocol_unsupported")
	assertProblem(t, c.WithHeader("X-Sync-Protocol", "abc").Get("/api/v1/sync/info"), 426, "protocol_unsupported")
	c.WithHeader("X-Sync-Protocol", "1").Get("/api/v1/sync/info").AssertOK()
}

func TestSyncPull(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone := f.device(t, "phone")
	account, category, txn := seedLedger(t, phone)

	all, cursor := phone.pullAll(0)
	items := only(all, account, category, txn)

	if len(items) != 3 {
		t.Fatalf("pulled %d items, want 3: %+v", len(items), items)
	}

	// Parents first: the account and the category before the transaction.
	if items[0].Entity != "account" || items[1].Entity != "category" || items[2].Entity != "transaction" {
		t.Errorf("order = %s, %s, %s", items[0].Entity, items[1].Entity, items[2].Entity)
	}

	row := items[2].Row
	if row["account_uid"] != account || row["category_uid"] != category || row["amount_cents"] != float64(250) || items[2].UID != txn {
		t.Errorf("transaction row = %+v", row)
	}

	if _, leaked := row["id"]; leaked {
		t.Error("the row leaks the integer id")
	}

	// Nothing new: the cursor holds and the page is empty.
	page := phone.pull(cursor, 10)
	if len(page.Changes) != 0 || page.NextCursor != cursor || page.HasMore {
		t.Errorf("an idle pull = %+v", page)
	}

	// Pages: has_more and the cursor walk the log without losing or repeating a change.
	seen := map[string]bool{}

	for c, more := int64(0), true; more; {
		p := phone.pull(c, 1)
		for _, it := range p.Changes {
			seen[it.UID] = true
		}

		c, more = p.NextCursor, p.HasMore
	}

	for _, uid := range []string{account, category, txn} {
		if !seen[uid] {
			t.Errorf("paged pull never saw %s", uid)
		}
	}

	if len(seen) != len(all) {
		t.Errorf("paged pull saw %d rows, a full pull %d", len(seen), len(all))
	}
}

// A row edited twice shows once, in its latest state.
func TestSyncPullGroupsByRow(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone := f.device(t, "phone")
	account, _, _ := seedLedger(t, phone)

	phone.pushOK(mut("account", account, "upsert", 1, map[string]any{"name": "Purse"}))
	phone.pushOK(mut("account", account, "upsert", 2, map[string]any{"name": "Bag"}))

	items, _ := phone.pullAll(0)

	var n int

	for _, it := range items {
		if it.UID == account {
			n++

			if it.Row["name"] != "Bag" {
				t.Errorf("account row = %+v, want the latest name", it.Row)
			}
		}
	}

	if n != 1 {
		t.Errorf("the account came %d times in one pull, want once", n)
	}
}

// A page never splits the two legs of a transfer: a one-change page that ends
// on the first leg carries the second one too.
func TestSyncPullTransferLegsSamePage(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	from, to := twoAccounts(t, d)
	_, out, in, _ := pushTransfer(d, from, to)

	all, _ := d.pullAll(0)
	legs := only(all, out, in)

	if len(legs) != 2 {
		t.Fatalf("pull holds %d legs, want 2", len(legs))
	}

	first := min(legs[0].Seq, legs[1].Seq)

	page := d.pull(first-1, 1)
	got := only(page.Changes, out, in)

	if len(got) != 2 || !page.HasMore {
		t.Fatalf("a one-change page from the first leg = %+v, want both legs and has_more", page)
	}

	// Next page: the leg the client already has is not sent again.
	if next := d.pull(page.NextCursor, 1); len(only(next.Changes, out, in)) > 1 {
		t.Errorf("the next page repeats both legs: %+v", next.Changes)
	}
}
