package tests

import (
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func changeCount(t *testing.T, app *vukemtest.App, w who, entity, op string) int {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int

	err = db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM changes WHERE workspace_id = ? AND entity = ? AND op = ?", w.Workspace.ID, entity, op).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}

	return n
}

func TestDeleteAccountLogsItsTransactions(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	for _, amount := range []string{"1.00", "2.00", "3.00"} {
		if _, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: acc, Kind: models.KindExpense, Amount: amount, Date: today(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := fin.DeleteAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}

	if got := changeCount(t, app, ana, "transaction", "delete"); got != 3 {
		t.Errorf("delete changes of transactions = %d, want 3 (one per cascaded row)", got)
	}

	if got := changeCount(t, app, ana, "account", "delete"); got != 1 {
		t.Errorf("delete changes of accounts = %d, want 1", got)
	}

	if got := changeCount(t, app, bob, "transaction", "delete") + changeCount(t, app, bob, "account", "delete"); got != 0 {
		t.Errorf("another workspace holds %d of these changes, want 0", got)
	}

	if n := trashedCount[models.Transaction](t, app, ana); n != 3 {
		t.Errorf("transaction tombstones = %d, want 3", n)
	}
}

func TestRestore(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "personal"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	newTx := func(desc string) models.Transaction {
		t.Helper()

		tx, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: acc, Kind: models.KindExpense, Amount: "9.00", Description: desc, Date: today(),
		})
		if err != nil {
			t.Fatal(err)
		}

		return tx
	}

	t.Run("an account comes back with the transactions deleted with it, not the ones deleted before", func(t *testing.T) {
		before := newTx("deleted before")
		newTx("deleted with the account")

		if err := fin.DeleteTransaction(ctx, before.ID); err != nil {
			t.Fatal(err)
		}

		clock.Advance(time.Minute)

		if err := fin.DeleteAccount(ctx, acc); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreAccount(ctx, acc)
		if err != nil {
			t.Fatal(err)
		}

		if got.DeletedAt.Valid || got.Version != 3 {
			t.Fatalf("restored account = %+v, want live at version 3 (create, delete, restore)", got)
		}

		rows, err := fin.Transactions(ctx, finance.TxFilter{})
		if err != nil || len(rows) != 1 || rows[0].Description != "deleted with the account" {
			t.Fatalf("transactions = %+v, %v; want only the one deleted with the account", rows, err)
		}
	})

	t.Run("a transaction needs its account alive", func(t *testing.T) {
		tx := newTx("orphan")

		if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
			t.Fatal(err)
		}

		clock.Advance(time.Second)

		if err := fin.DeleteAccount(ctx, acc); err != nil {
			t.Fatal(err)
		}

		if _, err := fin.RestoreTransaction(ctx, tx.ID); err == nil {
			t.Fatal("restoring a transaction under a deleted account must fail")
		}

		if _, err := fin.RestoreAccount(ctx, acc); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreTransaction(ctx, tx.ID)
		if err != nil || got.DeletedAt.Valid {
			t.Fatalf("restore = %+v, %v; want it live", got, err)
		}
	})

	t.Run("a category and its budget come back unless the name was taken meanwhile", func(t *testing.T) {
		cat, err := fin.CreateCategory(ctx, "Pets", models.KindExpense)
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteCategory(ctx, cat.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := fin.CreateCategory(ctx, "Pets", models.KindExpense); err != nil {
			t.Fatal(err)
		}

		if _, err := fin.RestoreCategory(ctx, cat.ID); !isConflict(err) {
			t.Fatalf("restoring over a live name = %v, want the 409 conflict", err)
		}

		other, err := fin.CreateCategory(ctx, "Plants", models.KindExpense)
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteCategory(ctx, other.ID); err != nil {
			t.Fatal(err)
		}

		got, err := fin.RestoreCategory(ctx, other.ID)
		if err != nil || got.DeletedAt.Valid || got.Version != 3 {
			t.Fatalf("restore = %+v, %v; want live at version 3", got, err)
		}
	})

	t.Run("the window closes after 30 days", func(t *testing.T) {
		tx := newTx("late")

		if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
			t.Fatal(err)
		}

		clock.Advance(finance.RestoreWindow + time.Hour)

		if _, err := fin.RestoreTransaction(ctx, tx.ID); err == nil {
			t.Fatal("a restore after the window must fail")
		}
	})

	t.Run("a row that was never deleted is not restorable", func(t *testing.T) {
		tx := newTx("alive")

		if _, err := fin.RestoreTransaction(ctx, tx.ID); err == nil {
			t.Fatal("restoring a live row must fail")
		}
	})
}

// TestReturnedVersionIsTheStoredOne covers the stale-struct hazard: the
// database bumps version in a trigger, so every write re-reads the row.
func TestReturnedVersionIsTheStoredOne(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	tx, err := fin.CreateTransaction(ctx, finance.TxInput{AccountID: acc, Kind: models.KindExpense, Amount: "1.00", Date: today()})
	if err != nil {
		t.Fatal(err)
	}

	if tx.Version != 1 {
		t.Fatalf("created version = %d, want 1", tx.Version)
	}

	for want := int64(2); want <= 3; want++ {
		tx, err = fin.UpdateTransaction(ctx, tx.ID, finance.TxInput{
			AccountID: acc, Kind: models.KindExpense, Amount: strconv.FormatInt(want, 10) + ".00", Description: "edit", Date: today(),
		})
		if err != nil {
			t.Fatal(err)
		}

		stored := firstOf[models.Transaction](t, app, ana, "")
		if tx.Version != want || stored.Version != want {
			t.Fatalf("returned version %d, stored %d, want %d", tx.Version, stored.Version, want)
		}
	}
}
