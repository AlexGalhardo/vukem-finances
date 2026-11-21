package tests

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func TestCardsHTTP(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	vera := viewerOf(t, app, ana)
	bank := firstAccount(t, app, ana)

	form := func(last4 string) url.Values {
		return url.Values{
			"name": {"Visa"}, "limit": {"2000.00"}, "closingDay": {"10"}, "dueDay": {"25"}, "brand": {"visa"}, "last4": {last4},
			"paymentAccountId": {strconv.FormatInt(bank, 10)},
		}
	}

	t.Run("a card number is refused", func(t *testing.T) {
		ana.Client.PostForm("/cards", form("4242424242424242")).AssertRedirect("/cards")
		vukemtest.AssertDatabaseCount(t, app, "credit_cards", 0)
	})

	t.Run("a viewer cannot add a card", func(t *testing.T) {
		vera.Client.PostForm("/cards", form("4242")).AssertForbidden()
		vukemtest.AssertDatabaseCount(t, app, "credit_cards", 0)
	})

	ana.Client.PostForm("/cards", form("4242")).AssertRedirect("/cards")
	vukemtest.AssertDatabaseCount(t, app, "credit_cards", 1)

	card := firstOf[models.CreditCard](t, app, ana, "")
	base := "/cards/" + strconv.FormatInt(card.ID, 10)

	addTransaction(t, ana, card.AccountID, 0, "expense", "300.00", "2026-11-05").AssertRedirect("/transactions")

	t.Run("the list shows the limit and the statements", func(t *testing.T) {
		page := ana.Client.Inertia("").Get("/cards").AssertInertia("Cards")
		page.PropEquals("cards.0.name", "Visa")
		page.PropEquals("cards.0.usedCents", float64(30000))
		page.PropEquals("cards.0.availableCents", float64(170000))
		page.PropEquals("cards.0.due.ref", "2026-11-10")
		page.PropEquals("cards.0.due.status", "closed")

		detail := ana.Client.Inertia("").Get(base + "?ref=2026-11-10").AssertInertia("CardStatement")
		detail.PropEquals("statement.totalCents", float64(30000))
		detail.PropEquals("statement.rows.0.amount_cents", float64(30000))

		ana.Client.Get(base + "?ref=2026-11-11").AssertNotFound()
	})

	t.Run("a viewer cannot pay", func(t *testing.T) {
		vera.Client.PostForm(base+"/statements/2026-11-10/pay", url.Values{"fromAccountId": {strconv.FormatInt(bank, 10)}}).AssertForbidden()
		vukemtest.AssertDatabaseCount(t, app, "transactions", 1)
	})

	t.Run("paying makes the statement paid", func(t *testing.T) {
		ana.Client.PostForm(base+"/statements/2026-11-10/pay", url.Values{"fromAccountId": {strconv.FormatInt(bank, 10)}}).
			AssertRedirect(base + "?ref=2026-11-10")

		ana.Client.Inertia("").Get(base+"?ref=2026-11-10").AssertInertia("CardStatement").
			PropEquals("statement.status", "paid")
		vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{
			"account_id": card.AccountID, "kind": "income", "amount_cents": 30000, "statement_ref": "2026-11-10",
		})
	})

	t.Run("archive and delete", func(t *testing.T) {
		ana.Client.PostForm(base+"/archive", nil).AssertRedirect("/cards")
		ana.Client.PostForm(base+"/unarchive", nil).AssertRedirect("/cards")
		ana.Client.Delete(base).AssertRedirect("/cards")
		ana.Client.Get(base).AssertNotFound()
	})
}
