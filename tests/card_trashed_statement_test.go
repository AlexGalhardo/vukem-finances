package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// A purchase in the trash, alone on its statement, follows a closing-day change.
func TestTrashedPurchaseFollowsTheClosingDay(t *testing.T) {
	t.Parallel()

	app, ana, _ := proWorkspace(t)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	tx, err := fin.CreateTransaction(ctx, finance.TxInput{
		AccountID: card.AccountID, Kind: models.KindExpense, Amount: "5.00", Description: "alone", Date: "2026-03-05",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.UpdateCard(ctx, card.ID, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 20, DueDay: 25}); err != nil {
		t.Fatal(err)
	}

	got, err := fin.RestoreTransaction(ctx, tx.ID)
	if err != nil || got.StatementRef.V != "2026-03-20" {
		t.Errorf("restored statement = %q, %v; want 2026-03-20", got.StatementRef.V, err)
	}
}
