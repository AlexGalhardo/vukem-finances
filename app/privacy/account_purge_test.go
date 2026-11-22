package privacy_test

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// makeAccountDue ends the grace period of the account and of the workspaces that
// entered it together.
func (w *world) makeAccountDue(t *testing.T, userID int64) {
	t.Helper()

	past := time.Now().UTC().Add(-time.Hour)
	w.exec(t, "UPDATE users SET purge_at = ? WHERE id = ?", past, userID)
	w.exec(t, "UPDATE workspaces SET purge_at = ? WHERE deletion_requested_by = ? AND state = 'pending_deletion'", past, userID)
}

// userTables are the tables that hold a user_id and are private to that person.
var userTables = []string{
	"assistant_messages", "assistant_conversations", "notification_preferences", "notifications", "sync_devices",
	"pairing_codes", "mcp_grants", "telegram_links", "telegram_link_codes", "telegram_pending", "remember_tokens",
	"social_accounts", "memberships",
}

func TestPurgeAccount(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	// Bob owns a shared workspace with a row in every table; Ana is a member of it
	// and owns a workspace of her own that only she uses.
	shared := fullWorld(t, app, "bob@example.com")
	ana := fullWorld(t, app, "ana@example.com")
	ana.withAccess(t)
	shared.joinAs(t, ana.owner.ID, models.RoleMember)

	// What is Ana's alone in the shared workspace, and what she wrote there.
	for _, table := range []string{
		"assistant_conversations", "assistant_messages", "notification_preferences", "notifications", "sync_devices",
		"pairing_codes", "mcp_grants",
	} {
		shared.insertInto(t, table, ana.owner.ID, nil)
	}

	shared.insertInto(t, "telegram_links", ana.owner.ID, nil)
	shared.exec(t, "UPDATE audit_events SET actor_user_id = ? WHERE workspace_id = ?", ana.owner.ID, shared.space.ID)
	shared.exec(t, "UPDATE ai_credentials SET created_by = ? WHERE workspace_id = ?", ana.owner.ID, shared.space.ID)
	shared.exec(t, "UPDATE pix_payments SET created_by = ? WHERE workspace_id = ?", ana.owner.ID, shared.space.ID)
	shared.exec(t, "INSERT INTO social_accounts (user_id, provider, provider_id, created_at, updated_at) VALUES (?, 'google', 'g-1', ?, ?)",
		ana.owner.ID, time.Now().UTC(), time.Now().UTC())

	token := shared.count(t, "SELECT COUNT(*) FROM access_tokens WHERE user_id = ?", itoa(shared.owner.ID))
	bobsBefore := shared.rowCounts(t)

	for _, table := range []string{"assistant_conversations", "notifications", "sync_devices", "mcp_grants"} {
		if shared.count(t, "SELECT COUNT(*) FROM "+table+" WHERE user_id = ? AND workspace_id = ?", ana.owner.ID, shared.space.ID) == 0 {
			t.Fatalf("the test did not give Ana a row of %s", table)
		}
	}

	if err := ana.svc.RequestAccountDeletion(t.Context(), ana.owner); err != nil {
		t.Fatal(err)
	}

	// Not due: nothing is removed yet.
	if rep, err := ana.svc.Purge(t.Context()); err != nil || rep.Accounts != 0 || rep.Workspaces != 0 {
		t.Fatalf("a pass before the end of the grace period: %+v, %v", rep, err)
	}

	ana.makeAccountDue(t, ana.owner.ID)

	rep, err := ana.svc.Purge(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if rep.Accounts != 1 || rep.Workspaces != 1 || rep.Blocked != 0 {
		t.Fatalf("report = %+v", rep)
	}

	if ana.count(t, "SELECT COUNT(*) FROM users WHERE id = ?", ana.owner.ID) != 0 {
		t.Fatal("the user is still there")
	}

	for _, table := range userTables {
		if n := ana.count(t, "SELECT COUNT(*) FROM "+table+" WHERE user_id = ?", ana.owner.ID); n != 0 {
			t.Errorf("%s still has %d rows of the purged account", table, n)
		}
	}

	if n := ana.count(t, "SELECT COUNT(*) FROM access_tokens WHERE user_id = ?", itoa(ana.owner.ID)); n != 0 {
		t.Errorf("%d API tokens of the purged account remain", n)
	}

	if n := ana.count(t, "SELECT COUNT(*) FROM oauth_grants WHERE id NOT IN (SELECT grant_id FROM mcp_grants) AND id IN (SELECT grant_id FROM mcp_grants)"); n != 0 {
		t.Errorf("%d OAuth grants are orphaned", n)
	}

	if ana.count(t, "SELECT COUNT(*) FROM workspaces WHERE id = ?", ana.space.ID) != 0 {
		t.Error("the workspace only Ana used is still there")
	}

	// The shared workspace stays, with its financial rows, and the author is forgotten.
	for _, table := range []string{"accounts", "transactions", "categories", "audit_events", "ai_credentials", "pix_payments"} {
		if shared.count(t, "SELECT COUNT(*) FROM "+table+" WHERE workspace_id = ?", shared.space.ID) == 0 {
			t.Errorf("the shared workspace lost its %s", table)
		}
	}

	for _, q := range []string{
		"SELECT COUNT(*) FROM audit_events WHERE actor_user_id = ?",
		"SELECT COUNT(*) FROM ai_credentials WHERE created_by = ?",
		"SELECT COUNT(*) FROM pix_payments WHERE created_by = ?",
	} {
		if n := shared.count(t, q, ana.owner.ID); n != 0 {
			t.Errorf("%q: %d rows still name the purged account", q, n)
		}
	}

	// Bob is untouched: his rows, his token, his place.
	bobsAfter := shared.rowCounts(t)
	for _, table := range []string{"accounts", "transactions", "categories", "audit_events", "ai_credentials", "pix_payments", "invites", "billing_events"} {
		if bobsAfter[table] != bobsBefore[table] {
			t.Errorf("%s of the shared workspace: %d rows, was %d", table, bobsAfter[table], bobsBefore[table])
		}
	}

	if shared.count(t, "SELECT COUNT(*) FROM access_tokens WHERE user_id = ?", itoa(shared.owner.ID)) != token {
		t.Error("the tokens of another person changed")
	}

	if shared.count(t, "SELECT COUNT(*) FROM users WHERE id = ?", shared.owner.ID) != 1 ||
		shared.count(t, "SELECT COUNT(*) FROM memberships WHERE user_id = ? AND workspace_id = ?", shared.owner.ID, shared.space.ID) != 1 {
		t.Error("Bob lost his account or his place")
	}

	if events, err := ana.svc.Audit(t.Context(), ana.svc.Ref("user", ana.owner.ID)); err != nil || len(events) != 2 ||
		events[1].Event != "account.purged" {
		t.Errorf("audit = %+v, %v", events, err)
	}

	t.Run("running it again changes nothing", func(t *testing.T) {
		audit := ana.count(t, "SELECT COUNT(*) FROM privacy_audit")

		rep, err := ana.svc.Purge(t.Context())
		if err != nil || rep.Accounts != 0 || rep.Workspaces != 0 {
			t.Fatalf("report = %+v, %v", rep, err)
		}

		if err := ana.svc.PurgeAccount(t.Context(), ana.owner.ID); err != nil {
			t.Fatalf("purging a purged account: %v", err)
		}

		if ana.count(t, "SELECT COUNT(*) FROM privacy_audit") != audit {
			t.Error("the second pass wrote audit rows")
		}
	})
}

func TestPurgeAccountWaitsForItsWorkspaces(t *testing.T) {
	t.Parallel()

	f := newFakeStripe(t)
	w := billingWorld(t, f, "ana@example.com")

	// Request while Stripe works, then it fails at the purge.
	if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatal(err)
	}

	w.exec(t, "UPDATE workspace_subscriptions SET status = 'active' WHERE workspace_id = ?", w.space.ID)
	f.setFail(true)
	w.makeAccountDue(t, w.owner.ID)

	rep, err := w.svc.Purge(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if rep.Accounts != 0 || rep.Blocked != 2 {
		t.Fatalf("report = %+v, want the workspace and the account both waiting", rep)
	}

	if w.count(t, "SELECT COUNT(*) FROM users WHERE id = ?", w.owner.ID) != 1 {
		t.Fatal("the account went before its workspace")
	}

	f.setFail(false)

	rep, err = w.svc.Purge(t.Context())
	if err != nil || rep.Accounts != 1 || rep.Workspaces != 1 {
		t.Fatalf("after Stripe recovered: %+v, %v", rep, err)
	}
}
