package tests

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/contrib/stripe"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
)

// The webhook attack suite (epic 24 task 6): forged, tampered, stale and
// replayed deliveries, and concurrent ones. The PIX and Telegram webhooks are
// attacked in TestSecurity_PixWebhook and TestSecurity_TelegramWebhook; this
// suite adds Stripe and Resend and the cross-cutting checks.

func TestSecurity_Webhook_StripeForgery(t *testing.T) {
	t.Parallel()

	f := newFakeStripe(t)
	app := billingApp(t, f)
	ana := signUp(t, app, "Ana", "ana@example.com")
	ana.Client.PostForm("/billing/checkout/pro", nil).AssertStatus(http.StatusSeeOther)
	f.setSubscription(testCustomer, "active")

	payload := []byte(`{"id":"evt_forged","type":"customer.subscription.created","created":` + itoa(time.Now().Unix()) +
		`,"data":{"object":{"id":"sub_1","customer":"cus_1"}}}`)
	other := []byte(`{"id":"evt_other","type":"customer.subscription.created","data":{"object":{"id":"sub_9","customer":"cus_9"}}}`)

	now := time.Now()

	post := func(header string, body []byte) *vukemtest.Response {
		c := app.Client(t)
		if header != "" {
			c = c.WithHeader("Stripe-Signature", header)
		}

		return c.Do(http.MethodPost, billing.WebhookPath, bytes.NewReader(body), "application/json")
	}

	cases := map[string]string{
		"no signature":                "",
		"garbage":                     "garbage",
		"a bare timestamp":            "t=1",
		"a bare v1":                   "v1=00",
		"an empty v1":                 "t=" + itoa(now.Unix()) + ",v1=",
		"a zeroed v1":                 "t=" + itoa(now.Unix()) + ",v1=" + strings.Repeat("0", 64),
		"a signature of another body": stripe.SignForTest(other, stripeSecret, now),
		"another secret":              stripe.SignForTest(payload, "whsec_other_"+"secret", now),
		"an empty secret":             stripe.SignForTest(payload, "", now),
		"ten minutes old":             stripe.SignForTest(payload, stripeSecret, now.Add(-10*time.Minute)),
		"ten minutes ahead":           stripe.SignForTest(payload, stripeSecret, now.Add(10*time.Minute)),
		"a year old":                  stripe.SignForTest(payload, stripeSecret, now.Add(-365*24*time.Hour)),
		"the test scheme only":        strings.Replace(stripe.SignForTest(payload, stripeSecret, now), "v1=", "v0=", 1),
	}

	for name, header := range cases {
		t.Run("refused: "+name, func(t *testing.T) {
			res := post(header, payload)
			if res.Status() < 400 || res.Status() >= 500 {
				t.Fatalf("answered %d", res.Status())
			}

			vukemtest.AssertDatabaseCount(t, app, "workspace_subscriptions", 0)
		})
	}

	t.Run("a good signature on a body changed afterwards is refused", func(t *testing.T) {
		header := stripe.SignForTest(payload, stripeSecret, now)

		tampered := bytes.Replace(payload, []byte("sub_1"), []byte("sub_2"), 1)
		if res := post(header, tampered); res.Status() < 400 {
			t.Fatalf("answered %d", res.Status())
		}

		vukemtest.AssertDatabaseCount(t, app, "workspace_subscriptions", 0)
	})

	t.Run("only POST", func(t *testing.T) {
		for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			if res := app.Client(t).Do(m, billing.WebhookPath, nil, ""); res.Status() != http.StatusMethodNotAllowed && res.Status() != http.StatusNotFound {
				t.Errorf("%s answered %d", m, res.Status())
			}
		}
	})

	t.Run("a body over the cap is refused", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), 2<<20)
		if res := post(stripe.SignForTest(big, stripeSecret, now), big); res.Status() < 400 || res.Status() >= 500 {
			t.Fatalf("answered %d", res.Status())
		}
	})

	t.Run("a forged call changes no plan and no secret reaches the logs", func(t *testing.T) {
		assertPlan(t, ana, "free")

		for _, secret := range []string{stripeSecret, stripeKey} {
			if strings.Contains(app.Logs(), secret) {
				t.Errorf("%q reached the logs", secret)
			}
		}
	})
}

