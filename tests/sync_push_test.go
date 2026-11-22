package tests

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// one pushes a single mutation and returns its result.
func (d syncDeviceClient) one(m map[string]any) pushResult {
	d.t.Helper()

	res := d.pushOK(m)
	if len(res) != 1 {
		d.t.Fatalf("pushed one mutation, got %d results", len(res))
	}

	return res[0]
}

func TestSyncPushValidation(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	account, category, _ := seedLedger(t, d)

	txn := func(fields map[string]any) map[string]any {
		base := map[string]any{
			"account_uid": account, "kind": "expense", "amount_cents": 100, "occurred_on": "2026-11-20",
		}
		for k, v := range fields {
			base[k] = v
		}

		return mut("transaction", newULID(), "upsert", 0, base)
	}

	cases := []struct {
		name string
		m    map[string]any
		code string
	}{
		{"unknown entity", mut("widget", newULID(), "upsert", 0, map[string]any{}), "unknown_entity"},
		{"unknown field", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "colour": "red"}), "unknown_field"},
		{"forged version", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "version": 99}), "unknown_field"},
		{"forged workspace", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "workspace_id": 1}), "unknown_field"},
		{"forged created_at", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "created_at": "2020-01-01T00:00:00Z"}), "unknown_field"},
		{"forged deleted_at", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "deleted_at": nil}), "unknown_field"},
		{"forged id", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "id": 7}), "unknown_field"},
		{"bad enum", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "bank"}), "invalid_field"},
		{"missing required", mut("account", newULID(), "upsert", 0, map[string]any{"kind": "cash"}), "missing_field"},
		{"empty name", mut("account", newULID(), "upsert", 0, map[string]any{"name": "  ", "kind": "cash"}), "invalid_field"},
		{"name too long", mut("account", newULID(), "upsert", 0, map[string]any{"name": strings.Repeat("x", 81), "kind": "cash"}), "invalid_field"},
		{"null where none allowed", mut("account", newULID(), "upsert", 0, map[string]any{"name": nil, "kind": "cash"}), "invalid_field"},
		{"fractional cents", txn(map[string]any{"amount_cents": 12.5}), "invalid_amount"},
		{"text amount", txn(map[string]any{"amount_cents": "12.50"}), "invalid_amount"},
		{"zero amount", txn(map[string]any{"amount_cents": 0}), "invalid_amount"},
		{"negative amount", txn(map[string]any{"amount_cents": -5}), "invalid_amount"},
		{"beyond the money cap", txn(map[string]any{"amount_cents": float64(models.MaxCents) + 1}), "invalid_amount"},
		{"opening beyond the cap", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "opening_cents": float64(models.MaxCents) + 1}), "invalid_amount"},
		{"opening below the cap", mut("account", newULID(), "upsert", 0, map[string]any{"name": "A", "kind": "cash", "opening_cents": -float64(models.MaxCents) - 1}), "invalid_amount"},
		{"impossible date", txn(map[string]any{"occurred_on": "2026-02-30"}), "invalid_date"},
		{"loose date", txn(map[string]any{"occurred_on": "2026-2-3"}), "invalid_date"},
		{"timestamp as date", txn(map[string]any{"occurred_on": "2026-11-20T10:00:00Z"}), "invalid_date"},
		{"bad kind", txn(map[string]any{"kind": "transfer"}), "invalid_field"},
		{"bad uid", mut("account", "not-a-ulid", "upsert", 0, map[string]any{"name": "A", "kind": "cash"}), "invalid_uid"},
		{"bad op id", map[string]any{"op_id": "x", "entity": "account", "uid": newULID(), "op": "upsert", "base_seq": 0, "fields": map[string]any{"name": "A", "kind": "cash"}}, "invalid_uid"},
		{"bad op", mut("account", newULID(), "upsert", 0, nil), "missing_field"},
		{"unknown op", mut("account", newULID(), "patch", 0, map[string]any{"name": "A"}), "invalid_op"},
		{"negative base_seq", mut("account", newULID(), "upsert", -1, map[string]any{"name": "A", "kind": "cash"}), "invalid_field"},
		{"delete with fields", mut("account", account, "delete", 0, map[string]any{"name": "A"}), "invalid_field"},
		{"unknown account", txn(map[string]any{"account_uid": newULID()}), "unknown_reference"},
		{"unknown category", txn(map[string]any{"category_uid": newULID()}), "unknown_reference"},
		{"category of another kind", txn(map[string]any{"kind": "income", "category_uid": category}), "invalid_state"},
		{"too many tags", txn(map[string]any{"tag_uids": manyULIDs(11)}), "invalid_field"},
		{"unknown tag", txn(map[string]any{"tag_uids": []string{newULID()}}), "unknown_reference"},
		{"description too long", txn(map[string]any{"description": strings.Repeat("d", 201)}), "invalid_field"},
		{"control character", txn(map[string]any{"description": "a\x00b"}), "invalid_field"},
		{"bad color", mut("category", newULID(), "upsert", 0, map[string]any{"name": "C", "kind": "expense", "color": "#ff0000"}), "invalid_field"},
		{"budget on an income category", mut("budget", newULID(), "upsert", 0, map[string]any{"category_uid": incomeCategory(t, d), "limit_cents": 100}), "invalid_state"},
	}

	// One push: the device is held to 30 pushes a minute.
	muts := make([]map[string]any, len(cases))
	for i, tc := range cases {
		muts[i] = tc.m
	}

	results := d.pushOK(muts...)

	for i, tc := range cases {
		got := results[i]
		if got.Status != "rejected" || got.code() != tc.code {
			t.Errorf("%s: status %q code %q (%+v), want rejected %s", tc.name, got.Status, got.code(), got.Error, tc.code)
		}
	}

	// The money cap itself is allowed, and so is a valid transaction.
	ok := d.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Rich", "kind": "cash", "opening_cents": models.MaxCents}))
	if ok.Status != "applied" {
		t.Errorf("an opening balance of exactly 2^53-1 = %+v", ok)
	}

	if got := d.one(txn(nil)); got.Status != "applied" {
		t.Errorf("a valid transaction = %+v", got)
	}

	// A rejected mutation changes nothing: only valid rows exist.
	if got := d.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Half", "kind": "cash", "colour": 1})); got.Status != "rejected" {
		t.Errorf("got %+v", got)
	}
}

