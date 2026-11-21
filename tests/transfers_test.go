package tests

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// transferEnv is a workspace with two accounts opened at 1000.00 and 500.00.
type transferEnv struct {
	app  *vukemtest.App
	ana  who
	fin  *finance.Service
	ctx  context.Context
	a, b models.Account
}

func newTransferEnv(t *testing.T) transferEnv {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	return transferEnv{
		app: app, ana: ana, fin: softService(t, app), ctx: workspaces.WithTenant(t.Context(), ana.Workspace.ID),
		a: insertAccount(t, app, ana, "Checking", 100000), b: insertAccount(t, app, ana, "Savings", 50000),
	}
}

func (e transferEnv) balance(t *testing.T, account int64) int64 {
	t.Helper()

	bs, err := e.fin.AccountBalances(e.ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	return balanceOf(t, bs, account).BalanceCents
}

func (e transferEnv) legs(t *testing.T, uid string, trashed bool) []models.Transaction {
	t.Helper()

	db, err := e.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	q := workspaces.Scoped[models.Transaction](e.ctx, db).Where("transfer_uid", uid)
	if trashed {
		q = q.OnlyTrashed()
	}

	got, err := q.Get(e.ctx)
	if err != nil {
		t.Fatal(err)
	}

	return got
}

func (e transferEnv) move(t *testing.T, from, to int64, amount string) finance.Transfer {
	t.Helper()

	tr, err := e.fin.CreateTransfer(e.ctx, finance.TransferInput{
		FromAccountID: from, ToAccountID: to, Amount: amount, Description: "move", Date: daysAgo(1),
	})
	if err != nil {
		t.Fatal(err)
	}

	return tr
}

func TestTransfer(t *testing.T) {
	t.Parallel()

	t.Run("two legs share a transfer uid and move the balances", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		tr := e.move(t, e.a.ID, e.b.ID, "100.00")

		if tr.UID == "" || tr.From.TransferUID.V != tr.UID || tr.To.TransferUID.V != tr.UID {
			t.Fatalf("legs = %+v, want both stamped with the transfer uid", tr)
		}

		if tr.From.Kind != models.KindExpense || tr.From.AccountID != e.a.ID || tr.To.Kind != models.KindIncome || tr.To.AccountID != e.b.ID {
			t.Errorf("legs = %+v, want an expense on the source and an income on the destination", tr)
		}

		if tr.From.AmountCents != 10000 || tr.To.AmountCents != 10000 || tr.From.CategoryID.Valid || tr.To.CategoryID.Valid {
			t.Errorf("legs = %+v, want equal amounts and no category", tr)
		}

		if got := e.balance(t, e.a.ID); got != 90000 {
			t.Errorf("source balance = %d, want 90000", got)
		}

		if got := e.balance(t, e.b.ID); got != 60000 {
			t.Errorf("destination balance = %d, want 60000", got)
		}

		rows, err := e.fin.Transactions(e.ctx, finance.TxFilter{AccountID: e.b.ID})
		if err != nil || len(rows) != 1 || rows[0].TransferID == nil || rows[0].TransferAccountName != "Checking" {
			t.Errorf("list = %+v, %v; want the leg marked as a transfer naming the other account", rows, err)
		}
	})

	t.Run("refusals", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		bob := signUp(t, e.app, "Bob", "bob@example.com")
		foreign := insertAccount(t, e.app, bob, "Bob's", 0)

		if _, err := e.fin.ArchiveAccount(e.ctx, e.b.ID); err != nil {
			t.Fatal(err)
		}

		open := insertAccount(t, e.app, e.ana, "Open", 0)

		for name, tc := range map[string]struct {
			in     finance.TransferInput
			status int
		}{
			"same account":        {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: e.a.ID, Amount: "1", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"archived target":     {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "1", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"archived source":     {finance.TransferInput{FromAccountID: e.b.ID, ToAccountID: e.a.ID, Amount: "1", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"foreign target":      {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: foreign.ID, Amount: "1", Date: daysAgo(0)}, http.StatusNotFound},
			"foreign source":      {finance.TransferInput{FromAccountID: foreign.ID, ToAccountID: e.a.ID, Amount: "1", Date: daysAgo(0)}, http.StatusNotFound},
			"unknown account":     {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: 999999, Amount: "1", Date: daysAgo(0)}, http.StatusNotFound},
			"zero amount":         {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: open.ID, Amount: "0", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"negative amount":     {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: open.ID, Amount: "-5", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"amount over the cap": {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: open.ID, Amount: "90071992547409.92", Date: daysAgo(0)}, http.StatusUnprocessableEntity},
			"bad date":            {finance.TransferInput{FromAccountID: e.a.ID, ToAccountID: open.ID, Amount: "1", Date: "soon"}, http.StatusUnprocessableEntity},
		} {
			if _, err := e.fin.CreateTransfer(e.ctx, tc.in); httpStatus(err) != tc.status {
				t.Errorf("%s: err = %v, want status %d", name, err, tc.status)
			}
		}

		if n := count[models.Transaction](t, e.app, e.ana); n != 0 {
			t.Errorf("refused transfers left %d transactions", n)
		}

		// The largest amount the cap allows is accepted and exact.
		tr := e.move(t, e.a.ID, open.ID, "90071992547409.91")
		if tr.From.AmountCents != models.MaxCents || tr.To.AmountCents != models.MaxCents {
			t.Errorf("max transfer = %d / %d, want %d", tr.From.AmountCents, tr.To.AmountCents, models.MaxCents)
		}
	})

	t.Run("editing one leg changes both", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		third := insertAccount(t, e.app, e.ana, "Cash", 0)
		tr := e.move(t, e.a.ID, e.b.ID, "100.00")

		got, err := e.fin.UpdateTransfer(e.ctx, tr.To.ID, finance.TransferInput{
			FromAccountID: e.a.ID, ToAccountID: third.ID, Amount: "25.50", Description: "edited", Date: daysAgo(2),
		})
		if err != nil {
			t.Fatal(err)
		}

		for _, leg := range []models.Transaction{got.From, got.To} {
			if leg.AmountCents != 2550 || leg.Description != "edited" || leg.OccurredOn != daysAgo(2) || leg.Version < 2 {
				t.Errorf("leg = %+v, want amount 2550, the new text and date, version bumped", leg)
			}
		}

		if got.UID != tr.UID || got.To.AccountID != third.ID || got.From.AccountID != e.a.ID {
			t.Errorf("transfer = %+v, want the same uid and the new destination", got)
		}

		if e.balance(t, e.a.ID) != 97450 || e.balance(t, e.b.ID) != 50000 || e.balance(t, third.ID) != 2550 {
			t.Errorf("balances = %d, %d, %d", e.balance(t, e.a.ID), e.balance(t, e.b.ID), e.balance(t, third.ID))
		}

		// An ordinary edit of a leg is refused: it would unbalance the pair.
		if _, err := e.fin.UpdateTransaction(e.ctx, tr.From.ID, finance.TxInput{
			AccountID: e.a.ID, Kind: models.KindExpense, Amount: "1.00", Date: daysAgo(1),
		}); httpStatus(err) != http.StatusUnprocessableEntity {
			t.Errorf("UpdateTransaction of a leg = %v, want 422", err)
		}

		if _, err := e.fin.UpdateTransfer(e.ctx, tr.From.ID, finance.TransferInput{
			FromAccountID: e.a.ID, ToAccountID: e.a.ID, Amount: "1", Date: daysAgo(1),
		}); httpStatus(err) != http.StatusUnprocessableEntity {
			t.Errorf("UpdateTransfer to the same account = %v, want 422", err)
		}

		// An ordinary transaction is not a transfer.
		plain := insertTx(t, e.app, e.ana, e.a.ID, models.KindExpense, 100, daysAgo(1))
		if _, err := e.fin.UpdateTransfer(e.ctx, plain.ID, finance.TransferInput{
			FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "1", Date: daysAgo(1),
		}); httpStatus(err) != http.StatusNotFound {
			t.Errorf("UpdateTransfer of a plain transaction = %v, want 404", err)
		}
	})

	t.Run("an archived account of an existing transfer may keep its description edited", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		tr := e.move(t, e.a.ID, e.b.ID, "10.00")

		if _, err := e.fin.ArchiveAccount(e.ctx, e.b.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := e.fin.UpdateTransfer(e.ctx, tr.From.ID, finance.TransferInput{
			FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "10.00", Description: "note", Date: daysAgo(1),
		}); err != nil {
			t.Errorf("editing the note of a transfer with an archived account: %v", err)
		}
	})

	t.Run("deleting one leg deletes both, and restoring one brings both back", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		tr := e.move(t, e.a.ID, e.b.ID, "100.00")

		if err := e.fin.DeleteTransaction(e.ctx, tr.From.ID); err != nil {
			t.Fatal(err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 0 {
			t.Fatalf("%d legs are still live after deleting one", n)
		}

		if n := len(e.legs(t, tr.UID, true)); n != 2 {
			t.Fatalf("%d tombstones, want 2 (both kept)", n)
		}

		if e.balance(t, e.a.ID) != 100000 || e.balance(t, e.b.ID) != 50000 {
			t.Errorf("balances = %d, %d; want the openings back", e.balance(t, e.a.ID), e.balance(t, e.b.ID))
		}

		if err := e.fin.DeleteTransfer(e.ctx, tr.To.ID); httpStatus(err) != http.StatusNotFound {
			t.Errorf("deleting a deleted transfer = %v, want 404", err)
		}

		if _, err := e.fin.RestoreTransaction(e.ctx, tr.To.ID); err != nil {
			t.Fatal(err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 2 {
			t.Fatalf("%d legs live after the restore, want 2", n)
		}

		if e.balance(t, e.a.ID) != 90000 || e.balance(t, e.b.ID) != 60000 {
			t.Errorf("balances = %d, %d after the restore", e.balance(t, e.a.ID), e.balance(t, e.b.ID))
		}

		if err := e.fin.DeleteTransfer(e.ctx, tr.To.ID); err != nil {
			t.Fatal(err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 0 {
			t.Errorf("DeleteTransfer left %d legs", n)
		}
	})

	t.Run("deleting an account takes the other leg of its transfers, and restoring brings them back", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		tr := e.move(t, e.a.ID, e.b.ID, "100.00")

		if err := e.fin.DeleteAccount(e.ctx, e.a.ID); err != nil {
			t.Fatal(err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 0 {
			t.Fatalf("%d legs live after deleting the source account, want 0 (no half transfer)", n)
		}

		if got := e.balance(t, e.b.ID); got != 50000 {
			t.Errorf("destination balance = %d, want 50000", got)
		}

		if _, err := e.fin.RestoreAccount(e.ctx, e.a.ID); err != nil {
			t.Fatal(err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 2 || e.balance(t, e.b.ID) != 60000 {
			t.Errorf("legs = %d, destination %d; want both back and 60000", n, e.balance(t, e.b.ID))
		}

		// With the other account deleted too, the restore of one is refused.
		if err := e.fin.DeleteAccount(e.ctx, e.a.ID); err != nil {
			t.Fatal(err)
		}

		if err := e.fin.DeleteAccount(e.ctx, e.b.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := e.fin.RestoreAccount(e.ctx, e.a.ID); httpStatus(err) != http.StatusConflict {
			t.Errorf("restoring over a deleted counterpart = %v, want 409", err)
		}

		if n := len(e.legs(t, tr.UID, false)); n != 0 {
			t.Errorf("a refused restore left %d legs live", n)
		}
	})

	t.Run("a failure on the second leg leaves no first leg", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		db, err := e.app.DB()
		if err != nil {
			t.Fatal(err)
		}

		if _, err := db.SQL().ExecContext(t.Context(), `CREATE TRIGGER force_second_leg_failure BEFORE INSERT ON transactions
			WHEN NEW.kind = 'income' AND NEW.description = 'boom'
			BEGIN SELECT RAISE(ABORT, 'forced failure'); END`); err != nil {
			t.Fatal(err)
		}

		_, err = e.fin.CreateTransfer(e.ctx, finance.TransferInput{
			FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "10.00", Description: "boom", Date: daysAgo(1),
		})
		if err == nil {
			t.Fatal("the forced failure of the second leg did not surface")
		}

		if n := count[models.Transaction](t, e.app, e.ana); n != 0 {
			t.Errorf("%d transactions after a failed transfer, want 0 (the first leg must roll back)", n)
		}

		if n := trashedCount[models.Transaction](t, e.app, e.ana); n != 0 {
			t.Errorf("%d tombstones after a failed transfer, want 0", n)
		}

		if got := e.balance(t, e.a.ID); got != 100000 {
			t.Errorf("source balance = %d, want 100000", got)
		}

		var logged int
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM changes WHERE entity = 'transaction'").Scan(&logged); err != nil || logged != 0 {
			t.Errorf("the change log holds %d transaction changes after the rollback, %v", logged, err)
		}
	})

	t.Run("concurrent edits, deletes and restores keep the legs consistent", func(t *testing.T) {
		t.Parallel()

		e := newTransferEnv(t)
		tr := e.move(t, e.a.ID, e.b.ID, "100.00")

		var wg sync.WaitGroup

		for i := range 24 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				switch i % 3 {
				case 0:
					_, _ = e.fin.UpdateTransfer(e.ctx, tr.To.ID, finance.TransferInput{
						FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "1.0" + string(rune('0'+i%10)), Description: "edit", Date: daysAgo(1),
					})
				case 1:
					_ = e.fin.DeleteTransfer(e.ctx, tr.From.ID)
				default:
					_, _ = e.fin.RestoreTransaction(e.ctx, tr.To.ID)
				}
			}()
		}

		wg.Wait()

		live, dead := e.legs(t, tr.UID, false), e.legs(t, tr.UID, true)
		if len(live)+len(dead) != 2 || (len(live) != 0 && len(live) != 2) {
			t.Fatalf("live %d, deleted %d legs: the pair is split", len(live), len(dead))
		}

		pair := append(live, dead...)
		if pair[0].AmountCents != pair[1].AmountCents || pair[0].OccurredOn != pair[1].OccurredOn || pair[0].Description != pair[1].Description {
			t.Errorf("legs differ after concurrent edits: %+v vs %+v", pair[0], pair[1])
		}

		net, err := e.fin.NetWorth(e.ctx, "")
		if err != nil || net != 100000+50000+firstOf[models.Account](t, e.app, e.ana, "").OpeningCents {
			t.Errorf("net worth = %d, %v; a transfer must not change it", net, err)
		}
	})
}

// TestTransfersAreNotIncomeOrExpense proves a transfer moves value between
// accounts without touching income, expense, spending by category, budgets or
// the monthly quota.
func TestTransfersAreNotIncomeOrExpense(t *testing.T) {
	t.Parallel()

	e := newTransferEnv(t)
	makePro(t, e.app, e.ana)

	groceries := categoryID(t, e.app, e.ana, "Groceries")
	if err := e.fin.SetBudget(e.ctx, groceries, "100.00"); err != nil {
		t.Fatal(err)
	}

	date := today()

	if _, err := e.fin.CreateTransaction(e.ctx, finance.TxInput{
		AccountID: e.a.ID, CategoryID: groceries, Kind: models.KindExpense, Amount: "30.00", Date: date,
	}); err != nil {
		t.Fatal(err)
	}

	month := e.fin.Month(e.ctx)

	before, err := e.fin.Dashboard(e.ctx, month)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.fin.CreateTransfer(e.ctx, finance.TransferInput{
		FromAccountID: e.a.ID, ToAccountID: e.b.ID, Amount: "100.00", Description: "move", Date: date,
	}); err != nil {
		t.Fatal(err)
	}

	after, err := e.fin.Dashboard(e.ctx, month)
	if err != nil {
		t.Fatal(err)
	}

	last := func(d finance.Dashboard) finance.MonthPoint { return d.Months[len(d.Months)-1] }

	if last(after) != last(before) {
		t.Errorf("month totals = %+v, want %+v unchanged", last(after), last(before))
	}

	if len(after.Categories) != len(before.Categories) || after.Categories[0] != before.Categories[0] {
		t.Errorf("category spending = %+v, want %+v unchanged (no Uncategorized from a transfer)", after.Categories, before.Categories)
	}

	if len(after.Budgets) != 1 || after.Budgets[0].SpentCents != 3000 {
		t.Errorf("budget = %+v, want 3000 spent unchanged", after.Budgets)
	}

	if after.BalanceCents != before.BalanceCents {
		t.Errorf("dashboard total = %d, want %d: a transfer between live accounts changes no total", after.BalanceCents, before.BalanceCents)
	}

	if got := e.balance(t, e.a.ID); got != 100000-3000-10000 {
		t.Errorf("source balance = %d, want %d", got, 100000-3000-10000)
	}

	if got := e.balance(t, e.b.ID); got != 60000 {
		t.Errorf("destination balance = %d, want 60000", got)
	}

	// A transfer does not count toward the monthly quota of the Free plan.
	free := newTransferEnv(t)
	for range 100 {
		insertTx(t, free.app, free.ana, free.a.ID, models.KindExpense, 1, date)
	}

	if _, err := free.fin.CreateTransaction(free.ctx, finance.TxInput{
		AccountID: free.a.ID, Kind: models.KindExpense, Amount: "1.00", Date: date,
	}); !workspaces.IsUpgrade(err) {
		t.Fatalf("the 101st transaction = %v, want the upgrade answer", err)
	}

	if _, err := free.fin.CreateTransfer(free.ctx, finance.TransferInput{
		FromAccountID: free.a.ID, ToAccountID: free.b.ID, Amount: "5.00", Date: date,
	}); err != nil {
		t.Errorf("a transfer in a month at the quota: %v", err)
	}

	for range 3 {
		if _, err := free.fin.CreateTransfer(free.ctx, finance.TransferInput{
			FromAccountID: free.a.ID, ToAccountID: free.b.ID, Amount: "5.00", Date: date,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := free.fin.CreateTransaction(free.ctx, finance.TxInput{
		AccountID: free.a.ID, Kind: models.KindExpense, Amount: "1.00", Date: "2020-01-01",
	}); err != nil {
		t.Errorf("another month still has room: %v", err)
	}
}