func TestSecurity_Webhook_StripeReplayAndConcurrency(t *testing.T) {
	t.Parallel()

	f := newFakeStripe(t)
	app := billingApp(t, f)
	ana := signUp(t, app, "Ana", "ana@example.com")
	ana.Client.PostForm("/billing/checkout/pro", nil).AssertStatus(http.StatusSeeOther)
	f.setSubscription(testCustomer, "active")

	object := map[string]any{"id": testSubscript, "customer": testCustomer}

	t.Run("50 concurrent deliveries of one event make one subscription", func(t *testing.T) {
		var wg sync.WaitGroup

		statuses := make([]int, 50)

		for i := range statuses {
			wg.Go(func() { statuses[i] = postStripe(t, app, "evt_same", "customer.subscription.created", object).Status() })
		}

		wg.Wait()

		for i, s := range statuses {
			if s >= 500 {
				t.Errorf("delivery %d answered %d", i, s)
			}
		}

		vukemtest.AssertDatabaseCount(t, app, "workspace_subscriptions", 1)
		assertPlan(t, ana, "pro")
	})

	t.Run("50 concurrent events about the same subscription still make one", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := range 50 {
			wg.Go(func() {
				if s := postStripe(t, app, "evt_many_"+itoa(int64(i)), "customer.subscription.updated", object).Status(); s >= 500 {
					t.Errorf("event %d answered %d", i, s)
				}
			})
		}

		wg.Wait()
		vukemtest.AssertDatabaseCount(t, app, "workspace_subscriptions", 1)
	})

	t.Run("a replay of a created event never revives a cancelled subscription", func(t *testing.T) {
		f.setSubscription(testCustomer, "canceled")
		postStripe(t, app, "evt_cancel", "customer.subscription.deleted", object).AssertOK()
		assertPlan(t, ana, "free")

		postStripe(t, app, "evt_same", "customer.subscription.created", object).AssertOK()
		postStripe(t, app, "evt_late", "customer.subscription.created", object).AssertOK()
		assertPlan(t, ana, "free")
	})
}

func TestSecurity_Webhook_ResendForgery(t *testing.T) {
	t.Parallel()

	secret := "whsec_" + "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
	app := newApp(t, map[string]string{"RESEND_WEBHOOK_SECRET": secret})

	bounce := `{"type":"email.bounced","created_at":"2026-11-01T10:00:00Z","data":{"email_id":"e1","to":["victim@example.com"],"bounce":{"type":"Permanent","message":"x"}}}`

	post := func(headers map[string]string, body string) *vukemtest.Response {
		c := app.Client(t)
		for k, v := range headers {
			c = c.WithHeader(k, v)
		}

		return c.Do(http.MethodPost, appmail.ResendWebhookPath, strings.NewReader(body), "application/json")
	}

	now := time.Now().Unix()

	for name, headers := range map[string]map[string]string{
		"no headers":   {},
		"empty":        {"svix-id": "", "svix-timestamp": "", "svix-signature": ""},
		"bad":          {"svix-id": "m1", "svix-timestamp": itoa(now), "svix-signature": "v1,AAAA"},
		"stale":        {"svix-id": "m2", "svix-timestamp": itoa(now - 3600), "svix-signature": "v1,AAAA"},
		"a future one": {"svix-id": "m3", "svix-timestamp": itoa(now + 3600), "svix-signature": "v1,AAAA"},
		"not a number": {"svix-id": "m4", "svix-timestamp": "yesterday", "svix-signature": "v1,AAAA"},
	} {
		t.Run("refused: "+name, func(t *testing.T) {
			if res := post(headers, bounce); res.Status() < 400 || res.Status() >= 500 {
				t.Fatalf("answered %d", res.Status())
			}
		})
	}

	t.Run("a forged bounce suppresses nobody", func(t *testing.T) {
		vukemtest.AssertDatabaseCount(t, app, "email_suppressions", 0)
	})
}

func TestSecurity_Webhook_AbsentInThePersonalEdition(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t)

	for _, path := range []string{billing.WebhookPath, billing.PixWebhookPath, billing.PixWebhookPath + "/anything", "/admin/revenue"} {
		for _, method := range []string{http.MethodPost, http.MethodGet} {
			t.Run(method+" "+path, func(t *testing.T) {
				res := app.Client(t).Do(method, path, strings.NewReader("{}"), "application/json")
				if res.Status() != http.StatusNotFound && (path != "/admin/revenue" || res.Status() != http.StatusMethodNotAllowed) {
					t.Errorf("answered %d, want 404", res.Status())
				}
			})
		}
	}
}

func TestSecurity_Webhook_ConcurrentPixDeliveriesActivateOnce(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)

	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	pa, pb := startPix(t, b, ana, 1), startPix(t, b, bob, 3)
	f.setStatus(pa.ProviderRef, "PAID")
	f.setStatus(pb.ProviderRef, "PAID")

	var wg sync.WaitGroup

	for range 25 {
		wg.Go(func() {
			sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", pa.ProviderRef), secret: abacateSecret})
		})
		wg.Go(func() {
			sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", pb.ProviderRef), secret: abacateSecret})
		})
	}

	wg.Wait()

	if got := pixEntitlementOf(t, app, ana.Workspace.ID).PaidMonths; got != 1 {
		t.Errorf("Ana paid months = %d, want 1", got)
	}

	if got := pixEntitlementOf(t, app, bob.Workspace.ID).PaidMonths; got != 3 {
		t.Errorf("Bob paid months = %d, want 3 (not mixed with another workspace)", got)
	}
}
