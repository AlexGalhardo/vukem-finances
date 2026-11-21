package tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// proViaStripe signs a workspace up and makes it Pro through the same webhook
// path production uses.
func proViaStripe(t *testing.T) (*fakeStripe, *vukemtest.App, who) {
	t.Helper()

	f := newFakeStripe(t)
	app := billingApp(t, f)
	ana := signUp(t, app, "Ana", "ana@example.com")

	ana.Client.PostForm("/billing/checkout/pro", nil).AssertStatus(303)
	f.setSubscription(testCustomer, "active")
	postStripe(t, app, "evt_sub", "customer.subscription.created",
		map[string]any{"id": testSubscript, "customer": testCustomer}).AssertOK()
	assertPlan(t, ana, "pro")

	return f, app, ana
}

func assertPlan(t *testing.T, w who, plan string) {
	t.Helper()

	w.Client.Inertia("").Get("/dashboard").AssertInertia("Dashboard").PropEquals("workspace.plan", plan)
}

func TestRefundRevokesPlan(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)

	postStripe(t, app, "evt_refund", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseHas(t, app, "workspace_subscriptions", map[string]any{
		"stripe_subscription_id": testSubscript, "revoked_reason": "refund", "revoked_ref": "ch_1", "status": "canceled",
	})
	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"kind": "access_revoked", "source": "stripe"})

	cancels := f.canceled()
	if len(cancels) != 1 || cancels[0].Get("prorate") != "false" || cancels[0].Get("invoice_now") != "false" {
		t.Errorf("cancel calls = %v, want one without proration and final invoice", cancels)
	}

	if !hasSubject(app, "ana@example.com", "Your Pro access has ended") {
		t.Error("the owner was not told")
	}

	// A later "active" from Stripe does not restore the access.
	f.setSubscription(testCustomer, "active")
	postStripe(t, app, "evt_upd", "customer.subscription.updated",
		map[string]any{"id": testSubscript, "customer": testCustomer}).AssertOK()
	assertPlan(t, ana, "free")
}

func TestPartialRefundKeepsPlan(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 300)

	postStripe(t, app, "evt_p1", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
	postStripe(t, app, "evt_p2", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()

	assertPlan(t, ana, "pro")
	vukemtest.AssertDatabaseCount(t, app, "billing_events", 1)
	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"kind": "partial_refund"})

	if n := len(f.canceled()); n != 0 {
		t.Errorf("a partial refund cancelled the subscription %d times", n)
	}
}

