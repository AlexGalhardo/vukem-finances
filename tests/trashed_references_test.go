package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// A transaction in the trash can come back, so deleting what it points at must
// not leave it pointing at a category or tag that no longer exists.
func TestRestoredTransactionHasNoDeadReferences(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	category := func(name string) models.Category {
		t.Helper()

		c, err := fin.AddCategory(ctx, finance.CategoryInput{Name: name, Kind: models.KindExpense})
		if err != nil {
			t.Fatal(err)
		}

		return c
	}

	trashedIn := func(categoryID int64, tags []string) models.Transaction {
		t.Helper()

		tx, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: acc, CategoryID: categoryID, Kind: models.KindExpense, Amount: "5.00",
			Description: "trashed", Date: "2026-03-01", Tags: tags,
		})
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
			t.Fatal(err)
		}

		return tx
	}

	t.Run("reassigned category", func(t *testing.T) {
		from, to := category("Old"), category("New")
		tx := trashedIn(from.ID, nil)

		if err := fin.DeleteCategoryReassigning(ctx, from.ID, to.ID); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreTransaction(ctx, tx.ID)
		if err != nil || !got.CategoryID.Valid || got.CategoryID.V != to.ID {
			t.Errorf("restored = %+v, %v; want category %d", got.CategoryID, err, to.ID)
		}
	})

	t.Run("deleted category without a target", func(t *testing.T) {
		gone := category("Gone")
		tx := trashedIn(gone.ID, nil)

		if err := fin.DeleteCategoryReassigning(ctx, gone.ID, 0); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreTransaction(ctx, tx.ID)
		if err != nil || got.CategoryID.Valid {
			t.Errorf("restored = %+v, %v; want no category", got.CategoryID, err)
		}
	})

	t.Run("deleted tag", func(t *testing.T) {
		tag, err := fin.CreateTag(ctx, "trip", "")
		if err != nil {
			t.Fatal(err)
		}

		tx := trashedIn(0, []string{"trip"})

		if err := fin.DeleteTag(ctx, tag.ID); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreTransaction(ctx, tx.ID)
		if err != nil || got.TagUIDs != "[]" {
			t.Errorf("restored tag_uids = %q, %v; want []", got.TagUIDs, err)
		}
	})
}
