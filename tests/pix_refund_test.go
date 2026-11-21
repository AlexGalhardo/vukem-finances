package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// paidPix starts a charge, has the provider report it paid and applies it.
func paidPix(t *testing.T, f *fakeAbacatePay, b *billing.Billing, w who, months int) models.PixPayment {
	t.Helper()

	p := startPix(t, b, w, months)
	f.setStatus(p.ProviderRef, "PAID")

	if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
		t.Fatal(err)
	}

	return p
}

func TestPixRefundRevokes(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := paidPix(t, f, b, ana, 3)
	assertPlan(t, ana, "pro")

	f.setStatus(p.ProviderRef, "REFUNDED")

	for range 3 {
		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}
	}

	assertPlan(t, ana, "free")

	if got := pixPayments(t, app)[0].Status; got != models.PixRefunded {
		t.Errorf("status = %q, want refunded", got)
	}

	ent := pixEntitlementOf(t, app, ana.Workspace.ID)
	if ent.PaidMonths != 0 || !ent.RevokedAt.Valid || ent.RevokedReason != models.RevokedRefund {
		t.Errorf("entitlement = %+v", ent)
	}

	revoked := 0

	for _, m := range app.Mails().SentTo("ana@example.com") {
		if m.Subject == "Your Pro access has ended" {
			revoked++
		}
	}

	if revoked != 1 {
		t.Errorf("revocation mails = %d, want 1", revoked)
	}

	// Buying again after a refund works and starts a fresh period.
	paidPix(t, f, b, ana, 1)
	assertPlan(t, ana, "pro")

	if e := pixEntitlementOf(t, app, ana.Workspace.ID); e.PaidMonths != 1 || e.RevokedAt.Valid {
		t.Errorf("entitlement after buying again = %+v", e)
	}
}

func TestPixRefundKeepsOtherPaidMonths(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	first := paidPix(t, f, b, ana, 1)
	paidPix(t, f, b, ana, 3)

	f.setStatus(first.ProviderRef, "REFUNDED")

	if err := b.HandlePixEvent(t.Context(), first.ProviderRef); err != nil {
		t.Fatal(err)
	}

	assertPlan(t, ana, "pro")

	ent := pixEntitlementOf(t, app, ana.Workspace.ID)
	if ent.PaidMonths != 3 || ent.ValidUntil.UTC().Format("2006-01-02") != "2027-04-30" {
		t.Errorf("entitlement = %d months until %s, want 3 months until 2027-04-30", ent.PaidMonths, ent.ValidUntil)
	}

	// The owner is told only when access actually ended.
	for _, m := range app.Mails().SentTo("ana@example.com") {
		if m.Subject == "Your Pro access has ended" {
			t.Error("a refund that left paid months sent an \"ended\" mail")
		}
	}
}

func TestPixRefundOfEndedPeriodKeepsCurrentPeriod(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	old := paidPix(t, f, b, ana, 1)

	// The login session does not survive this jump, so access is read from
	// the entitlement instead of the dashboard.
	clock.Advance(45 * 24 * time.Hour)

	if pixEntitlementOf(t, app, ana.Workspace.ID).Granting(clock.Now()) {
		t.Fatal("first period should have ended")
	}

	second := startPix(t, b, ana, 1)
	f.setStatus(second.ProviderRef, "PAID")

	if err := b.HandlePixEvent(t.Context(), second.ProviderRef); err != nil {
		t.Fatal(err)
	}

	before := pixEntitlementOf(t, app, ana.Workspace.ID)

	f.setStatus(old.ProviderRef, "REFUNDED")

	if err := b.HandlePixEvent(t.Context(), old.ProviderRef); err != nil {
		t.Fatal(err)
	}

	after := pixEntitlementOf(t, app, ana.Workspace.ID)
	if !after.Granting(clock.Now()) {
		t.Errorf("refunding an ended period cut current access: %+v", after)
	}

	if after.PaidMonths != before.PaidMonths || !after.ValidUntil.Equal(before.ValidUntil) {
		t.Errorf("entitlement changed: before %+v, after %+v", before, after)
	}

	for _, m := range app.Mails().SentTo("ana@example.com") {
		if m.Subject == "Your Pro access has ended" {
			t.Error("refunding an ended period sent an \"ended\" mail")
		}
	}
}

func TestPixDisputeWonOnUsedMonthsGrantsNothing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		newPeriod bool
	}{{"lapsed with no new payment", false}, {"lapsed with a new period", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
			f, app := pixApp(t, clock)
			b := pixBilling(t, app)
			ana := signUp(t, app, "Ana", "ana@example.com")
			old := paidPix(t, f, b, ana, 1)

			clock.Advance(45 * 24 * time.Hour)

			if tc.newPeriod {
				next := startPix(t, b, ana, 1)
				f.setStatus(next.ProviderRef, "PAID")

				if err := b.HandlePixEvent(t.Context(), next.ProviderRef); err != nil {
					t.Fatal(err)
				}
			}

			before := pixEntitlementOf(t, app, ana.Workspace.ID)

			for _, status := range []string{"UNDER_DISPUTE", "PAID"} {
				f.setStatus(old.ProviderRef, status)

				if err := b.HandlePixEvent(t.Context(), old.ProviderRef); err != nil {
					t.Fatal(err)
				}
			}

			after := pixEntitlementOf(t, app, ana.Workspace.ID)
			if after.PaidMonths != before.PaidMonths || !after.ValidUntil.Equal(before.ValidUntil) ||
				after.Period != before.Period {
				t.Errorf("entitlement changed: before %+v, after %+v", before, after)
			}

			if after.Granting(clock.Now()) != tc.newPeriod {
				t.Errorf("granting = %v, want %v", after.Granting(clock.Now()), tc.newPeriod)
			}
		})
	}
}

