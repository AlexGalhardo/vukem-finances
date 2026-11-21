package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

const (
	abacateKey    = "abc_dev_test_key_only"
	abacateSecret = "pix-webhook-secret-test"
)

// fakeAbacatePay stands in for api.abacatepay.com: create, check and simulate,
// with the charge states a test sets. No test reaches the network.
type fakeAbacatePay struct {
	t   *testing.T
	srv *httptest.Server
	now func() time.Time

	mu      sync.Mutex
	seq     int
	charges map[string]*fakeCharge
	creates []map[string]any
	checks  int
}

type fakeCharge struct {
	ID        string
	Amount    int64
	Status    string
	ExpiresAt time.Time
}

func newFakeAbacatePay(t *testing.T, now func() time.Time) *fakeAbacatePay {
	t.Helper()

	f := &fakeAbacatePay{t: t, now: now, charges: map[string]*fakeCharge{}}
	mux := http.NewServeMux()

	reply := func(w http.ResponseWriter, c *fakeCharge) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": map[string]any{
			"id": c.ID, "amount": c.Amount, "status": c.Status, "brCode": "00020101021226-" + c.ID,
			"brCodeBase64": "data:image/png;base64,iVBORw0KGgo=", "expiresAt": c.ExpiresAt.UTC().Format(time.RFC3339Nano),
		}})
	}

	mux.HandleFunc("POST /transparents/create", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string         `json:"method"`
			Data   map[string]any `json:"data"`
		}

		_ = json.NewDecoder(r.Body).Decode(&body)

		f.mu.Lock()
		f.seq++
		amount, _ := body.Data["amount"].(float64)
		expires, _ := body.Data["expiresIn"].(float64)
		c := &fakeCharge{
			ID: "pix_char_" + strconv.Itoa(f.seq), Amount: int64(amount), Status: "PENDING",
			ExpiresAt: f.now().Add(time.Duration(expires) * time.Second),
		}
		f.charges[c.ID] = c
		f.creates = append(f.creates, body.Data)
		f.mu.Unlock()

		reply(w, c)
	})
	mux.HandleFunc("GET /transparents/check", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.checks++
		c, ok := f.charges[r.URL.Query().Get("id")]
		snapshot := fakeCharge{}
		if ok {
			snapshot = *c
		}
		f.mu.Unlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		reply(w, &snapshot)
	})
	mux.HandleFunc("POST /transparents/simulate-payment", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		c, ok := f.charges[r.URL.Query().Get("id")]
		if ok && c.Status == "PENDING" {
			c.Status = "PAID"
		}
		f.mu.Unlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		reply(w, c)
	})

	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+abacateKey {
			t.Errorf("%s %s: wrong Authorization", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.srv.Close)

	return f
}

func (f *fakeAbacatePay) setStatus(id, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.charges[id].Status = status
}

func (f *fakeAbacatePay) created() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]map[string]any(nil), f.creates...)
}

func pixEnv(f *fakeAbacatePay) map[string]string {
	return map[string]string{
		"ABACATEPAY_API_KEY": abacateKey, "ABACATEPAY_API_BASE": f.srv.URL, "ABACATEPAY_WEBHOOK_SECRET": abacateSecret,
		"PIX_PRICE_PRO_MONTHLY_CENTS": "1990",
	}
}

// pixApp builds a saas app with PIX on and a clock the test moves.
func pixApp(t *testing.T, clock *vukemtest.FakeClock) (*fakeAbacatePay, *vukemtest.App) {
	t.Helper()

	f := newFakeAbacatePay(t, clock.Now)
	env := pixEnv(f)
	env["ADMIN_EMAILS"], env["APP_EDITION"] = "boss@example.com", "saas"

	return f, vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: env, Clock: clock.Now})
}

func pixBilling(t *testing.T, app *vukemtest.App) *billing.Billing {
	t.Helper()

	b, err := billing.New(app.App, service(t, app))
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func pixPayments(t *testing.T, app *vukemtest.App) []models.PixPayment {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows, err := orm.QueryOf[models.PixPayment](db).OrderBy("id", orm.Asc).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func pixEntitlementOf(t *testing.T, app *vukemtest.App, workspace int64) models.PixEntitlement {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	e, err := orm.QueryOf[models.PixEntitlement](db).Where("workspace_id", workspace).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return e
}
