package categorize_test

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/aitest"
	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestAcceptTeachesTheMemoryAndRejectDoesNot(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries := r.Category(t, "Groceries")
	answerAll(r, func(string) (string, float64) { return groceries.UID, 0.7 })

	a := r.Add(t, "Padaria Sao Joao", 0)
	b := r.Add(t, "Cinema Central", 0)

	svc := newService(r)
	if _, err := svc.Run(r.Ctx, categorize.Options{}); err != nil {
		t.Fatal(err)
	}

	for _, s := range suggestionsOf(t, r) {
		if s.TransactionUID == a.UID {
			if err := svc.Accept(r.Ctx, s.ID); err != nil {
				t.Fatal(err)
			}

			if err := svc.Accept(r.Ctx, s.ID); err == nil {
				t.Error("a suggestion can be accepted once")
			}
		} else if err := svc.Reject(r.Ctx, s.ID); err != nil {
			t.Fatal(err)
		}
	}

	if got := r.Reload(t, a.ID).CategoryID; !got.Valid || got.V != groceries.ID {
		t.Errorf("accepted category = %+v", got)
	}

	if r.Reload(t, b.ID).CategoryID.Valid {
		t.Error("a rejected suggestion changed the transaction")
	}

	if _, ok, _ := r.Fin.RuleCategory(r.Ctx, r.DB, "Padaria Sao Joao 77", models.KindExpense); !ok {
		t.Error("an accepted suggestion must teach the memory")
	}

	if _, ok, _ := r.Fin.RuleCategory(r.Ctx, r.DB, "Cinema Central", models.KindExpense); ok {
		t.Error("a rejected suggestion must not teach the memory")
	}

	// A rejected suggestion is not proposed again.
	r.Provider.Reply = nil
	before := r.Provider.Calls()

	if _, err := svc.Run(r.Ctx, categorize.Options{}); err != nil || r.Provider.Calls() != before {
		t.Errorf("run after a rejection made a call: %v", err)
	}
}

func TestAcceptRefusesAStaleSuggestion(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries, leisure := r.Category(t, "Groceries"), r.Category(t, "Leisure")
	answerAll(r, func(string) (string, float64) { return groceries.UID, 0.7 })

	tx := r.Add(t, "Padaria", 0)
	svc := newService(r)

	if _, err := svc.Run(r.Ctx, categorize.Options{}); err != nil {
		t.Fatal(err)
	}

	// The person categorizes it by hand meanwhile.
	if _, err := r.Fin.UpdateTransaction(r.Ctx, tx.ID, finanInput(r, tx, leisure.ID)); err != nil {
		t.Fatal(err)
	}

	if err := svc.Accept(r.Ctx, suggestionsOf(t, r)[0].ID); err == nil {
		t.Fatal("a suggestion for a categorized transaction must be refused")
	}

	if got := r.Reload(t, tx.ID).CategoryID.V; got != leisure.ID {
		t.Errorf("category = %d, the person's choice must stay", got)
	}
}

func TestAutoApplyIsOffByDefault(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries := r.Category(t, "Groceries")
	answerAll(r, func(string) (string, float64) { return groceries.UID, 0.99 })

	tx := r.Add(t, "Padaria", 0)

	if res, err := newService(r).Run(r.Ctx, categorize.Options{}); err != nil || res.Applied != 0 {
		t.Fatalf("%+v %v", res, err)
	}

	if r.Reload(t, tx.ID).CategoryID.Valid {
		t.Error("nothing is applied unless the owner opted in")
	}
}

func TestUndoAutoApplied(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, true, ai.FeatureCategorize)
	groceries, leisure := r.Category(t, "Groceries"), r.Category(t, "Leisure")
	answerAll(r, func(string) (string, float64) { return groceries.UID, 0.95 })

	low := r.Add(t, "Low confidence", 0)
	tx := r.Add(t, "Padaria", 0)
	changed := r.Add(t, "Mercado", 0)

	svc := newService(r)

	res, err := svc.Run(r.Ctx, categorize.Options{})
	if err != nil || res.Applied != 3 {
		t.Fatalf("result = %+v, %v", res, err)
	}

	_ = low

	if got := r.Reload(t, tx.ID).CategoryID; !got.Valid || got.V != groceries.ID {
		t.Fatalf("auto-applied category = %+v", got)
	}

	_, applied, err := svc.List(r.Ctx)
	if err != nil || len(applied) != 3 {
		t.Fatalf("applied list = %v, %v", applied, err)
	}

	byTx := map[string]int64{}
	for _, a := range applied {
		byTx[a.TransactionUID] = a.ID
	}

	// The person recategorized one of them: its undo must not take their choice away.
	if _, err := r.Fin.UpdateTransaction(r.Ctx, changed.ID, finanInput(r, changed, leisure.ID)); err != nil {
		t.Fatal(err)
	}

	if err := svc.Undo(r.Ctx, byTx[tx.UID]); err != nil {
		t.Fatal(err)
	}

	if r.Reload(t, tx.ID).CategoryID.Valid {
		t.Error("undo must take the category back")
	}

	if err := svc.Undo(r.Ctx, byTx[changed.UID]); err != nil {
		t.Fatal(err)
	}

	if got := r.Reload(t, changed.ID).CategoryID.V; got != leisure.ID {
		t.Error("undo must keep a category the person chose since")
	}

	if err := svc.Undo(r.Ctx, byTx[tx.UID]); err == nil {
		t.Error("undo works once")
	}

	// After 7 days it is final.
	late := categorize.New(r.DB, r.Client, r.Fin, nil, func() time.Time { return time.Now().Add(8 * 24 * time.Hour) })
	if err := late.Undo(r.Ctx, byTx[low.UID]); err == nil {
		t.Error("an undo after 7 days must be refused")
	}
}

func finanInput(r *aitest.Rig, tx models.Transaction, category int64) finance.TxInput {
	return finance.TxInput{
		AccountID: r.Account, CategoryID: category, Kind: tx.Kind, Amount: "10.00", Description: tx.Description, Date: tx.OccurredOn,
	}
}
