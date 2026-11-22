package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// TestMonthTotalsFollowCategoryAndAccountRemoval covers the writes no service
// names a transaction for: a deleted category clears it, and a foreign key
// cascade removes the rows of a deleted account.
func TestMonthTotalsFollowCategoryAndAccountRemoval(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	category, err := fin.CreateCategory(ctx, "Food", models.KindExpense)
	if err != nil {
		t.Fatal(err)
	}

	kept := firstAccount(t, app, ana)
	removed := insertAccount(t, app, ana, "Closed", 1000).ID

	for _, account := range []int64{kept, removed} {
		if _, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: account, CategoryID: category.ID, Kind: models.KindExpense, Amount: "12.34",
			Description: "x", Date: "2026-03-10",
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := fin.DeleteCategory(ctx, category.ID); err != nil {
		t.Fatal(err)
	}

	if err := finance.CheckMonthTotals(t.Context(), db); err != nil {
		t.Fatalf("after deleting the category: %v", err)
	}

	if _, err := db.ExecContext(t.Context(), "DELETE FROM accounts WHERE id = ?", removed); err != nil {
		t.Fatal(err)
	}

	var left int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM transactions WHERE account_id = ?", removed).Scan(&left); err != nil {
		t.Fatal(err)
	}

	if left != 0 {
		t.Fatalf("the cascade left %d transactions", left)
	}

	if err := finance.CheckMonthTotals(t.Context(), db); err != nil {
		t.Fatalf("after the cascade from the account: %v", err)
	}
}
