package tests

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	capAmount  = "90071992547409.91" // 9007199254740991 cents, 2^53 - 1
	overAmount = "90071992547409.92"
)

// TestAmountCap checks that every write path accepts 2^53 - 1 cents and
// refuses one more, so any client can hold every stored amount exactly.
func TestAmountCap(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)
	groceries := categoryID(t, app, ana, "Groceries")

	t.Run("transactions", func(t *testing.T) {
		in := finance.TxInput{AccountID: acc, Kind: models.KindIncome, Amount: overAmount, Date: today()}
		if _, err := fin.CreateTransaction(ctx, in); err == nil {
			t.Fatal("a transaction over the cap was accepted")
		}

		in.Amount = capAmount

		tx, err := fin.CreateTransaction(ctx, in)
		if err != nil || tx.AmountCents != models.MaxCents {
			t.Fatalf("a transaction at the cap = %+v, %v; want it stored", tx, err)
		}

		in.Amount = overAmount
		if _, err := fin.UpdateTransaction(ctx, tx.ID, in); err == nil {
			t.Fatal("an edit over the cap was accepted")
		}
	})

	t.Run("account opening balance", func(t *testing.T) {
		if _, err := fin.CreateAccount(ctx, "Over", "savings", overAmount); err == nil {
			t.Fatal("an opening balance over the cap was accepted")
		}

		if _, err := fin.CreateAccount(ctx, "Exact", "savings", "-"+capAmount); err != nil {
			t.Fatalf("an opening balance at the negative cap: %v", err)
		}
	})

	t.Run("budget limit", func(t *testing.T) {
		if err := fin.SetBudget(ctx, groceries, overAmount); err == nil {
			t.Fatal("a budget limit over the cap was accepted")
		}

		if err := fin.SetBudget(ctx, groceries, capAmount); err != nil {
			t.Fatalf("a budget limit at the cap: %v", err)
		}
	})

	t.Run("the models refuse it too, whatever the writer", func(t *testing.T) {
		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		for name, m := range map[string]any{
			"account":     &models.Account{Name: "x", Kind: "cash", OpeningCents: models.MaxCents + 1},
			"transaction": &models.Transaction{AccountID: acc, Kind: models.KindIncome, AmountCents: models.MaxCents + 1, OccurredOn: today()},
			"budget":      &models.Budget{CategoryID: sql.Null[int64]{V: groceries, Valid: true}, LimitCents: models.MaxCents + 1},
		} {
			var err error

			switch v := m.(type) {
			case *models.Account:
				err = orm.Create(ctx, db, v)
			case *models.Transaction:
				err = orm.Create(ctx, db, v)
			case *models.Budget:
				err = orm.Create(ctx, db, v)
			}

			if !errors.Is(err, models.ErrAmountRange) {
				t.Errorf("%s over the cap: err = %v, want models.ErrAmountRange", name, err)
			}
		}
	})
}
