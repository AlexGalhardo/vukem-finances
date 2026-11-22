package privacy_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
)

// makeDue puts the workspace in a grace period that ended an hour ago.
func (w *world) makeDue(t *testing.T) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	past := time.Now().UTC().Add(-time.Hour)
	if _, err := db.ExecContext(t.Context(),
		"UPDATE workspaces SET state = 'pending_deletion', purge_at = ?, deletion_requested_at = ?, deletion_requested_by = ? WHERE id = ?",
		past, past.Add(-privacy.GracePeriod), w.owner.ID, w.space.ID); err != nil {
		t.Fatal(err)
	}
}

// fullWorld is a world with a row in every table of the schema that holds a
// workspace_id, plus the rows of the tables the purge reaches by other means.
func fullWorld(t *testing.T, app *vukemtest.App, email string) *world {
	t.Helper()

	w := newWorld(t, app, email)

	if left := w.seedEveryTable(t, w.owner.ID); len(left) > 0 {
		t.Fatalf("these tables with a workspace_id could not be seeded, so the purge test cannot vouch for them: %v", left)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var grant string
	if err := db.QueryRowContext(t.Context(), "SELECT grant_id FROM mcp_grants WHERE workspace_id = ?", w.space.ID).Scan(&grant); err != nil {
		t.Fatal(err)
	}

	w.insertInto(t, "oauth_grants", w.owner.ID, map[string]any{"id": grant, "subject": fmt.Sprintf("%d:%d", w.owner.ID, w.space.ID)})
	w.insertInto(t, "oauth_codes", w.owner.ID, map[string]any{"grant_id": grant})
	w.insertInto(t, "oauth_refresh_tokens", w.owner.ID, map[string]any{"grant_id": grant})
	w.insertInto(t, "telegram_links", w.owner.ID, nil)
	w.insertInto(t, "telegram_link_codes", w.owner.ID, nil)
	w.insertInto(t, "telegram_pending", w.owner.ID, nil)
	w.insertInto(t, "remember_tokens", w.owner.ID, nil)

	if _, err := db.ExecContext(t.Context(), "UPDATE telegram_links SET current_workspace_id = ? WHERE user_id = ?", w.space.ID, w.owner.ID); err != nil {
		t.Fatal(err)
	}

	// A paid PIX charge: the one thing the law keeps a trace of.
	w.insertInto(t, "pix_payments", w.owner.ID, map[string]any{
		"status": models.PixPaid, "amount_cents": 900, "currency": "BRL", "provider_ref": "pix_" + itoa(w.space.ID), "paid_at": time.Now().UTC(),
	})

	return w
}

// rowCounts counts, per table, the rows that belong to the workspace of w: every
// table with a workspace_id, then the tables that reach it another way.
func (w *world) rowCounts(t *testing.T) map[string]int {
	t.Helper()

	out := map[string]int{}

	for _, tbl := range w.workspaceTablesOf(t) {
		out[tbl.Name] = w.count(t, "SELECT COUNT(*) FROM "+tbl.Name+" WHERE workspace_id = ?", w.space.ID)
	}

	ws := itoa(w.space.ID)
	grants := "(SELECT grant_id FROM mcp_grants WHERE workspace_id = " + ws + ")"

	out["access_tokens(resource)"] = w.count(t, "SELECT COUNT(*) FROM access_tokens WHERE resource = ?", ws)
	out["oauth_grants"] = w.count(t, "SELECT COUNT(*) FROM oauth_grants WHERE id IN "+grants)
	out["oauth_codes"] = w.count(t, "SELECT COUNT(*) FROM oauth_codes WHERE grant_id IN "+grants)
	out["oauth_refresh_tokens"] = w.count(t, "SELECT COUNT(*) FROM oauth_refresh_tokens WHERE grant_id IN "+grants)
	out["telegram_links(current)"] = w.count(t, "SELECT COUNT(*) FROM telegram_links WHERE current_workspace_id = ?", w.space.ID)
	out["telegram_link_codes"] = w.count(t, "SELECT COUNT(*) FROM telegram_link_codes WHERE for_workspace_id = ?", w.space.ID)
	out["telegram_pending"] = w.count(t, "SELECT COUNT(*) FROM telegram_pending WHERE for_workspace_id = ?", w.space.ID)
	out["workspaces"] = w.count(t, "SELECT COUNT(*) FROM workspaces WHERE id = ?", w.space.ID)

	return out
}

func TestPurgeClassifiesEveryTable(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")

	var inSchema []string
	for _, tbl := range w.workspaceTablesOf(t) {
		inSchema = append(inSchema, tbl.Name)
	}

	slices.Sort(inSchema)

	listed := privacy.PurgedTables()
	slices.Sort(listed)

	if !slices.Equal(inSchema, listed) {
		t.Fatalf("tables with a workspace_id and tables the purge removes differ:\n schema: %v\n purge:  %v", inSchema, listed)
	}

	// The check also runs at the start of every purge, and stops it.
	svc := w.svc
	if _, err := svc.Purge(t.Context()); err != nil {
		t.Fatalf("a pass over a classified schema failed: %v", err)
	}
}

func TestPurge(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	a := fullWorld(t, app, "ana@example.com")
	b := fullWorld(t, app, "bob@example.com")

	beforeA, beforeB := a.rowCounts(t), b.rowCounts(t)
	for table, n := range beforeA {
		if n == 0 {
			t.Fatalf("the seed left %s empty for the workspace to purge: the test would prove nothing for it", table)
		}
	}

	// Not due yet: a workspace in its grace period is untouched.
	if _, err := a.svc.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}

	if !maps.Equal(a.rowCounts(t), beforeA) {
		t.Fatal("the purge removed rows before the grace period ended")
	}

	a.makeDue(t)

	rep, err := a.svc.Purge(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if rep.Workspaces != 1 || rep.Blocked != 0 {
		t.Fatalf("report = %+v", rep)
	}

	afterA := a.rowCounts(t)
	for table, n := range afterA {
		if n != 0 {
			t.Errorf("%s still has %d rows of the purged workspace", table, n)
		}
	}

	if got := b.rowCounts(t); !maps.Equal(got, beforeB) {
		t.Errorf("another workspace changed:\n before %v\n after  %v", beforeB, got)
	}

	// The person is not touched by a workspace purge.
	if a.user(t, a.owner.ID).ID == 0 {
		t.Fatal("the owner is gone")
	}

	t.Run("what the law keeps", func(t *testing.T) {
		var amount int64
		var provider, subject string

		err := a.queryRow(t, "SELECT amount_cents, provider, subject_ref FROM payment_records WHERE provider_ref = ?",
			"pix_"+itoa(a.space.ID)).Scan(&amount, &provider, &subject)
		if err != nil {
			t.Fatalf("no payment record kept: %v", err)
		}

		if amount != 900 || provider != "abacatepay" || subject != a.svc.Ref("workspace", a.space.ID) {
			t.Errorf("record = %d %s %s", amount, provider, subject)
		}

		if subject == itoa(a.space.ID) {
			t.Error("the subject reference is the id")
		}

		if n := a.count(t, "SELECT COUNT(*) FROM payment_records WHERE subject_ref = ?", b.svc.Ref("workspace", b.space.ID)); n != 0 {
			t.Errorf("a record of the other workspace exists: %d", n)
		}

		if n := a.count(t, "SELECT COUNT(*) FROM purged_tokens"); n != 1 {
			t.Errorf("%d token tombstones, want 1 (the token of the purged workspace)", n)
		}

		events, err := a.svc.Audit(t.Context(), a.svc.Ref("workspace", a.space.ID))
		if err != nil || len(events) == 0 || events[len(events)-1].Event != "workspace.purged" {
			t.Errorf("audit = %+v, %v", events, err)
		}
	})

	t.Run("running it again changes nothing", func(t *testing.T) {
		records := a.count(t, "SELECT COUNT(*) FROM payment_records")
		audit := a.count(t, "SELECT COUNT(*) FROM privacy_audit")

		rep, err := a.svc.Purge(t.Context())
		if err != nil {
			t.Fatalf("the second pass failed: %v", err)
		}

		if rep.Workspaces != 0 || rep.Blocked != 0 {
			t.Errorf("report = %+v", rep)
		}

		if a.count(t, "SELECT COUNT(*) FROM payment_records") != records || a.count(t, "SELECT COUNT(*) FROM privacy_audit") != audit {
			t.Error("the second pass wrote records again")
		}

		if err := a.svc.PurgeWorkspace(t.Context(), a.space.ID); err != nil {
			t.Fatalf("purging an already purged workspace: %v", err)
		}

		if !maps.Equal(b.rowCounts(t), beforeB) {
			t.Error("another workspace changed")
		}
	})
}

func TestPurgeRefusesAWorkspaceThatIsNotDue(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := fullWorld(t, app, "ana@example.com")
	before := w.rowCounts(t)

	if err := w.svc.PurgeWorkspace(t.Context(), w.space.ID); err == nil {
		t.Fatal("an active workspace was purged")
	}

	if err := w.svc.PurgeWorkspace(t.Context(), 0); err == nil {
		t.Fatal("a purge with no workspace id ran")
	}

	if !maps.Equal(w.rowCounts(t), before) {
		t.Fatal("a refused purge removed rows")
	}
}

func TestPurgeWaitsForTheSubscriptionAndGoesOnWithTheOthers(t *testing.T) {
	t.Parallel()

	f := newFakeStripe(t)
	f.setFail(true)

	a := billingWorld(t, f, "ana@example.com")
	b := fullWorld(t, a.app, "bob@example.com")
	b.svc.Billing = a.svc.Billing

	// Bob has nothing live at Stripe: the generic seed row stands for a canceled one.
	b.exec(t, "UPDATE workspace_subscriptions SET status = 'canceled' WHERE workspace_id = ?", b.space.ID)

	a.makeDue(t)
	b.makeDue(t)

	rep, err := a.svc.Purge(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	// Bob's workspace has no subscription: it is purged; Ana's waits.
	if rep.Workspaces != 1 || rep.Blocked != 1 {
		t.Fatalf("report = %+v", rep)
	}

	if a.count(t, "SELECT COUNT(*) FROM workspaces WHERE id = ?", a.space.ID) != 1 || a.count(t, "SELECT COUNT(*) FROM accounts WHERE workspace_id = ?", a.space.ID) == 0 {
		t.Fatal("a workspace whose subscription could not be cancelled was purged")
	}

	if a.reload(t).PurgeError != privacy.PurgeBlockedBilling {
		t.Errorf("purge_error = %q", a.reload(t).PurgeError)
	}

	f.setFail(false)

	rep, err = a.svc.Purge(t.Context())
	if err != nil || rep.Workspaces != 1 || rep.Blocked != 0 {
		t.Fatalf("after Stripe recovered: %+v, %v", rep, err)
	}

	if a.count(t, "SELECT COUNT(*) FROM workspaces WHERE id = ?", a.space.ID) != 0 {
		t.Fatal("the workspace is still there")
	}

	if n := a.count(t, "SELECT COUNT(*) FROM payment_records WHERE provider = 'stripe' AND provider_ref = ?", "sub_"+itoa(a.space.ID)); n != 1 {
		t.Errorf("%d subscription records kept, want 1", n)
	}
}

func TestPurgeRemovesRetainedRecordsAfterTheirRetention(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	for _, row := range []struct {
		ref   string
		until time.Time
	}{{"old", now.Add(-time.Hour)}, {"kept", now.AddDate(1, 0, 0)}} {
		if _, err := db.ExecContext(t.Context(), `INSERT INTO payment_records
(subject_ref, provider, provider_ref, amount_cents, currency, paid_at, retain_until, created_at, updated_at)
VALUES ('x', 'abacatepay', ?, 100, 'BRL', ?, ?, ?, ?)`, row.ref, now, row.until, now, now); err != nil {
			t.Fatal(err)
		}
	}

	rep, err := w.svc.Purge(t.Context())
	if err != nil || rep.Records != 1 {
		t.Fatalf("report = %+v, %v", rep, err)
	}

	if w.count(t, "SELECT COUNT(*) FROM payment_records WHERE provider_ref = 'old'") != 0 || w.count(t, "SELECT COUNT(*) FROM payment_records WHERE provider_ref = 'kept'") != 1 {
		t.Fatal("the wrong records were removed")
	}
}

func (w *world) queryRow(t *testing.T, query string, args ...any) interface{ Scan(...any) error } {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	return db.QueryRowContext(t.Context(), query, args...)
}

func (w *world) exec(t *testing.T, query string, args ...any) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}
