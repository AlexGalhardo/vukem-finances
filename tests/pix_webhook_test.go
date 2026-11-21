package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
)

func pixBody(event, chargeID string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id": "log_1", "event": event, "apiVersion": 2, "devMode": true,
		"data": map[string]any{"transparent": map[string]any{"id": chargeID, "status": "PAID"}},
	})

	return b
}

type pixDelivery struct {
	path      string
	secret    string // X-Webhook-Secret
	signature string // "" signs the body
	body      []byte
}

func sendPix(t *testing.T, app *vukemtest.App, d pixDelivery) *vukemtest.Response {
	t.Helper()

	c := app.Client(t)
	if d.secret != "" {
		c = c.WithHeader("X-Webhook-Secret", d.secret)
	}

	sig := d.signature
	if sig == "" {
		sig = billing.SignForTest(d.body)
	}

	path := d.path
	if path == "" {
		path = billing.PixWebhookPath
	}

	return c.WithHeader("X-Webhook-Signature", sig).Do(http.MethodPost, path, bytes.NewReader(d.body), "application/json")
}

func TestSecurity_PixWebhook(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 1)
	paid := pixBody("transparent.completed", p.ProviderRef)

	// No secret, a wrong secret, a secret of the right start, and the right
	// secret in the query string only: all 401.
	sendPix(t, app, pixDelivery{body: paid}).AssertUnauthorized()
	sendPix(t, app, pixDelivery{body: paid, secret: "wrong"}).AssertUnauthorized()
	sendPix(t, app, pixDelivery{body: paid, secret: abacateSecret[:5]}).AssertUnauthorized()
	sendPix(t, app, pixDelivery{body: paid, path: billing.PixWebhookPath + "?webhookSecret=" + abacateSecret}).
		AssertUnauthorized()
	sendPix(t, app, pixDelivery{body: paid, path: billing.PixWebhookPath + "/wrong"}).AssertUnauthorized()

	// A right secret with a bad or missing signature is refused too.
	sendPix(t, app, pixDelivery{body: paid, secret: abacateSecret, signature: "AAAA"}).AssertUnauthorized()

	// A forged "paid" body for a charge the provider reports pending activates nothing.
	sendPix(t, app, pixDelivery{body: paid, secret: abacateSecret}).AssertOK()
	assertPlan(t, ana, "free")

	if got := pixPayments(t, app)[0].Status; got != "pending" {
		t.Errorf("status = %q after a forged event", got)
	}

	// Once the provider says paid, the same delivery activates, by header or by path segment.
	f.setStatus(p.ProviderRef, "PAID")
	sendPix(t, app, pixDelivery{body: paid, path: billing.PixWebhookPath + "/" + abacateSecret}).AssertOK()
	assertPlan(t, ana, "pro")

	// Unknown events and unknown charges answer 200 and change nothing.
	sendPix(t, app, pixDelivery{body: pixBody("payout.completed", "x"), secret: abacateSecret}).AssertOK()
	sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", "pix_char_unknown"), secret: abacateSecret}).
		AssertOK()

	// A body of the wrong shape, an oversized body and the wrong method.
	sendPix(t, app, pixDelivery{body: []byte(`not json`), secret: abacateSecret}).AssertStatus(http.StatusBadRequest)
	sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", ""), secret: abacateSecret}).
		AssertStatus(http.StatusBadRequest)
	sendPix(t, app, pixDelivery{body: bytes.Repeat([]byte("a"), 70<<10), secret: abacateSecret}).
		AssertStatus(http.StatusRequestEntityTooLarge)
	app.Client(t).WithHeader("X-Webhook-Secret", abacateSecret).Do(http.MethodGet, billing.PixWebhookPath, nil, "").
		AssertStatus(http.StatusMethodNotAllowed)

	// No secret or key in the logs.
	logs := app.Logs()
	for _, secret := range []string{abacateSecret, abacateKey} {
		if strings.Contains(logs, secret) {
			t.Errorf("a secret reached the logs: %q", secret)
		}
	}
}

func TestPixWebhookRefusesWhenNotConfigured(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", "x"), secret: abacateSecret}).
		AssertStatus(http.StatusServiceUnavailable)
}

func TestPixWebhookDoesNotExistInThePersonalEdition(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t)
	sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", "x"), secret: abacateSecret}).AssertNotFound()
}

func TestPixConcurrentWebhookRequests(t *testing.T) {
	t.Parallel()

	clock := newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC))
	f, app := pixApp(t, clock)
	b := pixBilling(t, app)
	ana := signUp(t, app, "Ana", "ana@example.com")
	p := startPix(t, b, ana, 1)
	f.setStatus(p.ProviderRef, "PAID")

	body := pixBody("transparent.completed", p.ProviderRef)

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() { sendPix(t, app, pixDelivery{body: body, secret: abacateSecret}).AssertOK() })
	}

	wg.Wait()

	if ent := pixEntitlementOf(t, app, ana.Workspace.ID); ent.PaidMonths != 1 {
		t.Errorf("paid months = %d after 50 deliveries, want 1", ent.PaidMonths)
	}
}
