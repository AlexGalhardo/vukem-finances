package reports_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

func TestNetWorthSeries(t *testing.T) {
	t.Parallel()

	f := newFx(t)
	opening := f.account.OpeningCents
	savings := f.newAccount("Savings", models.AccountSavings, "1000.00")
	old := f.newAccount("Old", models.AccountCash, "999.00")

	card, err := f.fin.CreateCard(f.ctx, finance.CardInput{Name: "Visa", Limit: "5000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	f.tx(f.account.ID, 0, "income", "500.00", "2026-01-05")
	f.tx(card.AccountID, 0, "expense", "200.00", "2026-01-15")

	if _, err := f.fin.CreateTransfer(f.ctx, finance.TransferInput{
		FromAccountID: f.account.ID, ToAccountID: savings.ID, Amount: "300.00", Date: "2026-01-20",
	}); err != nil {
		t.Fatal(err)
	}

	f.tx(f.account.ID, 0, "expense", "50.00", "2026-02-10")
	f.tx(old.ID, 0, "expense", "10.00", "2026-02-11")

	if _, err := f.fin.ArchiveAccount(f.ctx, old.ID); err != nil {
		t.Fatal(err)
	}

	got, err := f.rep.NetWorthSeries(f.ctx, f.period("2026-01-01", "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}

	// Jan: opening + 1000.00 (savings) + 500.00 - 200.00 of card debt. The transfer changes nothing.
	want := []int64{opening + 130000, opening + 125000, opening + 125000}

	if len(got.Points) != len(want) {
		t.Fatalf("points = %+v", got.Points)
	}

	for i, w := range want {
		if got.Points[i].BalanceCents != w {
			t.Errorf("%s = %d, want %d", got.Points[i].Month, got.Points[i].BalanceCents, w)
		}
	}

	t.Run("it agrees with the net worth the dashboard shows", func(t *testing.T) {
		now, err := f.fin.NetWorth(f.ctx, "2026-03-31")
		if err != nil {
			t.Fatal(err)
		}

		if now != got.Points[2].BalanceCents {
			t.Errorf("NetWorth = %d, series = %d", now, got.Points[2].BalanceCents)
		}
	})

	t.Run("card debt lowers the net worth", func(t *testing.T) {
		before := got.Points[0].BalanceCents
		f.tx(card.AccountID, 0, "expense", "100.00", "2026-02-12")

		again, err := f.rep.NetWorthSeries(f.ctx, f.period("2026-01-01", "2026-03-31"))
		if err != nil {
			t.Fatal(err)
		}

		if again.Points[0].BalanceCents != before || again.Points[1].BalanceCents != want[1]-10000 {
			t.Errorf("points = %+v, want January unchanged and February 100.00 lower", again.Points)
		}
	})
}

func TestCompare(t *testing.T) {
	t.Parallel()

	f := newFx(t)

	f.tx(f.account.ID, 0, "expense", "50.00", "2026-02-10")
	f.tx(f.account.ID, 0, "income", "10.00", "2026-03-10")

	got, err := f.rep.Compare(f.ctx, f.period("2026-03-01", "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}

	if got.Previous.From != "2026-02-01" || got.Previous.To != "2026-02-28" {
		t.Fatalf("previous = %+v", got.Previous)
	}

	want := map[string]reports.CompareRow{
		reports.CompareIncome:  {Key: "income", CurrentCents: 1000, DiffCents: 1000, NotApplicable: true},
		reports.CompareExpense: {Key: "expense", PreviousCents: 5000, DiffCents: -5000, ChangeBps: -10000},
		reports.CompareNet:     {Key: "net", CurrentCents: 1000, PreviousCents: -5000, DiffCents: 6000, ChangeBps: 12000},
	}

	for _, r := range got.Rows {
		if r != want[r.Key] {
			t.Errorf("%s = %+v, want %+v", r.Key, r, want[r.Key])
		}
	}

	if len(got.Rows) != 3 {
		t.Errorf("rows = %+v", got.Rows)
	}
}
