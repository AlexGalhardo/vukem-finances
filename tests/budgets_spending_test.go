package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func budgetOf(t *testing.T, fin *finance.Service, w who, month string, pick func(finance.BudgetStatus) bool) finance.BudgetStatus {
	t.Helper()

	all, err := fin.Budgets(wsCtx(t, w), month)
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range all {
		if pick(b) {
			return b
		}
	}

	t.Fatalf("no such budget in %+v", all)

	return finance.BudgetStatus{}
}

func spend(t *testing.T, fin *finance.Service, w who, acc, cat int64, amount, date string) models.Transaction {
	t.Helper()

	tx, err := fin.CreateTransaction(wsCtx(t, w), finance.TxInput{
		AccountID: acc, CategoryID: cat, Kind: models.KindExpense, Amount: amount, Description: "x", Date: date,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tx
}

func TestBudgetSpending(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-15")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, ana, acc, food, "10.00", "2027-03-02")
	gone := spend(t, fin, ana, acc, food, "40.00", "2027-03-03")
	spend(t, fin, ana, acc, food, "5.00", "2027-02-28") // another month
	spend(t, fin, ana, acc, 0, "7.00", "2027-03-04")    // uncategorized

	if err := fin.DeleteTransaction(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}

	// A transfer moves money: it is never spending, whatever it is called.
	other, err := fin.CreateAccount(ctx, "Second", "savings", "0")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateTransfer(ctx, finance.TransferInput{
		FromAccountID: acc, ToAccountID: other.ID, Amount: "300.00", Date: "2027-03-05",
	}); err != nil {
		t.Fatal(err)
	}

	// A statement payment is a transfer too; the purchase it pays counts once.
	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	spend(t, fin, ana, card.AccountID, food, "20.00", "2027-03-06")

	if _, err := fin.PayStatement(ctx, card.ID, "2027-03-10", finance.PaymentInput{FromAccountID: acc, Date: "2027-03-12"}); err != nil {
		t.Fatal(err)
	}

	got := budgetOf(t, fin, ana, "2027-03", func(b finance.BudgetStatus) bool { return b.CategoryID == food })
	if got.SpentCents != 3000 || got.Percent != 30 {
		t.Fatalf("spent = %d (%d%%), want 3000 (30%%)", got.SpentCents, got.Percent)
	}

	if _, err := fin.Budgets(ctx, "0000-01"); err == nil {
		t.Error("the year 0000 was accepted as a month")
	}

	if _, err := fin.Budgets(ctx, "2027-3"); err == nil {
		t.Error("a short month was accepted")
	}
}

func TestParentBudget(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-15")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	kid, err := fin.AddCategory(ctx, finance.CategoryInput{Name: "Snacks", ParentID: food})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []int64{food, kid.ID} {
		if err := fin.SetBudget(ctx, id, "100.00"); err != nil {
			t.Fatal(err)
		}
	}

	spend(t, fin, ana, acc, food, "10.00", "2027-03-02")
	spend(t, fin, ana, acc, kid.ID, "25.00", "2027-03-02")

	parent := budgetOf(t, fin, ana, "2027-03", func(b finance.BudgetStatus) bool { return b.CategoryID == food })
	child := budgetOf(t, fin, ana, "2027-03", func(b finance.BudgetStatus) bool { return b.CategoryID == kid.ID })

	if parent.SpentCents != 3500 || child.SpentCents != 2500 {
		t.Fatalf("parent %d child %d, want 3500 and 2500", parent.SpentCents, child.SpentCents)
	}
}

func TestOverallBudget(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-15")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	for _, limit := range []string{"200.00", "50.00"} { // the second write edits the first
		if err := fin.SaveBudget(ctx, finance.BudgetInput{Limit: limit}); err != nil {
			t.Fatal(err)
		}
	}

	spend(t, fin, ana, acc, food, "10.00", "2027-03-02")
	spend(t, fin, ana, acc, 0, "15.00", "2027-03-02")

	second, err := fin.CreateAccount(ctx, "Second", "savings", "0")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateTransfer(ctx, finance.TransferInput{
		FromAccountID: acc, ToAccountID: second.ID, Amount: "99.00", Date: "2027-03-03",
	}); err != nil {
		t.Fatal(err)
	}

	all, err := fin.Budgets(ctx, "2027-03")
	if err != nil || len(all) != 1 || !all[0].Overall {
		t.Fatalf("budgets = %+v (%v), want one overall budget", all, err)
	}

	if all[0].LimitCents != 5000 || all[0].SpentCents != 2500 || all[0].Percent != 50 {
		t.Fatalf("overall = %+v", all[0])
	}
}

func TestBudgetRollover(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-02-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	on := true
	if err := fin.SaveBudget(ctx, finance.BudgetInput{CategoryID: food, Limit: "100.00", Rollover: &on}); err != nil {
		t.Fatal(err)
	}

	// The rows are stamped with the real clock; say the budget was made in February.
	if _, err := mustDB(t, app).SQL().ExecContext(t.Context(), "UPDATE budgets SET created_at = '2027-02-10T10:00:00Z'"); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, ana, acc, food, "30.00", "2027-02-11")
	spend(t, fin, ana, acc, food, "150.00", "2027-03-02")

	clock.Set(day("2027-03-15"))

	// The budget was made in February, so February's 70.00 left over carries.
	// A month that overspent carries nothing (April).
	mar := budgetOf(t, fin, ana, "2027-03", func(b finance.BudgetStatus) bool { return b.CategoryID == food })
	if mar.CarryCents != 7000 || mar.AvailableCents != 17000 || mar.Percent != 88 {
		t.Fatalf("march: carry %d available %d percent %d, want 7000, 17000 and 88", mar.CarryCents, mar.AvailableCents, mar.Percent)
	}

	apr := budgetOf(t, fin, ana, "2027-04", func(b finance.BudgetStatus) bool { return b.CategoryID == food })
	if apr.CarryCents != 0 || apr.AvailableCents != 10000 {
		t.Fatalf("april after an overspent month: carry %d available %d, want 0 and 10000", apr.CarryCents, apr.AvailableCents)
	}

	// The creation month itself has no earlier month to carry from.
	feb := budgetOf(t, fin, ana, "2027-02", func(b finance.BudgetStatus) bool { return b.CategoryID == food })
	if feb.CarryCents != 0 {
		t.Fatalf("carry in the month the budget was made = %d, want 0", feb.CarryCents)
	}
}
