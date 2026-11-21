package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func newPixClock(start time.Time) *vukemtest.FakeClock { return vukemtest.NewFakeClock(start) }

// startPix starts a PIX charge as the owner of w and returns the stored payment.
func startPix(t *testing.T, b *billing.Billing, w who, months int) models.PixPayment {
	t.Helper()

	p, err := b.StartPix(t.Context(), w.Workspace, w.User.ID, months)
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func TestPixAmountComesFromTheServer(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")

	p := startPix(t, b, ana, 3)
	if p.AmountCents != 3*1990 || p.Months != 3 || p.Status != models.PixPending || p.Currency != "BRL" {
		t.Errorf("payment = %+v", p)
	}

	if got := f.created()[0]["amount"]; got != float64(3*1990) || f.created()[0]["expiresIn"] != float64(1800) {
		t.Errorf("the provider was asked %v", f.created()[0])
	}

	for _, months := range []int{0, 2, -1, 13, 1000} {
		if _, err := b.StartPix(t.Context(), ana.Workspace, ana.User.ID, months); err == nil {
			t.Errorf("%d months were accepted", months)
		}
	}

	if n := len(f.created()); n != 1 {
		t.Errorf("provider saw %d charges, want 1", n)
	}
}

func TestPixActivationIsIdempotent(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 3)

	// The provider still says pending: a forged "paid" delivery changes nothing.
	if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
		t.Fatal(err)
	}

	assertPlan(t, ana, "free")

	f.setStatus(p.ProviderRef, "PAID")

	for range 5 {
		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}
	}

	assertPlan(t, ana, "pro")

	if ent := pixEntitlementOf(t, app, ana.Workspace.ID); ent.PaidMonths != 3 {
		t.Errorf("paid months = %d, want 3", ent.PaidMonths)
	}

	vukemtest.AssertDatabaseCount(t, app, "billing_events", 2) // pix_created, pix_paid

	receipts := 0

	for _, m := range app.Mails().SentTo("ana@example.com") {
		if len(m.Subject) > 8 && m.Subject[:8] == "Your PIX" {
			receipts++
		}
	}

	if receipts != 1 {
		t.Errorf("receipts = %d, want 1", receipts)
	}
}

func TestPixConcurrentWebhooks(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 1)
	f.setStatus(p.ProviderRef, "PAID")

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	if ent := pixEntitlementOf(t, app, ana.Workspace.ID); ent.PaidMonths != 1 {
		t.Errorf("paid months = %d after 50 deliveries, want 1", ent.PaidMonths)
	}

	vukemtest.AssertDatabaseCount(t, app, "billing_events", 2)
}

func TestPixJan31(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")

	for i, wantDay := range []string{"2027-02-28", "2027-03-31"} {
		p := startPix(t, b, ana, 1)
		f.setStatus(p.ProviderRef, "PAID")

		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}

		ent := pixEntitlementOf(t, app, ana.Workspace.ID)
		if got := ent.ValidUntil.UTC().Format("2006-01-02"); got != wantDay {
			t.Errorf("after payment %d the plan ends %s, want %s", i+1, got, wantDay)
		}
	}
}

func TestPixStackedPaymentsDoNotDrift(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")

	pay := func(months int) {
		p := startPix(t, b, ana, months)
		f.setStatus(p.ProviderRef, "PAID")

		if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
			t.Fatal(err)
		}
	}

	pay(1)
	clock.Advance(24 * time.Hour) // Feb 1, still valid
	pay(1)
	pay(3)

	ent := pixEntitlementOf(t, app, ana.Workspace.ID)
	if ent.PaidMonths != 5 || ent.ValidUntil.UTC().Format("2006-01-02") != "2027-06-30" {
		t.Errorf("entitlement = %d months until %s, want 5 months until 2027-06-30", ent.PaidMonths, ent.ValidUntil)
	}

	// After the end, a new payment starts again from now.
	clock.Set(time.Date(2027, 8, 15, 9, 0, 0, 0, time.UTC))
	pay(1)

	ent = pixEntitlementOf(t, app, ana.Workspace.ID)
	if ent.PaidMonths != 1 || ent.ValidUntil.UTC().Format("2006-01-02") != "2027-09-15" {
		t.Errorf("entitlement after a lapse = %d months until %s", ent.PaidMonths, ent.ValidUntil)
	}
}

func TestPixPaidAfterTimerActivates(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 1)

	// The QR timer ran out locally, but the provider never said expired.
	clock.Advance(40 * time.Minute)

	got, err := b.RefreshPix(t.Context(), p)
	if err != nil || got.Status != models.PixPending {
		t.Fatalf("after the timer, status = %q, %v: only the provider may expire a charge", got.Status, err)
	}

	f.setStatus(p.ProviderRef, "PAID")

	if err := b.HandlePixEvent(t.Context(), p.ProviderRef); err != nil {
		t.Fatal(err)
	}

	assertPlan(t, ana, "pro")

	// The provider confirms an expiry, and a payment that arrives even later still counts.
	q := startPix(t, b, ana, 1)
	f.setStatus(q.ProviderRef, "EXPIRED")

	if got, err := b.RefreshPix(t.Context(), q); err != nil || got.Status != models.PixExpired {
		t.Fatalf("status = %q, %v, want expired", got.Status, err)
	}

	f.setStatus(q.ProviderRef, "PAID")

	if err := b.HandlePixEvent(t.Context(), q.ProviderRef); err != nil {
		t.Fatal(err)
	}

	if ent := pixEntitlementOf(t, app, ana.Workspace.ID); ent.PaidMonths != 2 {
		t.Errorf("paid months = %d, want 2 (one per payment)", ent.PaidMonths)
	}
}