func manyULIDs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = newULID()
	}

	return out
}

func incomeCategory(t *testing.T, d syncDeviceClient) string {
	t.Helper()

	uid := newULID()
	if r := d.one(mut("category", uid, "upsert", 0, map[string]any{"name": "Pay " + uid, "kind": "income"})); r.Status != "applied" {
		t.Fatalf("income category: %+v", r)
	}

	return uid
}

func TestSyncPushBatchLimits(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")

	var muts []map[string]any
	for range 201 {
		muts = append(muts, mut("tag", newULID(), "upsert", 0, map[string]any{"name": "t"}))
	}

	assertProblem(t, d.push(muts...), 413, "batch_too_large")

	// An empty batch is fine.
	d.pushOK()
}

// One rejected mutation does not abort the others of the batch.
func TestSyncPushIsolatesFailures(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	good, bad := newULID(), newULID()

	res := d.pushOK(
		mut("account", good, "upsert", 0, map[string]any{"name": "Good", "kind": "cash"}),
		mut("account", bad, "upsert", 0, map[string]any{"name": "Bad", "kind": "vault"}),
	)

	if res[0].Status != "applied" || res[1].Status != "rejected" || res[0].OpID == res[1].OpID {
		t.Fatalf("results = %+v", res)
	}

	items, _ := d.pullAll(0)
	if len(only(items, good)) != 1 || len(only(items, bad)) != 0 {
		t.Error("the good row must exist and the bad one must not")
	}
}