func TestPixRefundAfterDisputeWonRevokes(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := paidPix(t, f, b, ana, 1)

	for _, status := range []string{"UNDER_DISPUTE", "PAID", "REFUNDED"} {
		f.setStatus(p.ProviderRef, status)

		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}
	}

	assertPlan(t, ana, "free")
}

func TestPixDisputeRevokesAndWonRestores(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := paidPix(t, f, b, ana, 1)

	f.setStatus(p.ProviderRef, "UNDER_DISPUTE")

	for range 2 {
		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}
	}

	assertPlan(t, ana, "free")

	f.setStatus(p.ProviderRef, "PAID") // won
	if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
		t.Fatal(err)
	}

	assertPlan(t, ana, "pro")

	// Lost: refunded after the dispute, the months are not subtracted twice.
	f.setStatus(p.ProviderRef, "UNDER_DISPUTE")
	_ = b.HandlePixEvent(t.Context(), p.ProviderRef)
	f.setStatus(p.ProviderRef, "REFUNDED")
	_ = b.HandlePixEvent(t.Context(), p.ProviderRef)
	assertPlan(t, ana, "free")

	if ent := pixEntitlementOf(t, app, ana.Workspace.ID); ent.PaidMonths != 0 {
		t.Errorf("paid months = %d, want 0", ent.PaidMonths)
	}
}

func TestReconcileRevokesRefunded(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")

	// A payment that is older than the window is not read.
	old := paidPix(t, f, b, ana, 1)
	clock.Advance(billing.ReconcileWindow + time.Hour)

	refunded := paidPix(t, f, b, ana, 1)
	kept := paidPix(t, f, b, ana, 3)

	f.setStatus(refunded.ProviderRef, "REFUNDED")
	f.setStatus(old.ProviderRef, "REFUNDED")

	if err := b.ReconcilePix(t.Context()); err != nil {
		t.Fatal(err)
	}

	byRef := map[string]string{}
	for _, p := range pixPayments(t, app) {
		byRef[p.ProviderRef] = p.Status
	}

	if byRef[refunded.ProviderRef] != "refunded" || byRef[old.ProviderRef] != "paid" || byRef[kept.ProviderRef] != "paid" {
		t.Errorf("statuses = %v", byRef)
	}

	// A second run changes nothing.
	before := pixEntitlementOf(t, app, ana.Workspace.ID)

	if err := b.ReconcilePix(t.Context()); err != nil {
		t.Fatal(err)
	}

	if after := pixEntitlementOf(t, app, ana.Workspace.ID); after.PaidMonths != before.PaidMonths {
		t.Errorf("a second run changed paid months %d -> %d", before.PaidMonths, after.PaidMonths)
	}
}

func TestReconcileResolvesTimersFromTheProvider(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	expired := startPix(t, b, ana, 1)
	late := startPix(t, b, ana, 1)
	open := startPix(t, b, ana, 1)

	clock.Advance(time.Hour)
	f.setStatus(expired.ProviderRef, "EXPIRED")
	f.setStatus(late.ProviderRef, "PAID") // paid, webhook lost

	if err := b.ReconcilePix(t.Context()); err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, p := range pixPayments(t, app) {
		got[p.ProviderRef] = p.Status
	}

	if got[expired.ProviderRef] != "expired" || got[late.ProviderRef] != "paid" || got[open.ProviderRef] != "pending" {
		t.Errorf("statuses = %v: only the provider may expire a charge", got)
	}

	assertPlan(t, ana, "pro")
}

func TestAdminMarkRefunded(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	boss := signUp(t, app, "Boss", "boss@example.com")
	p := paidPix(t, f, b, ana, 1)
	url := "/admin/pix-payments/" + p.PublicRef + "/refund"

	// Only an admin can use it: everyone else gets 404.
	ana.Client.PostForm(url, nil).AssertNotFound()
	assertPlan(t, ana, "pro")

	boss.Client.PostForm(url, nil).AssertRedirect("/admin/revenue")
	boss.Client.PostForm(url, nil).AssertRedirect("/admin/revenue") // idempotent

	assertPlan(t, ana, "free")
	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"source": "admin", "kind": "admin_mark_refunded"})
	vukemtest.AssertDatabaseHas(t, app, "billing_events", map[string]any{"source": "admin", "kind": "pix_refunded"})

	if got := pixPayments(t, app)[0].Status; got != models.PixRefunded {
		t.Errorf("status = %q", got)
	}

	boss.Client.PostForm("/admin/pix-payments/nope/refund", nil).AssertNotFound()
}
