package privacy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const stripeKey = "rk_test_privacy"

// fakeStripe answers the one call a deletion makes: cancel a subscription.
type fakeStripe struct {
	srv *httptest.Server

	mu      sync.Mutex
	cancels []string
	fail    bool
}

func newFakeStripe(t *testing.T) *fakeStripe {
	t.Helper()

	f := &fakeStripe{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+stripeKey {
			t.Errorf("%s %s: wrong Authorization", r.Method, r.URL.Path)
		}

		f.mu.Lock()
		defer f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodDelete || len(r.URL.Path) <= len("/v1/subscriptions/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if f.fail {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "api_error", "message": "secret-detail-from-stripe"}})

			return
		}

		id := r.URL.Path[len("/v1/subscriptions/"):]
		f.cancels = append(f.cancels, id)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "canceled"})
	}))
	t.Cleanup(f.srv.Close)

	return f
}

func (f *fakeStripe) cancelled() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.cancels...)
}

func (f *fakeStripe) setFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fail = v
}

// billingWorld is a world on a saas app wired to the fake Stripe, with an active subscription.
func billingWorld(t *testing.T, f *fakeStripe, email string) *world {
	t.Helper()

	app := newApp(t, map[string]string{
		"STRIPE_KEY": stripeKey, "STRIPE_WEBHOOK_SECRET": "whsec_privacy", "STRIPE_API_BASE": f.srv.URL,
		"STRIPE_PRICE_PRO": "price_pro", "STRIPE_CURRENCIES": "usd",
	})

	return billingWorldOn(t, app, email)
}

func billingWorldOn(t *testing.T, app *vukemtest.App, email string) *world {
	t.Helper()

	w := newWorld(t, app, email)
	w.withAccess(t)

	b, err := billing.New(app.App, w.ws)
	if err != nil {
		t.Fatal(err)
	}

	w.svc.Billing = b

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	sub := models.WorkspaceSubscription{
		WorkspaceID: w.space.ID, StripeSubscriptionID: "sub_" + itoa(w.space.ID), Status: "active", PriceID: "price_pro", UnitAmount: 900,
	}
	if err := orm.Create(t.Context(), db, &sub); err != nil {
		t.Fatal(err)
	}

	return w
}

func (w *world) subscriptionStatus(t *testing.T) string {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var status string
	if err := db.QueryRowContext(t.Context(), "SELECT status FROM workspace_subscriptions WHERE workspace_id = ?", w.space.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}

	return status
}

func TestBillingCancellation(t *testing.T) {
	t.Parallel()

	t.Run("a workspace deletion cancels the subscription once", func(t *testing.T) {
		t.Parallel()

		f := newFakeStripe(t)
		w := billingWorld(t, f, "ana@example.com")
		ctx := workspaces.WithTenant(t.Context(), w.space.ID)

		if err := w.svc.RequestWorkspaceDeletion(ctx, w.owner, w.space.Name); err != nil {
			t.Fatal(err)
		}

		if got := f.cancelled(); len(got) != 1 || got[0] != "sub_"+itoa(w.space.ID) {
			t.Fatalf("cancels = %v", got)
		}

		if got := w.subscriptionStatus(t); got != "canceled" {
			t.Fatalf("local status = %s", got)
		}

		// Idempotent: calling it again asks Stripe nothing.
		if err := w.svc.Billing.CancelWorkspaceSubscriptions(t.Context(), w.space.ID); err != nil {
			t.Fatal(err)
		}

		if got := f.cancelled(); len(got) != 1 {
			t.Fatalf("a second call cancelled again: %v", got)
		}

		if w.reload(t).PurgeError != "" {
			t.Fatalf("purge_error = %q", w.reload(t).PurgeError)
		}
	})

	t.Run("a failure keeps the request and blocks the purge until it succeeds", func(t *testing.T) {
		t.Parallel()

		f := newFakeStripe(t)
		f.setFail(true)

		w := billingWorld(t, f, "ana@example.com")
		ctx := workspaces.WithTenant(t.Context(), w.space.ID)

		if err := w.svc.RequestWorkspaceDeletion(ctx, w.owner, w.space.Name); err != nil {
			t.Fatalf("a Stripe failure must not fail the request: %v", err)
		}

		ws := w.reload(t)
		if !ws.PendingDeletion() || ws.PurgeError != "billing_cancel_failed" {
			t.Fatalf("workspace = %+v, want pending with the purge blocked", ws)
		}

		if w.subscriptionStatus(t) != "active" {
			t.Fatal("the local row says canceled though Stripe refused")
		}

		// The next attempt, after Stripe recovers, finishes the job.
		f.setFail(false)

		if err := w.svc.Billing.CancelWorkspaceSubscriptions(t.Context(), w.space.ID); err != nil {
			t.Fatal(err)
		}

		if w.subscriptionStatus(t) != "canceled" || len(f.cancelled()) != 1 {
			t.Fatalf("status %s, cancels %v", w.subscriptionStatus(t), f.cancelled())
		}
	})

	t.Run("an account deletion cancels the subscription of the workspace that goes with it", func(t *testing.T) {
		t.Parallel()

		f := newFakeStripe(t)
		w := billingWorld(t, f, "ana@example.com")

		if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
			t.Fatal(err)
		}

		if got := f.cancelled(); len(got) != 1 {
			t.Fatalf("cancels = %v", got)
		}
	})

	t.Run("without a Stripe client a live subscription cannot be cancelled", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		w := billingWorldOn(t, app, "ana@example.com")
		w.svc.Billing = nil

		b, err := billing.New(app.App, w.ws)
		if err != nil {
			t.Fatal(err)
		}

		if b.Stripe != nil {
			t.Fatal("the app has a Stripe client with no STRIPE_KEY")
		}

		if err := b.CancelWorkspaceSubscriptions(t.Context(), w.space.ID); err == nil {
			t.Fatal("cancelling with no Stripe client reported success")
		}
	})
}
