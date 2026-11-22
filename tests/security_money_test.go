package tests

import (
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// The money attack suite (epic 24 task 6): amounts the client cannot choose,
// plan limits that hold under concurrency, and sums that cannot overflow.

func TestSecurity_Money_PlanLimitsHoldUnderConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("50 concurrent accounts on the Free plan stay within its limit", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		before := rowCount(t, app, "accounts")

		var wg sync.WaitGroup

		for i := range 50 {
			wg.Go(func() {
				c := browser(t, app, ana.User, ana.Workspace.ID)
				c.PostForm("/accounts", url.Values{"name": {"Extra " + strconv.Itoa(i)}, "kind": {"savings"}})
			})
		}

		wg.Wait()

		if got := rowCount(t, app, "accounts"); got != before {
			t.Fatalf("%d accounts, the Free plan allows %d", got, before)
		}
	})

	t.Run("50 concurrent API writes at the monthly limit stay at the limit", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		token := fullToken(t, app, ana)
		acc := firstOf[models.Account](t, app, ana, "")

		month := time.Now().UTC().Format("2006-01")
		fillMonth(t, app, ana, acc.ID, 90, month)

		var wg sync.WaitGroup

		for range 50 {
			wg.Go(func() {
				apiClient(t, app, token).PostJSON("/api/v1/transactions", map[string]any{
					"account_uid": acc.UID, "kind": "expense", "amount_cents": 100, "date": month + "-16", "description": "race",
				})
			})
		}

		wg.Wait()
		vukemtest.AssertDatabaseCount(t, app, "transactions", 100)
	})

	t.Run("50 concurrent invitations never pass the member limit", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")

		var wg sync.WaitGroup

		for i := range 50 {
			wg.Go(func() {
				c := browser(t, app, ana.User, ana.Workspace.ID)
				c.PostForm("/workspace/invites", url.Values{"email": {"guest" + strconv.Itoa(i) + "@example.com"}, "role": {"member"}})
			})
		}

		wg.Wait()

		if got := rowCount(t, app, "memberships"); got != 1 {
			t.Fatalf("%d memberships on a Free workspace", got)
		}
	})
}

func TestSecurity_Money_ServerOwnsThePrice(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	ana := signUp(t, app, "Ana", "ana@example.com")

	t.Run("a request that names an amount, a currency or a price is charged the server price", func(t *testing.T) {
		res := ana.Client.PostForm("/billing/pix", url.Values{
			"months": {"1"}, "amount": {"1"}, "amountCents": {"1"}, "amount_cents": {"1"}, "price": {"0.01"},
			"currency": {"USD"}, "priceId": {"price_free"}, "discount": {"100"}, "coupon": {"FREE"},
		})
		res.AssertStatus(http.StatusCreated)
		res.AssertJSONPath("amountCents", float64(1990))

		if got := f.created()[0]["amount"]; got != float64(1990) {
			t.Fatalf("the provider was asked to charge %v", got)
		}
	})

	t.Run("a month count outside the options is refused and charges nothing", func(t *testing.T) {
		made := len(f.created())

		for _, months := range []string{"0", "-1", "2", "13", "1.5", "1e1", "abc", ""} {
			if r := ana.Client.PostForm("/billing/pix", url.Values{"months": {months}}); r.Status() != http.StatusUnprocessableEntity {
				t.Errorf("months %q answered %d, want 422", months, r.Status())
			}
		}

		if got := len(f.created()); got != made {
			t.Fatalf("%d charges were made at the provider", got-made)
		}
	})

	t.Run("a paid webhook for a charge the provider reports pending activates nothing", func(t *testing.T) {
		p := pixPayments(t, app)[0]
		sendPix(t, app, pixDelivery{body: pixBody("transparent.completed", p.ProviderRef), secret: abacateSecret}).AssertOK()
		assertPlan(t, ana, "free")
	})
}

func TestSecurity_Money_SumsDoNotOverflow(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	token := fullToken(t, app, ana)
	acc := firstOf[models.Account](t, app, ana, "")

	for i := range 3 {
		apiClient(t, app, token).PostJSON("/api/v1/transactions", map[string]any{
			"account_uid": acc.UID, "kind": "income", "amount_cents": models.MaxCents, "date": "2026-11-0" + strconv.Itoa(i+1), "description": "max",
		}).AssertCreated()
	}

	t.Run("one cent more than the maximum is refused", func(t *testing.T) {
		res := apiClient(t, app, token).PostJSON("/api/v1/transactions", map[string]any{
			"account_uid": acc.UID, "kind": "income", "amount_cents": models.MaxCents + 1, "date": "2026-11-05",
		})
		if res.Status() != http.StatusUnprocessableEntity {
			t.Fatalf("answered %d", res.Status())
		}
	})

	t.Run("pages and reports that add the amounts answer instead of failing", func(t *testing.T) {
		for _, target := range []string{
			"/dashboard", "/reports", "/reports?period=custom&from=2026-11-01&to=2026-11-30", "/accounts", "/transactions", "/budgets",
		} {
			if res := ana.Client.Inertia("").Get(target); res.Status() != http.StatusOK {
				t.Errorf("GET %s answered %d: %.200s", target, res.Status(), res.Body())
			}
		}

		for _, target := range []string{"/api/v1/accounts", "/api/v1/transactions", "/api/v1/reports/cashflow"} {
			if res := apiClient(t, app, token).Get(target); res.Status() >= 500 {
				t.Errorf("GET %s answered %d: %.200s", target, res.Status(), res.Body())
			}
		}
	})

	t.Run("the export writes the amounts as plain decimals", func(t *testing.T) {
		rows := csvRecords(t, ana.Client.Get("/transactions/export.csv").AssertOK().Body())
		if len(rows) != 4 {
			t.Fatalf("%d rows", len(rows))
		}
	})
}
