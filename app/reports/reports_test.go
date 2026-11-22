package reports_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

// seedQ1 writes a first quarter with a transfer, a deleted row and rows just
// outside the period, which no report of 2026-01-01..2026-03-31 may count.
func seedQ1(f *fx) (other models.Account) {
	f.t.Helper()

	other = f.newAccount("Savings", models.AccountSavings, "0")

	f.tx(f.account.ID, 0, "income", "1000.00", "2026-01-05")
	f.tx(f.account.ID, 0, "expense", "200.00", "2026-01-06")
	f.tx(other.ID, 0, "expense", "50.50", "2026-01-20")
	f.tx(f.account.ID, 0, "expense", "100.00", "2026-02-10")
	f.tx(f.account.ID, 0, "income", "5.00", "2026-03-31")

	if _, err := f.fin.CreateTransfer(f.ctx, finance.TransferInput{
		FromAccountID: f.account.ID, ToAccountID: other.ID, Amount: "300.00", Description: "move", Date: "2026-01-10",
	}); err != nil {
		f.t.Fatal(err)
	}

	gone := f.tx(f.account.ID, 0, "expense", "999.00", "2026-01-15")
	if err := f.fin.DeleteTransaction(f.ctx, gone.ID); err != nil {
		f.t.Fatal(err)
	}

	f.tx(f.account.ID, 0, "expense", "77.00", "2025-12-31")
	f.tx(f.account.ID, 0, "expense", "88.00", "2026-04-01")

	return other
}

