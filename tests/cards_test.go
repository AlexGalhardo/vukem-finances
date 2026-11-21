package tests

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// cardEnv is a Pro workspace on 2026-11-20 with a checking account holding
// 5000.00 and a card closing on the 10th and due on the 25th, limit 2000.00.
type cardEnv struct {
	app  *vukemtest.App
	ana  who
	fin  *finance.Service
	ctx  context.Context
	bank models.Account
	card finance.CardView
}

func newCardEnv(t *testing.T) cardEnv {
	t.Helper()

	clock := vukemtest.NewFakeClock(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	e := cardEnv{app: app, ana: ana, fin: softService(t, app), ctx: wsCtx(t, ana)}
	e.bank = insertAccount(t, app, ana, "Checking", 500000)

	card, err := e.fin.CreateCard(e.ctx, e.input("Visa", 10))
	if err != nil {
		t.Fatal(err)
	}

	e.card = card

	return e
}

func (e cardEnv) input(name string, closing int) finance.CardInput {
	return finance.CardInput{
		Name: name, Limit: "2000.00", ClosingDay: closing, DueDay: 25, PaymentAccountID: e.bank.ID, Brand: "visa", Last4: "4242",
	}
}

func (e cardEnv) buy(t *testing.T, amount, date string) models.Transaction {
	t.Helper()

	tx, err := e.fin.CreateTransaction(e.ctx, finance.TxInput{
		AccountID: e.card.AccountID, Kind: models.KindExpense, Amount: amount, Description: "buy", Date: date,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tx
}

func (e cardEnv) refOf(t *testing.T, id int64) string {
	t.Helper()

	db, err := e.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := workspaces.Scoped[models.Transaction](e.ctx, db).WithTrashed().Where("id", id).First(e.ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !tx.StatementRef.Valid {
		return ""
	}

	return tx.StatementRef.V
}

func (e cardEnv) statement(t *testing.T, ref string) finance.StatementSummary {
	t.Helper()

	list, err := e.fin.Statements(e.ctx, e.card.ID)
	if err != nil {
		t.Fatal(err)
	}

	for _, st := range list {
		if st.Ref == ref {
			return st
		}
	}

	t.Fatalf("no statement %s in %+v", ref, list)

	return finance.StatementSummary{}
}

func (e cardEnv) pay(t *testing.T, ref, amount string) finance.Transfer {
	t.Helper()

	tr, err := e.fin.PayStatement(e.ctx, e.card.ID, ref, finance.PaymentInput{FromAccountID: e.bank.ID, Amount: amount})
	if err != nil {
		t.Fatal(err)
	}

	return tr
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()

	var he *errs.HTTPError
	if !errors.As(err, &he) || he.Status != status {
		t.Fatalf("err = %v, want HTTP %d", err, status)
	}
}

func TestCards(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)

	t.Run("a card is an account of kind credit_card", func(t *testing.T) {
		if e.card.Name != "Visa" || e.card.ClosingDay != 10 || e.card.DueDay != 25 || e.card.Last4 != "4242" {
			t.Fatalf("card = %+v", e.card)
		}

		vukemtest.AssertDatabaseHas(t, e.app, "accounts", map[string]any{"id": e.card.AccountID, "kind": "credit_card", "opening_cents": 0})
	})

	t.Run("invalid input is refused", func(t *testing.T) {
		for name, mutate := range map[string]func(*finance.CardInput){
			"16-digit last4":   func(in *finance.CardInput) { in.Last4 = "4242424242424242" },
			"three digits":     func(in *finance.CardInput) { in.Last4 = "424" },
			"letters":          func(in *finance.CardInput) { in.Last4 = "42a2" },
			"closing day 0":    func(in *finance.CardInput) { in.ClosingDay = 0 },
			"closing day 32":   func(in *finance.CardInput) { in.ClosingDay = 32 },
			"due day 0":        func(in *finance.CardInput) { in.DueDay = 0 },
			"no limit":         func(in *finance.CardInput) { in.Limit = "" },
			"negative limit":   func(in *finance.CardInput) { in.Limit = "-5" },
			"unknown brand":    func(in *finance.CardInput) { in.Brand = "cvv" },
			"empty name":       func(in *finance.CardInput) { in.Name = " " },
			"card as payer":    func(in *finance.CardInput) { in.PaymentAccountID = e.card.AccountID },
			"foreign payer id": func(in *finance.CardInput) { in.PaymentAccountID = 99999 },
		} {
			in := e.input("Other", 5)
			mutate(&in)

			if _, err := e.fin.CreateCard(e.ctx, in); err == nil {
				t.Errorf("%s: accepted", name)
			} else {
				wantStatus(t, err, http.StatusUnprocessableEntity)
			}
		}

		vukemtest.AssertDatabaseCount(t, e.app, "credit_cards", 1)
	})

	t.Run("update changes the card and the name of its account", func(t *testing.T) {
		in := e.input("Visa Gold", 10)
		in.Limit = "3000.00"
		in.PaymentAccountID = 0

		got, err := e.fin.UpdateCard(e.ctx, e.card.ID, in)
		if err != nil || got.Name != "Visa Gold" || got.LimitCents != 300000 || got.PaymentAccountID.Valid {
			t.Fatalf("updated = %+v, %v", got, err)
		}
	})

	t.Run("the account of a card cannot be edited as a plain account", func(t *testing.T) {
		_, err := e.fin.UpdateAccount(e.ctx, e.card.AccountID, "Renamed", models.AccountChecking, "")
		wantStatus(t, err, http.StatusUnprocessableEntity)

		_, err = e.fin.CreateAccount(e.ctx, "Fake", models.AccountCreditCard, "")
		wantStatus(t, err, http.StatusUnprocessableEntity)
	})

	t.Run("archive, unarchive, delete and restore", func(t *testing.T) {
		got, err := e.fin.ArchiveCard(e.ctx, e.card.ID)
		if err != nil || !got.Archived {
			t.Fatalf("archived = %+v, %v", got, err)
		}

		if got, err = e.fin.UnarchiveCard(e.ctx, e.card.ID); err != nil || got.Archived {
			t.Fatalf("unarchived = %+v, %v", got, err)
		}

		buy := e.buy(t, "10.00", "2026-11-05")

		if err := e.fin.DeleteCard(e.ctx, e.card.ID); err != nil {
			t.Fatal(err)
		}

		cards, err := e.fin.Cards(e.ctx)
		if err != nil || len(cards) != 0 {
			t.Fatalf("cards after delete = %+v, %v", cards, err)
		}

		if _, err := e.fin.Card(e.ctx, e.card.ID); err == nil {
			t.Fatal("a deleted card was found")
		}

		if _, err := e.fin.RestoreAccount(e.ctx, e.card.AccountID); err != nil {
			t.Fatal(err)
		}

		cards, err = e.fin.Cards(e.ctx)
		if err != nil || len(cards) != 1 || cards[0].ID != e.card.ID {
			t.Fatalf("cards after restore = %+v, %v", cards, err)
		}

		if e.refOf(t, buy.ID) != "2026-11-10" {
			t.Errorf("the purchase lost its statement")
		}
	})

	t.Run("deleting the payment account clears the default", func(t *testing.T) {
		pay := insertAccount(t, e.app, e.ana, "Payer", 0)
		in := e.input("Visa Gold", 10)
		in.PaymentAccountID = pay.ID

		if _, err := e.fin.UpdateCard(e.ctx, e.card.ID, in); err != nil {
			t.Fatal(err)
		}

		if err := e.fin.DeleteAccount(e.ctx, pay.ID); err != nil {
			t.Fatal(err)
		}

		got, err := e.fin.Card(e.ctx, e.card.ID)
		if err != nil || got.PaymentAccountID.Valid {
			t.Fatalf("card = %+v, %v; want no payment account", got.PaymentAccountID, err)
		}
	})
}

func TestStatementRefOnWrite(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)

	for date, want := range map[string]string{
		"2026-11-10": "2026-11-10",
		"2026-11-11": "2026-12-10",
		"2026-12-31": "2027-01-10",
		"2026-11-01": "2026-11-10",
	} {
		if got := e.refOf(t, e.buy(t, "1.00", date).ID); got != want {
			t.Errorf("purchase on %s is on statement %q, want %s", date, got, want)
		}
	}

	t.Run("an edit moves the purchase", func(t *testing.T) {
		tx := e.buy(t, "5.00", "2026-11-05")

		moved, err := e.fin.UpdateTransaction(e.ctx, tx.ID, finance.TxInput{
			AccountID: e.card.AccountID, Kind: models.KindExpense, Amount: "5.00", Date: "2026-11-15",
		})
		if err != nil || moved.StatementRef.V != "2026-12-10" {
			t.Fatalf("edited = %+v, %v", moved.StatementRef, err)
		}

		elsewhere, err := e.fin.UpdateTransaction(e.ctx, tx.ID, finance.TxInput{
			AccountID: e.bank.ID, Kind: models.KindExpense, Amount: "5.00", Date: "2026-11-15",
		})
		if err != nil || elsewhere.StatementRef.Valid {
			t.Fatalf("on the bank account = %+v, %v; want no statement", elsewhere.StatementRef, err)
		}
	})

	t.Run("an ordinary account has no statement", func(t *testing.T) {
		tx, err := e.fin.CreateTransaction(e.ctx, finance.TxInput{
			AccountID: e.bank.ID, Kind: models.KindExpense, Amount: "5.00", Date: "2026-11-15",
		})
		if err != nil || tx.StatementRef.Valid {
			t.Fatalf("created = %+v, %v", tx.StatementRef, err)
		}
	})

	t.Run("an import into the card account assigns statements", func(t *testing.T) {
		csv := "Date,Description,Amount\n2026-10-09,Shop A,-10.00\n2026-10-11,Shop B,-20.00\n"
		runImport(t, e.app, e.ana, e.card.AccountID, "card.csv", []byte(csv), csvMapping)

		vukemtest.AssertDatabaseHas(t, e.app, "transactions", map[string]any{"description": "Shop A", "statement_ref": "2026-10-10"})
		vukemtest.AssertDatabaseHas(t, e.app, "transactions", map[string]any{"description": "Shop B", "statement_ref": "2026-11-10"})
	})
}

func TestClosingDayChangeRecomputesOpenOnly(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)

	paid := e.buy(t, "100.00", "2026-10-05")  // statement 2026-10-10
	closed := e.buy(t, "50.00", "2026-11-05") // statement 2026-11-10, closed, unpaid
	open := e.buy(t, "30.00", "2026-11-15")   // statement 2026-12-10, open
	trashed := e.buy(t, "7.00", "2026-11-12") // statement 2026-12-10, in the trash

	if err := e.fin.DeleteTransaction(e.ctx, trashed.ID); err != nil {
		t.Fatal(err)
	}

	e.pay(t, "2026-10-10", "")

	if st := e.statement(t, "2026-10-10"); st.Status != finance.StatementPaid {
		t.Fatalf("October = %+v, want paid", st)
	}

	// A partial payment of the closed November statement moves with it.
	e.pay(t, "2026-11-10", "20.00")

	if _, err := e.fin.UpdateCard(e.ctx, e.card.ID, e.input("Visa", 20)); err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct {
		id   int64
		want string
	}{
		"paid":    {paid.ID, "2026-10-10"},
		"closed":  {closed.ID, "2026-11-20"},
		"open":    {open.ID, "2026-11-20"},
		"trashed": {trashed.ID, "2026-11-20"},
	} {
		if got := e.refOf(t, c.id); got != c.want {
			t.Errorf("%s purchase is on %q, want %s", name, got, c.want)
		}
	}

	st := e.statement(t, "2026-11-20")
	if st.PurchasesCents != 8000 || st.PaidCents != 2000 || st.RemainingCents != 6000 {
		t.Errorf("statement 2026-11-20 = %+v, want 80.00 bought, 20.00 paid", st)
	}

	if st := e.statement(t, "2026-10-10"); st.Status != finance.StatementPaid || st.PurchasesCents != 10000 {
		t.Errorf("the paid statement changed: %+v", st)
	}
}

func TestClosingDayChangeWithManyRows(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	db, err := e.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	err = func() error {
		for i := range 5000 {
			d := start.AddDate(0, 0, i%120).Format("2006-01-02")
			tx := models.Transaction{
				AccountID: e.card.AccountID, Kind: models.KindExpense, AmountCents: 100, Description: "bulk", OccurredOn: d,
				StatementRef: sql.Null[string]{V: "2026-08-10", Valid: true},
			}

			if err := orm.Create(e.ctx, db, &tx); err != nil {
				return err
			}
		}

		return nil
	}()
	if err != nil {
		t.Fatal(err)
	}

	began := time.Now()

	if _, err := e.fin.UpdateCard(e.ctx, e.card.ID, e.input("Visa", 28)); err != nil {
		t.Fatal(err)
	}

	if time.Since(began) > 20*time.Second {
		t.Errorf("recomputing 5000 rows took %s", time.Since(began))
	}

	vukemtest.AssertDatabaseHas(t, e.app, "transactions", map[string]any{"occurred_on": "2026-07-29", "statement_ref": "2026-08-28"})
	vukemtest.AssertDatabaseHas(t, e.app, "transactions", map[string]any{"occurred_on": "2026-07-28", "statement_ref": "2026-07-28"})
}

func TestStatementStatusAndPayStatement(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	e.buy(t, "300.00", "2026-11-05")
	e.buy(t, "40.00", "2026-11-15")

	closed := e.statement(t, "2026-11-10")
	if closed.Status != finance.StatementClosed || closed.TotalCents != 30000 || closed.DueOn != "2026-11-25" {
		t.Fatalf("closed = %+v", closed)
	}

	if open := e.statement(t, "2026-12-10"); open.Status != finance.StatementOpen || open.TotalCents != 4000 {
		t.Fatalf("open = %+v", open)
	}

	t.Run("a partial payment leaves the rest", func(t *testing.T) {
		e.pay(t, "2026-11-10", "100.00")

		if st := e.statement(t, "2026-11-10"); st.PaidCents != 10000 || st.RemainingCents != 20000 || st.Status != finance.StatementClosed {
			t.Fatalf("after partial = %+v", st)
		}
	})

	t.Run("the default pays the rest", func(t *testing.T) {
		tr := e.pay(t, "2026-11-10", "")
		if tr.To.AmountCents != 20000 || tr.To.AccountID != e.card.AccountID || tr.From.AccountID != e.bank.ID {
			t.Fatalf("payment = %+v", tr)
		}

		if tr.To.StatementRef.V != "2026-11-10" || tr.From.StatementRef.Valid {
			t.Errorf("only the card leg carries the statement: %+v / %+v", tr.To.StatementRef, tr.From.StatementRef)
		}

		if st := e.statement(t, "2026-11-10"); st.Status != finance.StatementPaid || st.RemainingCents != 0 {
			t.Fatalf("after exact = %+v", st)
		}
	})

	t.Run("nothing left to pay by default", func(t *testing.T) {
		_, err := e.fin.PayStatement(e.ctx, e.card.ID, "2026-11-10", finance.PaymentInput{FromAccountID: e.bank.ID})
		wantStatus(t, err, http.StatusUnprocessableEntity)
	})

	t.Run("an overpayment is credit on the card", func(t *testing.T) {
		e.pay(t, "2026-11-10", "10.00")

		ov, err := e.fin.CardOverviews(e.ctx)
		if err != nil || len(ov) != 1 {
			t.Fatal(ov, err)
		}

		// bought 340.00, paid 310.00
		if ov[0].BalanceCents != -3000 || ov[0].UsedCents != 3000 || ov[0].AvailableCents != 197000 || ov[0].OverLimit {
			t.Fatalf("overview = %+v", ov[0])
		}
	})

	t.Run("refusals", func(t *testing.T) {
		in := finance.PaymentInput{FromAccountID: e.bank.ID, Amount: "1.00"}

		_, err := e.fin.PayStatement(e.ctx, e.card.ID, "2026-02-30", in)
		wantStatus(t, err, http.StatusUnprocessableEntity)

		_, err = e.fin.PayStatement(e.ctx, e.card.ID, "2026-11-11", in)
		wantStatus(t, err, http.StatusUnprocessableEntity)

		_, err = e.fin.PayStatement(e.ctx, e.card.ID, "2026-11-10", finance.PaymentInput{FromAccountID: e.card.AccountID, Amount: "1.00"})
		wantStatus(t, err, http.StatusUnprocessableEntity)

		_, err = e.fin.PayStatement(e.ctx, e.card.ID, "2026-11-10", finance.PaymentInput{FromAccountID: e.bank.ID, Amount: "1,0,0"})
		wantStatus(t, err, http.StatusUnprocessableEntity)

		if _, err = e.fin.ArchiveAccount(e.ctx, e.bank.ID); err != nil {
			t.Fatal(err)
		}

		_, err = e.fin.PayStatement(e.ctx, e.card.ID, "2026-11-10", in)
		wantStatus(t, err, http.StatusUnprocessableEntity)
	})
}

func TestOverdueAndOverLimit(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	e.buy(t, "2500.00", "2026-10-05")

	st := e.statement(t, "2026-10-10")
	if st.Status != finance.StatementOverdue {
		t.Fatalf("statement = %+v, want overdue", st)
	}

	ov, err := e.fin.CardOverviews(e.ctx)
	if err != nil || len(ov) != 1 || !ov[0].OverLimit || ov[0].Due == nil || ov[0].Due.Ref != "2026-10-10" {
		t.Fatalf("overview = %+v, %v", ov, err)
	}
}

// Purchases count on their date and a payment is a transfer, so the expense
// of a month is the same before and after the statement is paid.
func TestCardPaymentIsNotExpense(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	e.buy(t, "300.00", "2026-11-05")

	expense := func() int64 {
		t.Helper()

		d, err := e.fin.Dashboard(e.ctx, "2026-11")
		if err != nil {
			t.Fatal(err)
		}

		for _, m := range d.Months {
			if m.Month == "2026-11" {
				return m.ExpenseCents
			}
		}

		t.Fatal("no November in the dashboard")

		return 0
	}

	if got := expense(); got != 30000 {
		t.Fatalf("expense before the payment = %d, want 30000", got)
	}

	checkingBefore, cardBefore := e.balances(t)

	e.pay(t, "2026-11-10", "")

	if got := expense(); got != 30000 {
		t.Errorf("expense after the payment = %d, want 30000: the payment is not an expense", got)
	}

	checking, card := e.balances(t)
	if checkingBefore-checking != 30000 || card-cardBefore != 30000 || card != 0 {
		t.Errorf("checking %d -> %d, card %d -> %d; want -300.00 and +300.00", checkingBefore, checking, cardBefore, card)
	}
}

func (e cardEnv) balances(t *testing.T) (checking, card int64) {
	t.Helper()

	bs, err := e.fin.AccountBalances(e.ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	return balanceOf(t, bs, e.bank.ID).BalanceCents, balanceOf(t, bs, e.card.AccountID).BalanceCents
}

func TestNetWorthIncludesCards(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	e.buy(t, "300.00", "2026-11-05")

	net, err := e.fin.NetWorth(e.ctx, "")
	if err != nil || net != 500000-30000 {
		t.Fatalf("net worth = %d, %v; want the debt taken off 5000.00", net, err)
	}

	e.pay(t, "2026-11-10", "")

	if net, err = e.fin.NetWorth(e.ctx, ""); err != nil || net != 500000-30000 {
		t.Fatalf("net worth after paying = %d, %v; paying a card does not change it", net, err)
	}
}

// A purchase dated ahead (an installment) still takes limit.
func TestCardLimitCountsFuturePurchases(t *testing.T) {
	t.Parallel()

	e := newCardEnv(t)
	e.buy(t, "100.00", "2026-12-15")

	ov, err := e.fin.CardOverviews(e.ctx)
	if err != nil || len(ov) != 1 || ov[0].UsedCents != 10000 {
		t.Fatalf("overview = %+v, %v; want 100.00 used", ov, err)
	}
}

func TestCardsStoreNoSensitiveData(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`(?i)\b(cvv|cvc|card_number|card_expiry|card_exp|pan)\b`)

	for _, dir := range []string{"../app/models", "../database/migrations"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			if m := re.Find(b); m != nil {
				t.Errorf("%s mentions %q", f, m)
			}
		}
	}
}
