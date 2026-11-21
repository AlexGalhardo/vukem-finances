package tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestPixStartAmountIgnoresTheRequest(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	ana := signUp(t, app, "Ana", "ana@example.com")

	res := ana.Client.PostForm("/billing/pix", url.Values{"months": {"3"}, "amount": {"1"}, "amountCents": {"1"}})
	res.AssertStatus(http.StatusCreated)
	res.AssertJSONPath("amountCents", float64(3*1990))
	res.AssertJSONPath("status", "pending")

	if got := f.created()[0]["amount"]; got != float64(3*1990) {
		t.Errorf("the provider was asked to charge %v", got)
	}

	for _, months := range []string{"2", "0", "-1", "x", ""} {
		if r := ana.Client.PostForm("/billing/pix", url.Values{"months": {months}}); r.Status() != http.StatusUnprocessableEntity {
			t.Errorf("months %q answered %d, want 422", months, r.Status())
		}
	}
}

func TestOnlyTheOwnerStartsOrPollsAPixPayment(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 1)

	for _, role := range []string{models.RoleAdmin, models.RoleMember, models.RoleViewer} {
		u := newVerifiedUser(t, app, role, role+"@example.com")
		join(t, app, ana, u, role)
		c := browser(t, app, u, ana.Workspace.ID)

		c.PostForm("/billing/pix", url.Values{"months": {"1"}}).AssertForbidden()
		c.Get("/billing/pix/" + p.PublicRef).AssertForbidden()
	}

	if n := len(f.created()); n != 1 {
		t.Errorf("a non-owner caused a charge: %d created", n)
	}

	ana.Client.Get("/billing/pix/" + p.PublicRef).AssertOK()
}

func TestPixPaymentOfAnotherWorkspaceIsNotFound(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	p := startPix(t, b, ana, 1)
	f.setStatus(p.ProviderRef, "PAID")

	bob.Client.Get("/billing/pix/" + p.PublicRef).AssertNotFound()
	bob.Client.Get("/billing/pix/not-a-ref").AssertNotFound()

	// Bob's request neither read nor activated Ana's payment.
	if got := pixPayments(t, app)[0].Status; got != models.PixPending {
		t.Errorf("status = %q after a foreign poll", got)
	}

	assertPlan(t, bob, "free")
}

func TestPixPollActivatesAndTheBillingPageShowsIt(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	ana := signUp(t, app, "Ana", "ana@example.com")

	page := ana.Client.Inertia("").Get("/billing").AssertInertia("Billing")
	page.PropEquals("pix.currency", "BRL")
	page.PropEquals("pix.options.1.months", float64(3))
	page.PropEquals("pix.options.1.totalCents", float64(3*1990))

	var ref, provider string

	res := ana.Client.PostForm("/billing/pix", url.Values{"months": {"1"}})
	ref = pixPayments(t, app)[0].PublicRef
	provider = pixPayments(t, app)[0].ProviderRef
	res.AssertStatus(http.StatusCreated)

	poll := ana.Client.Get("/billing/pix/" + ref)
	poll.AssertOK()
	poll.AssertJSONPath("status", "pending")

	f.setStatus(provider, "PAID")

	poll = ana.Client.Get("/billing/pix/" + ref)
	poll.AssertJSONPath("status", "paid")
	poll.AssertJSONPath("plan", "pro")

	page = ana.Client.Inertia("").Get("/billing").AssertInertia("Billing")
	page.PropEquals("limits.plan", "pro")
	page.PropEquals("pixAccess.active", true)
	page.PropEquals("pixAccess.validUntil", "February 28, 2027")

	// A refund shows its reason.
	f.setStatus(provider, "REFUNDED")

	if err := pixBilling(t, app).HandlePixEvent(t.Context(), provider); err != nil {
		t.Fatal(err)
	}

	page = ana.Client.Inertia("").Get("/billing").AssertInertia("Billing")
	page.PropEquals("limits.plan", "free")
	page.PropEquals("pixAccess.active", false)
	page.PropEquals("pixAccess.revokedReason", "refund")
}

func TestPixPollIsRateLimited(t *testing.T) {
	t.Parallel()

	_, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	ana := signUp(t, app, "Ana", "ana@example.com")
	ref := startPix(t, pixBilling(t, app), ana, 1).PublicRef

	limited := 0

	for range 40 {
		if ana.Client.Get("/billing/pix/"+ref).Status() == http.StatusTooManyRequests {
			limited++
		}
	}

	if limited == 0 {
		t.Error("40 polls in a row were never limited")
	}
}

func TestPixIsAbsentWithoutConfigurationAndInThePersonalEdition(t *testing.T) {
	t.Parallel()

	saas := newApp(t)
	ana := signUp(t, saas, "Ana", "ana@example.com")
	ana.Client.PostForm("/billing/pix", url.Values{"months": {"1"}}).AssertStatus(http.StatusServiceUnavailable)
	ana.Client.Inertia("").Get("/billing").AssertInertia("Billing").PropEquals("pix", nil)

	f := newFakeAbacatePay(t, time.Now)
	env := pixEnv(f)
	personal := newPersonalApp(t, env)
	owner := signUp(t, personal, "Ana", "ana@example.com").Client

	owner.PostForm("/billing/pix", url.Values{"months": {"1"}}).AssertNotFound()
	owner.Get("/billing/pix/anything").AssertNotFound()
	owner.Get("/billing").AssertNotFound()
	vukemtest.AssertDatabaseCount(t, personal, "pix_payments", 0)
}