func TestCashFlow(t *testing.T) {
	t.Parallel()

	f := newFx(t)
	seedQ1(f)

	got, err := f.rep.CashFlow(f.ctx, f.period("2026-01-01", "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}

	want := []reports.MonthFlow{
		{Month: "2026-01", Sum: reports.Sum{IncomeCents: 100000, ExpenseCents: 25050, NetCents: 74950}},
		{Month: "2026-02", Sum: reports.Sum{ExpenseCents: 10000, NetCents: -10000}},
		{Month: "2026-03", Sum: reports.Sum{IncomeCents: 500, NetCents: 500}},
	}

	if len(got.Months) != len(want) {
		t.Fatalf("months = %+v, want %+v", got.Months, want)
	}

	for i := range want {
		if got.Months[i] != want[i] {
			t.Errorf("month %d = %+v, want %+v", i, got.Months[i], want[i])
		}
	}

	if got.Totals != (reports.Sum{IncomeCents: 100500, ExpenseCents: 35050, NetCents: 65450}) {
		t.Errorf("totals = %+v: the transfer, the deleted row and the rows outside the period must not count", got.Totals)
	}

	if got.Currency != "USD" {
		t.Errorf("currency = %q", got.Currency)
	}

	t.Run("a partial month counts the dates inside the period only", func(t *testing.T) {
		part, err := f.rep.CashFlow(f.ctx, f.period("2026-01-06", "2026-01-20"))
		if err != nil {
			t.Fatal(err)
		}

		if len(part.Months) != 1 || part.Months[0].IncomeCents != 0 || part.Months[0].ExpenseCents != 25050 {
			t.Errorf("months = %+v, want one month with 250.50 of expense", part.Months)
		}
	})

	t.Run("a future dated row counts only when the period reaches it", func(t *testing.T) {
		f.tx(f.account.ID, 0, "expense", "10.00", "2027-01-05")

		before, err := f.rep.CashFlow(f.ctx, f.period("2026-12-01", "2026-12-31"))
		if err != nil {
			t.Fatal(err)
		}

		after, err := f.rep.CashFlow(f.ctx, f.period("2026-12-01", "2027-01-31"))
		if err != nil {
			t.Fatal(err)
		}

		if before.Totals.ExpenseCents != 0 || after.Totals.ExpenseCents != 1000 || len(after.Months) != 2 {
			t.Errorf("before = %+v, after = %+v", before.Totals, after)
		}
	})
}

func TestByCategory(t *testing.T) {
	t.Parallel()

	f := newFx(t)
	food := f.category("Eating out", "expense", 0)
	groceries := f.category("Market runs", "expense", food.ID)
	dining := f.category("Restaurants", "expense", food.ID)
	rent := f.category("Home rent", "expense", 0)
	salary := f.category("Paycheck", "income", 0)
	other := f.newAccount("Savings", models.AccountSavings, "0")

	f.tx(f.account.ID, food.ID, "expense", "10.00", "2026-01-02")
	f.tx(f.account.ID, groceries.ID, "expense", "60.00", "2026-01-03")
	f.tx(f.account.ID, dining.ID, "expense", "30.00", "2026-01-04")
	f.tx(f.account.ID, rent.ID, "expense", "50.00", "2026-01-05")
	f.tx(f.account.ID, 0, "expense", "50.00", "2026-01-06")
	f.tx(f.account.ID, salary.ID, "income", "500.00", "2026-01-07")
	f.tx(f.account.ID, rent.ID, "expense", "400.00", "2026-02-01")

	if _, err := f.fin.CreateTransfer(f.ctx, finance.TransferInput{
		FromAccountID: f.account.ID, ToAccountID: other.ID, Amount: "70.00", Date: "2026-01-08",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := f.rep.ByCategory(f.ctx, f.period("2026-01-01", "2026-01-31"), "expense")
	if err != nil {
		t.Fatal(err)
	}

	if got.TotalCents != 20000 {
		t.Fatalf("total = %d, want 20000 (the transfer, the income and February are out)", got.TotalCents)
	}

	if len(got.Rows) != 3 {
		t.Fatalf("rows = %+v, want Food, Rent and the uncategorized row", got.Rows)
	}

	first := got.Rows[0]
	if first.Name != "Eating out" || first.Cents != 10000 || first.OwnCents != 1000 || first.ShareBps != 5000 || len(first.Children) != 2 {
		t.Errorf("Food = %+v, want 100.00 rolled up from its own 10.00 and two children", first)
	}

	if first.Children[0].Name != "Market runs" || first.Children[0].Cents != 6000 || first.Children[0].ShareBps != 3000 ||
		first.Children[1].Name != "Restaurants" || first.Children[1].Cents != 3000 {
		t.Errorf("children = %+v, want Groceries 60.00 then Dining 30.00", first.Children)
	}

	var seen bool

	for _, r := range got.Rows[1:] {
		switch {
		case r.Uncategorized:
			seen = true

			if r.Cents != 5000 || r.UID != "" || r.ShareBps != 2500 {
				t.Errorf("uncategorized = %+v", r)
			}
		case r.Name == "Home rent":
			if r.Cents != 5000 || len(r.Children) != 0 {
				t.Errorf("Rent = %+v", r)
			}
		default:
			t.Errorf("unexpected row %+v", r)
		}
	}

	if !seen {
		t.Error("no uncategorized row")
	}

	income, err := f.rep.ByCategory(f.ctx, f.period("2026-01-01", "2026-01-31"), "income")
	if err != nil {
		t.Fatal(err)
	}

	if income.TotalCents != 50000 || len(income.Rows) != 1 || income.Rows[0].Name != "Paycheck" || income.Rows[0].ShareBps != 10000 {
		t.Errorf("income = %+v", income)
	}
}

func TestByAccount(t *testing.T) {
	t.Parallel()

	f := newFx(t)
	other := seedQ1(f)

	got, err := f.rep.ByAccount(f.ctx, f.period("2026-01-01", "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}

	rows := map[string]reports.AccountRow{}
	for _, r := range got.Rows {
		rows[r.Name] = r
	}

	if r := rows[f.account.Name]; r.IncomeCents != 100500 || r.ExpenseCents != 30000 || r.NetCents != 70500 {
		t.Errorf("%s = %+v", f.account.Name, r)
	}

	if r := rows[other.Name]; r.IncomeCents != 0 || r.ExpenseCents != 5050 || r.NetCents != -5050 {
		t.Errorf("%s = %+v: the transfer leg must not count", other.Name, r)
	}

	if len(got.Rows) != 2 {
		t.Errorf("rows = %+v", got.Rows)
	}
}

func TestByTag(t *testing.T) {
	t.Parallel()

	f := newFx(t)

	f.tx(f.account.ID, 0, "expense", "40.00", "2026-01-02", "trip", "work")
	f.tx(f.account.ID, 0, "expense", "10.00", "2026-01-03", "trip")
	f.tx(f.account.ID, 0, "income", "25.00", "2026-01-04", "work")
	f.tx(f.account.ID, 0, "expense", "5.00", "2026-01-05")
	f.tx(f.account.ID, 0, "expense", "900.00", "2026-02-05", "trip")

	gone := f.tx(f.account.ID, 0, "expense", "700.00", "2026-01-06", "trip")
	if err := f.fin.DeleteTransaction(f.ctx, gone.ID); err != nil {
		t.Fatal(err)
	}

	got, err := f.rep.ByTag(f.ctx, f.period("2026-01-01", "2026-01-31"))
	if err != nil {
		t.Fatal(err)
	}

	rows := map[string]reports.TagRow{}
	for _, r := range got.Rows {
		rows[r.Name] = r
	}

	if r := rows["trip"]; r.ExpenseCents != 5000 || r.IncomeCents != 0 {
		t.Errorf("trip = %+v, want 50.00: the deleted row and February are out", r)
	}

	if r := rows["work"]; r.ExpenseCents != 4000 || r.IncomeCents != 2500 || r.NetCents != -1500 {
		t.Errorf("work = %+v", r)
	}

	if len(got.Rows) != 2 || got.Rows[0].Name != "trip" {
		t.Errorf("rows = %+v, want trip (the larger expense) then work", got.Rows)
	}
}
