package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func statusOf(err error) int {
	if e, ok := errAs[*errs.HTTPError](err); ok {
		return e.Status
	}

	return 0
}

func errAs[T error](err error) (T, bool) {
	var zero T

	for err != nil {
		if v, ok := err.(T); ok { //nolint:errorlint // walks the chain by hand to stay generic
			return v, true
		}

		u, ok := err.(interface{ Unwrap() error }) //nolint:errorlint // same
		if !ok {
			return zero, false
		}

		err = u.Unwrap()
	}

	return zero, false
}

func TestCategories(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	add := func(name, kind string, parent int64) models.Category {
		t.Helper()

		c, err := fin.AddCategory(ctx, finance.CategoryInput{Name: name, Kind: kind, ParentID: parent})
		if err != nil {
			t.Fatalf("add %s: %v", name, err)
		}

		return c
	}

	t.Run("one level only, same kind", func(t *testing.T) {
		food := add("Food", models.KindExpense, 0)
		pizza := add("Pizza", "", food.ID)

		if !pizza.ParentID.Valid || pizza.ParentID.V != food.ID || pizza.Kind != models.KindExpense {
			t.Fatalf("pizza = %+v, want a child of Food with its kind", pizza)
		}

		// Depth: a child cannot have children.
		if _, err := fin.AddCategory(ctx, finance.CategoryInput{Name: "Slice", ParentID: pizza.ID}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("a child of a child = %v, want 422", err)
		}

		// Kind: a child has the kind of its parent.
		if _, err := fin.AddCategory(ctx, finance.CategoryInput{Name: "Bonus", Kind: models.KindIncome, ParentID: food.ID}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("a child of another kind = %v, want 422", err)
		}

		// Cycles: itself, its own child as parent, and a parent that becomes a child.
		for name, parent := range map[string]int64{"itself": food.ID, "its child": pizza.ID} {
			if _, err := fin.UpdateCategory(ctx, food.ID, finance.CategoryInput{Name: "Food", ParentID: parent}); statusOf(err) != http.StatusUnprocessableEntity {
				t.Errorf("Food under %s = %v, want 422", name, err)
			}
		}

		other := add("Dining", models.KindExpense, 0)
		if _, err := fin.UpdateCategory(ctx, food.ID, finance.CategoryInput{Name: "Food", ParentID: other.ID}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("a parent with children under another = %v, want 422", err)
		}

		// A move to another top-level category of the same kind works, and so does a promotion.
		moved, err := fin.UpdateCategory(ctx, pizza.ID, finance.CategoryInput{Name: "Pizza", ParentID: other.ID, Color: "red", Icon: "utensils"})
		if err != nil || moved.ParentID.V != other.ID || moved.Color != "red" || moved.Icon != "utensils" {
			t.Fatalf("move = %+v, %v", moved, err)
		}

		if p, err := fin.UpdateCategory(ctx, pizza.ID, finance.CategoryInput{Name: "Pizza"}); err != nil || p.ParentID.Valid {
			t.Fatalf("promote = %+v, %v", p, err)
		}

		// Names stay unique, and look values come from the lists.
		if _, err := fin.UpdateCategory(ctx, pizza.ID, finance.CategoryInput{Name: "Food"}); statusOf(err) != http.StatusConflict {
			t.Errorf("a taken name = %v, want 409", err)
		}

		if _, err := fin.UpdateCategory(ctx, pizza.ID, finance.CategoryInput{Name: "Pizza", Color: "#ff0000"}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("a hex color = %v, want 422", err)
		}

		if _, err := fin.UpdateCategory(ctx, pizza.ID, finance.CategoryInput{Name: "Pizza", Icon: "rocket"}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("an unknown icon = %v, want 422", err)
		}
	})

	t.Run("archive hides it from the pickers and keeps its history", func(t *testing.T) {
		books := add("Books", models.KindExpense, 0)
		comics := add("Comics", "", books.ID)
		addTransaction(t, ana, acc, books.ID, "expense", "9.99", "2026-03-02").AssertRedirect("/transactions")

		got, err := fin.ArchiveCategory(ctx, books.ID)
		if err != nil || !got.ArchivedAt.Valid {
			t.Fatalf("archive = %+v, %v", got, err)
		}

		child, err := fin.UnarchiveCategory(ctx, comics.ID)
		if statusOf(err) != http.StatusConflict {
			t.Fatalf("unarchive a child of an archived parent = %+v, %v; want 409", child, err)
		}

		active, err := fin.ActiveCategories(ctx)
		if err != nil {
			t.Fatal(err)
		}

		for _, c := range active {
			if c.ID == books.ID || c.ID == comics.ID {
				t.Errorf("%s is archived and must not be offered", c.Name)
			}
		}

		// New entries refuse it, history stays.
		addTransaction(t, ana, acc, books.ID, "expense", "1.00", "2026-03-03").AssertStatus(http.StatusSeeOther)
		rows, err := fin.Transactions(ctx, finance.TxFilter{CategoryID: books.ID})
		if err != nil || len(rows) != 1 {
			t.Fatalf("history = %d rows, %v; want 1", len(rows), err)
		}

		if _, err := fin.UpdateTransaction(ctx, rows[0].ID, finance.TxInput{
			AccountID: acc, CategoryID: books.ID, Kind: "expense", Amount: "10.00", Description: "kept", Date: "2026-03-02",
		}); err != nil {
			t.Errorf("editing a row that already has the archived category: %v", err)
		}

		back, err := fin.UnarchiveCategory(ctx, books.ID)
		if err != nil || back.ArchivedAt.Valid {
			t.Fatalf("unarchive = %+v, %v", back, err)
		}

		kid, _ := fin.Categories(ctx)
		for _, c := range kid {
			if c.ID == comics.ID && c.ArchivedAt.Valid {
				t.Error("the child archived with its parent comes back with it")
			}
		}
	})

	t.Run("delete needs a target, and moves transactions and budgets in one go", func(t *testing.T) {
		from := add("Old pets", models.KindExpense, 0)
		to := add("Pets", models.KindExpense, 0)
		income := add("Gifts in", models.KindIncome, 0)

		addTransaction(t, ana, acc, from.ID, "expense", "20.00", "2026-03-04").AssertRedirect("/transactions")
		addTransaction(t, ana, acc, from.ID, "expense", "5.00", "2026-03-05").AssertRedirect("/transactions")

		if err := fin.SetBudget(ctx, from.ID, "100.00"); err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteCategoryReassigning(ctx, from.ID, 0); statusOf(err) != http.StatusUnprocessableEntity {
			t.Fatalf("delete without a target = %v, want 422", err)
		}

		for name, target := range map[string]int64{"itself": from.ID, "another kind": income.ID, "a foreign id": 999999} {
			if err := fin.DeleteCategoryReassigning(ctx, from.ID, target); statusOf(err) != http.StatusUnprocessableEntity {
				t.Errorf("delete into %s = %v, want 422", name, err)
			}
		}

		if rows, _ := fin.Transactions(ctx, finance.TxFilter{CategoryID: from.ID}); len(rows) != 2 {
			t.Fatalf("a refused delete moved %d rows away", 2-len(rows))
		}

		if err := fin.DeleteCategoryReassigning(ctx, from.ID, to.ID); err != nil {
			t.Fatal(err)
		}

		rows, err := fin.Transactions(ctx, finance.TxFilter{CategoryID: to.ID})
		if err != nil || len(rows) != 2 {
			t.Fatalf("target has %d rows, %v; want 2", len(rows), err)
		}

		budgets, err := fin.Budgets(ctx, "2026-03")
		if err != nil {
			t.Fatal(err)
		}

		var moved bool

		for _, b := range budgets {
			if b.CategoryID == from.ID {
				t.Error("the budget still points at the deleted category")
			}

			moved = moved || (b.CategoryID == to.ID && b.LimitCents == 10000 && b.SpentCents == 2500)
		}

		if !moved {
			t.Errorf("budgets = %+v, want the 100.00 budget on Pets with 25.00 spent", budgets)
		}

		// A parent with subcategories is refused; an unused category needs no target.
		parent := add("Travel", models.KindExpense, 0)
		add("Flights", "", parent.ID)

		if err := fin.DeleteCategoryReassigning(ctx, parent.ID, to.ID); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("delete a parent with children = %v, want 422", err)
		}

		if err := fin.DeleteCategoryReassigning(ctx, income.ID, 0); err != nil {
			t.Errorf("delete an unused category: %v", err)
		}
	})

	t.Run("a target that has a budget keeps its own", func(t *testing.T) {
		a := add("Budget A", models.KindExpense, 0)
		b := add("Budget B", models.KindExpense, 0)

		if err := fin.SetBudget(ctx, a.ID, "10.00"); err != nil {
			t.Fatal(err)
		}

		if err := fin.SetBudget(ctx, b.ID, "20.00"); err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteCategoryReassigning(ctx, a.ID, b.ID); err != nil {
			t.Fatal(err)
		}

		budgets, _ := fin.Budgets(ctx, "2026-03")
		for _, bs := range budgets {
			if bs.CategoryID == b.ID && bs.LimitCents != 2000 {
				t.Errorf("target budget = %d, want its own 2000", bs.LimitCents)
			}
		}
	})

	t.Run("transfer legs are never touched by a category change", func(t *testing.T) {
		from := add("Source", models.KindExpense, 0)
		to := add("Sink", models.KindExpense, 0)

		other, err := fin.CreateAccount(ctx, "Second", "savings", "0")
		if err != nil {
			t.Fatal(err)
		}

		tr, err := fin.CreateTransfer(ctx, finance.TransferInput{
			FromAccountID: acc, ToAccountID: other.ID, Amount: "7.00", Date: "2026-03-06",
		})
		if err != nil {
			t.Fatal(err)
		}

		addTransaction(t, ana, acc, from.ID, "expense", "1.00", "2026-03-06").AssertRedirect("/transactions")

		if err := fin.DeleteCategoryReassigning(ctx, from.ID, to.ID); err != nil {
			t.Fatal(err)
		}

		legs, err := fin.Transactions(ctx, finance.TxFilter{})
		if err != nil {
			t.Fatal(err)
		}

		var seen int

		for _, l := range legs {
			if l.TransferUID.Valid {
				seen++

				if l.CategoryID.Valid || l.TransferUID.V != tr.UID {
					t.Errorf("leg %d = %+v, want no category and its transfer", l.ID, l)
				}
			}
		}

		if seen < 2 {
			t.Errorf("saw %d transfer legs, want both", seen)
		}
	})
}
