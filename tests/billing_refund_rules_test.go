package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// setInvoicePeriod makes the invoice in_1 say which period it paid for.
func (f *fakeStripe) setInvoicePeriod(end time.Time, hasMore bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.invoices["in_1"]["lines"] = map[string]any{
		"has_more": hasMore,
		"data":     []any{map[string]any{"period": map[string]any{"end": end.Unix()}}},
	}
}

func TestRefundOfAnEndedPeriodKeepsTheSubscription(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_old", 900, 900)
	f.setInvoicePeriod(app.Now().Add(-24*time.Hour), false)

	for _, id := range []string{"evt_r1", "evt_r2"} {
		postStripe(t, app, id, "charge.refunded", map[string]any{"id": "ch_old"}).AssertOK()
	}

	assertPlan(t, ana, "pro")

	if n := len(f.canceled()); n != 0 {
		t.Errorf("cancel calls = %d, want 0", n)
	}

	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"kind": "past_period_refund"})

	if got := rowCountWhere(t, app, "billing_events", "kind = 'past_period_refund'"); got != 1 {
		t.Errorf("past_period_refund events = %d, want 1", got)
	}
}

func TestRefundRevokesWhenThePeriodIsCurrentOrUnknown(t *testing.T) {
	t.Parallel()

	for name, set := range map[string]func(*fakeStripe, time.Time){
		"current period":   func(f *fakeStripe, now time.Time) { f.setInvoicePeriod(now.Add(24*time.Hour), false) },
		"incomplete lines": func(f *fakeStripe, now time.Time) { f.setInvoicePeriod(now.Add(-24*time.Hour), true) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f, app, ana := proViaStripe(t)
			f.setCharge("ch_1", 900, 900)
			set(f, app.Now())

			postStripe(t, app, "evt_r", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
			assertPlan(t, ana, "free")
		})
	}
}

func TestLateDeliveryOfACurrentPeriodRefundStillRevokes(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)

	now := app.Now()
	f.setInvoicePeriod(now.Add(-time.Hour), false) // ended after the refund, before this delivery

	postStripeAt(t, app, "evt_late", "charge.refunded", map[string]any{"id": "ch_1"}, now.Add(-2*time.Hour)).AssertOK()
	assertPlan(t, ana, "free")

	if n := len(f.canceled()); n != 1 {
		t.Errorf("cancel calls = %d, want 1", n)
	}
}

func TestRetryAfterThePeriodEndedFinishesTheCancel(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 900)
	f.setInvoicePeriod(app.Now().Add(24*time.Hour), false)

	f.mu.Lock()
	f.cancelErr = true
	f.mu.Unlock()

	if r := postStripe(t, app, "evt_x", "charge.refunded", map[string]any{"id": "ch_1"}); r.Status() < 500 {
		t.Fatalf("a failed cancel answered %d, want a retryable error", r.Status())
	}

	f.mu.Lock()
	f.cancelErr = false
	f.mu.Unlock()
	f.setInvoicePeriod(app.Now().Add(-time.Hour), false)

	postStripe(t, app, "evt_x", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
	assertPlan(t, ana, "free")

	if n := len(f.canceled()); n != 1 {
		t.Errorf("cancel calls after the retry = %d, want 1", n)
	}
}

func TestDisputeWonWhilePastDueRestoresAccessOnRecovery(t *testing.T) {
	t.Parallel()

	f, app, ana := proViaStripe(t)
	f.setCharge("ch_1", 900, 0)
	f.setDispute("dp_1", "ch_1", "needs_response")
	disputeEvent(t, app, "evt_d1", "charge.dispute.created")
	assertPlan(t, ana, "free")

	f.setSubscription(testCustomer, "past_due")
	f.setDispute("dp_1", "ch_1", "won")
	disputeEvent(t, app, "evt_d2", "charge.dispute.closed")
	assertPlan(t, ana, "free") // past_due does not grant Pro by itself

	f.setSubscription(testCustomer, "active")
	postStripe(t, app, "evt_upd", "customer.subscription.updated",
		map[string]any{"id": testSubscript, "customer": testCustomer}).AssertOK()
	assertPlan(t, ana, "pro")
}