func TestSyncPushPermissions(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	ana := f.who

	vera := viewerOf(t, f.app, ana)
	viewer := newSyncDevice(t, f.app, rawToken(t, f.app, vera.User, ana.Workspace.ID, "sync"), "viewer phone")

	member := newVerifiedUser(t, f.app, "Max", "max@example.com")
	join(t, f.app, ana, member, models.RoleMember)
	memberDev := newSyncDevice(t, f.app, rawToken(t, f.app, member, ana.Workspace.ID, "sync"), "member phone")

	owner := f.device(t, "owner phone")
	account, category, _ := seedLedger(t, owner)

	newTxn := func() map[string]any {
		return mut("transaction", newULID(), "upsert", 0, map[string]any{
			"account_uid": account, "kind": "expense", "amount_cents": 5, "occurred_on": "2026-11-20",
		})
	}

	// A viewer may pull and may not write anything.
	viewer.pull(0, 10)

	for _, m := range []map[string]any{
		newTxn(),
		mut("account", newULID(), "upsert", 0, map[string]any{"name": "V", "kind": "cash"}),
		mut("category", category, "delete", 1, nil),
	} {
		if got := viewer.one(m); got.Status != "rejected" || got.code() != "forbidden" {
			t.Errorf("a viewer's %s = %+v", m["entity"], got)
		}
	}

	// A member records transactions but does not manage accounts or categories.
	if got := memberDev.one(newTxn()); got.Status != "applied" {
		t.Errorf("a member's transaction = %+v", got)
	}

	for _, m := range []map[string]any{
		mut("account", newULID(), "upsert", 0, map[string]any{"name": "M", "kind": "cash"}),
		mut("category", newULID(), "upsert", 0, map[string]any{"name": "M", "kind": "expense"}),
		mut("budget", newULID(), "upsert", 0, map[string]any{"category_uid": category, "limit_cents": 5}),
		mut("account", account, "delete", 1, nil),
	} {
		if got := memberDev.one(m); got.Status != "rejected" || got.code() != "forbidden" {
			t.Errorf("a member's %s %s = %+v", m["entity"], m["op"], got)
		}
	}

	// Nothing the viewer or the member was refused exists.
	items, _ := owner.pullAll(0)
	if n := len(only(items, category)); n != 1 {
		t.Errorf("the category must still be live once, got %d items", n)
	}

	// A token without the scope gets nothing; a role never exceeds the token.
	noScope := apiClient(t, f.app, rawToken(t, f.app, ana.User, ana.Workspace.ID, "accounts:read")).WithHeader("X-Sync-Protocol", "1")
	assertProblem(t, noScope.Get("/api/v1/sync/info"), 403, "missing_scope")
}