func TestRefundIsIdempotent(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)

	// The same event, then 50 more deliveries of the same refund under new ids.
	postStripe(t, app, "evt_r", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
	postStripe(t, app, "evt_r", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()

	for i := range 50 {
		postStripe(t, app, "evt_again_"+strconv.Itoa(i), "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
	}

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseCount(t, app, "billing_events", 1)

	if n := len(f.canceled()); n != 1 {
		t.Errorf("cancel calls = %d, want 1", n)
	}

	if n := len(app.Mails().SentTo("ana@example.com")); n != 1 {
		t.Errorf("mails = %d, want 1", n)
	}
}

func TestRefundRetriesTheCancelAndNeverLeaksProviderText(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)

	f.mu.Lock()
	f.cancelErr = true
	f.mu.Unlock()

	r := postStripe(t, app, "evt_x", "charge.refunded", map[string]any{"id": "ch_1"})
	if r.Status() < 500 {
		t.Errorf("a failed cancel answered %d, want a retryable error", r.Status())
	}

	if body := r.Body(); strings.Contains(body, "secret-detail-from-stripe") {
		t.Errorf("provider text reached the response: %s", body)
	}

	// Access is already gone: the local revoke does not wait for Stripe.
	assertPlan(t, ana, "free")

	f.mu.Lock()
	f.cancelErr = false
	f.mu.Unlock()

	postStripe(t, app, "evt_x", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()

	if n := len(f.canceled()); n != 1 {
		t.Errorf("cancel calls after the retry = %d, want 1", n)
	}

	vukemtest.AssertDatabaseCount(t, app, "billing_events", 1)
}

func disputeEvent(t *testing.T, app *vukemtest.App, evt, typ string) {
	t.Helper()
	postStripe(t, app, evt, typ, map[string]any{"id": "dp_1"}).AssertOK()
}

func TestDisputeRevokesPlan(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "needs_response")

	disputeEvent(t, app, "evt_d1", "charge.dispute.created")
	disputeEvent(t, app, "evt_d1b", "charge.dispute.created")

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseHas(t, app, "workspace_subscriptions", map[string]any{
		"stripe_subscription_id": testSubscript, "revoked_reason": "dispute", "revoked_ref": "ch_1", "status": "active",
	})
	vukemtest.AssertDatabaseCount(t, app, "billing_events", 1)

	if n := len(f.canceled()); n != 0 {
		t.Errorf("an open dispute cancelled the subscription %d times", n)
	}

	if !hasSubject(app, "ana@example.com", "Your Pro access has ended") {
		t.Error("the owner was not told")
	}
}

func TestDisputeWonReinstates(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "needs_response")
	disputeEvent(t, app, "evt_d1", "charge.dispute.created")
	assertPlan(t, ana, "free")

	f.setDispute("dp_1", "ch_1", "won")
	disputeEvent(t, app, "evt_d2", "charge.dispute.closed")
	assertPlan(t, ana, "pro")
	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"kind": "access_reinstated"})

	// A replay of the opening event after the win cannot revoke again.
	disputeEvent(t, app, "evt_d3", "charge.dispute.created")
	assertPlan(t, ana, "pro")
}

func TestDisputeLostStaysRevoked(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "needs_response")
	disputeEvent(t, app, "evt_d1", "charge.dispute.created")

	f.setDispute("dp_1", "ch_1", "lost")
	disputeEvent(t, app, "evt_d2", "charge.dispute.closed")
	disputeEvent(t, app, "evt_d3", "charge.dispute.closed")

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseHas(t, app, "workspace_subscriptions", map[string]any{
		"stripe_subscription_id": testSubscript, "revoked_reason": "dispute_lost", "status": "canceled",
	})

	if n := len(f.canceled()); n != 1 {
		t.Errorf("cancel calls = %d, want 1", n)
	}
}

func TestDisputeWonDoesNotUndoRefund(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)
	f.setDispute("dp_1", "ch_1", "won")

	postStripe(t, app, "evt_refund", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
	disputeEvent(t, app, "evt_d1", "charge.dispute.closed")

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseHas(t, app, "workspace_subscriptions", map[string]any{"revoked_reason": "refund"})
}

func TestSyncDoesNotRegrantRevoked(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "under_review")
	disputeEvent(t, app, "evt_d1", "charge.dispute.created")

	for i, typ := range []string{
		"customer.subscription.updated", "customer.subscription.resumed", "customer.subscription.created",
	} {
		f.setSubscription(testCustomer, "active")
		postStripe(t, app, "evt_sync_"+strconv.Itoa(i), typ,
			map[string]any{"id": testSubscript, "customer": testCustomer}).AssertOK()
		assertPlan(t, ana, "free")
	}

	postStripe(t, app, "evt_inv", "invoice.paid", map[string]any{
		"id": "in_9", "customer": testCustomer, "amount_paid": 900, "currency": "usd",
		"parent": map[string]any{"subscription_details": map[string]any{"subscription": testSubscript}},
	}).AssertOK()
	assertPlan(t, ana, "free")
}

func TestDisputeEventsOutOfOrder(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "won")

	// The closing event arrives first and the opening one after it.
	disputeEvent(t, app, "evt_closed", "charge.dispute.closed")
	disputeEvent(t, app, "evt_created", "charge.dispute.created")
	assertPlan(t, ana, "pro")
}