func TestSecurity_PushOtherWorkspaceUID(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, ana)
	makePro(t, app, bob)

	anaDev := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "ana")
	bobDev := newSyncDevice(t, app, rawToken(t, app, bob.User, bob.Workspace.ID, "sync"), "bob")

	bobAccount, bobCategory, bobTxn := seedLedger(t, bobDev)
	bobTag := newULID()
	bobDev.one(mut("tag", bobTag, "upsert", 0, map[string]any{"name": "bobs"}))
	anaAccount, _, _ := seedLedger(t, anaDev)

	rejectedWith := func(name string, m map[string]any, code string) {
		t.Helper()

		if got := anaDev.one(m); got.Status != "rejected" || got.code() != code {
			t.Errorf("%s = status %q code %q, want rejected %s", name, got.Status, got.code(), code)
		}
	}

	// References to another workspace read as references to nothing.
	rejectedWith("account", mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": bobAccount, "kind": "expense", "amount_cents": 1, "occurred_on": "2026-11-20",
	}), "unknown_reference")
	rejectedWith("category", mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": anaAccount, "category_uid": bobCategory, "kind": "expense", "amount_cents": 1, "occurred_on": "2026-11-20",
	}), "unknown_reference")
	rejectedWith("tag", mut("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": anaAccount, "kind": "expense", "amount_cents": 1, "occurred_on": "2026-11-20", "tag_uids": []string{bobTag},
	}), "unknown_reference")
	rejectedWith("parent", mut("category", newULID(), "upsert", 0, map[string]any{
		"name": "Kid", "kind": "expense", "parent_uid": bobCategory,
	}), "unknown_reference")
	rejectedWith("budget", mut("budget", newULID(), "upsert", 0, map[string]any{"category_uid": bobCategory, "limit_cents": 5}), "unknown_reference")

	// Editing, deleting or restoring a row that exists only in another workspace.
	rejectedWith("edit", mut("transaction", bobTxn, "upsert", 3, map[string]any{"description": "mine now"}), "missing_field")
	rejectedWith("delete", mut("account", bobAccount, "delete", 0, nil), "unknown_row")
	rejectedWith("restore", mut("account", bobAccount, "restore", 0, nil), "unknown_row")

	// Another workspace's device id is refused, and the other workspace is untouched.
	stolen := anaDev.client().WithHeader("X-Device-Id", bobDev.Device)
	assertProblem(t, stolen.Get("/api/v1/sync/pull"), 403, "device_revoked")

	items, _ := bobDev.pullAll(0)

	for _, it := range only(items, bobAccount, bobCategory, bobTxn, bobTag) {
		if it.Op != "upsert" || it.Version != 1 {
			t.Errorf("bob's %s changed: %+v", it.Entity, it)
		}
	}

	// Ana's pull never shows Bob's rows.
	anaItems, _ := anaDev.pullAll(0)
	if n := len(only(anaItems, bobAccount, bobCategory, bobTxn, bobTag)); n != 0 {
		t.Errorf("ana's pull holds %d of bob's rows", n)
	}

	// The same uid in two workspaces is two rows: pushing it creates Ana's own.
	if got := anaDev.one(mut("account", bobAccount, "upsert", 0, map[string]any{"name": "Same uid", "kind": "cash"})); got.Status != "applied" {
		t.Errorf("a uid that another workspace uses = %+v, want a row of her own", got)
	}

	bobItems, _ := bobDev.pullAll(0)
	for _, it := range only(bobItems, bobAccount) {
		if it.Row["name"] == "Same uid" {
			t.Error("ana's push renamed bob's account")
		}
	}
}

// The mutations of one row keep their request order, whatever the fields say: a
// category created with a null parent (or with a parent) and renamed in the same
// batch must be created first.
func TestSyncPushKeepsTheOrderOfOneRow(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")

	root, plain, kid := newULID(), newULID(), newULID()

	results := d.pushOK(
		mut("category", root, "upsert", 0, map[string]any{"name": "Root", "kind": "expense"}),
		mut("category", plain, "upsert", 0, map[string]any{"name": "Plain", "kind": "expense", "parent_uid": nil}),
		mut("category", plain, "upsert", 0, map[string]any{"name": "Plain renamed"}),
		mut("category", kid, "upsert", 0, map[string]any{"name": "Kid", "kind": "expense", "parent_uid": root}),
		mut("category", kid, "upsert", 0, map[string]any{"name": "Kid renamed"}),
	)

	for i, want := range []string{"applied", "applied", "merged", "applied", "merged"} {
		if results[i].Status != want {
			t.Errorf("result %d = %s (%s), want %s", i, results[i].Status, results[i].code(), want)
		}
	}

	rows, _ := d.pullAll(0)
	names := map[string]string{}

	for _, r := range rows {
		if r.Op == "upsert" {
			names[r.UID], _ = r.Row["name"].(string)
		}
	}

	if names[plain] != "Plain renamed" || names[kid] != "Kid renamed" {
		t.Errorf("names = %v", names)
	}
}
