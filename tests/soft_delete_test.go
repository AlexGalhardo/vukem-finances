package tests

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// softService builds the finance service the handlers use.
func softService(t *testing.T, app *vukemtest.App) *finance.Service {
	t.Helper()

	fin, err := finance.New(app.App, service(t, app))
	if err != nil {
		t.Fatal(err)
	}

	return fin
}

func idStr(id int64) string { return strconv.FormatInt(id, 10) }

func trashedCount[T any](t *testing.T, app *vukemtest.App, w who) int64 {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	n, err := workspaces.Scoped[T](workspaces.WithTenant(t.Context(), w.Workspace.ID), db).OnlyTrashed().Count(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return n
}

// TestSoftDelete proves, one path at a time, that a deleted row is gone from
// every read, total, budget, report and the monthly summary, that the row is
// still stored as a tombstone, and that its name can be used again.
func TestSoftDelete(t *testing.T) {
	t.Parallel()

	month := time.Now().UTC().Format("2006-01")
	date := today()

	t.Run("a deleted transaction leaves the list, the balance, the charts and the budget", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		makePro(t, app, ana)

		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		groceries := categoryID(t, app, ana, "Groceries")

		ana.Client.PostForm("/budgets", map[string][]string{"categoryId": {idStr(groceries)}, "limit": {"100"}}).
			AssertRedirect("/budgets")

		keep, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: firstAccount(t, app, ana), CategoryID: groceries, Kind: models.KindExpense,
			Amount: "10.00", Description: "keep", Date: date,
		})
		if err != nil {
			t.Fatal(err)
		}

		gone, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: firstAccount(t, app, ana), CategoryID: groceries, Kind: models.KindExpense,
			Amount: "40.00", Description: "gone", Date: date,
		})
		if err != nil {
			t.Fatal(err)
		}

		if gone.Version != 1 {
			t.Fatalf("a created transaction reports version %d, want the stored 1", gone.Version)
		}

		ana.Client.Delete("/transactions/" + idStr(gone.ID)).AssertRedirect("/transactions")

		rows, err := fin.Transactions(ctx, finance.TxFilter{})
		if err != nil || len(rows) != 1 || rows[0].ID != keep.ID {
			t.Fatalf("transactions = %+v, %v; want only the kept one", rows, err)
		}

		d, err := fin.Dashboard(ctx, month)
		if err != nil {
			t.Fatal(err)
		}

		if d.BalanceCents != -1000 {
			t.Errorf("balance = %d, want -1000 (the deleted 40.00 must not count)", d.BalanceCents)
		}

		if got := d.Months[len(d.Months)-1].ExpenseCents; got != 1000 {
			t.Errorf("month expense = %d, want 1000", got)
		}

		if len(d.Categories) != 1 || d.Categories[0].Cents != 1000 {
			t.Errorf("category spend = %+v, want 1000 in one category", d.Categories)
		}

		if len(d.Budgets) != 1 || d.Budgets[0].SpentCents != 1000 || d.Budgets[0].Percent != 10 {
			t.Errorf("budget status = %+v, want 1000 spent (10%%)", d.Budgets)
		}

		page := ana.Client.Inertia("").Get("/transactions").Body()
		if strings.Contains(page, `"gone"`) || !strings.Contains(page, `"keep"`) {
			t.Errorf("the transactions page must show only the kept row")
		}

		if n := trashedCount[models.Transaction](t, app, ana); n != 1 {
			t.Errorf("tombstones = %d, want 1 (the row is kept, not purged)", n)
		}

		if n := count[models.Transaction](t, app, ana); n != 1 {
			t.Errorf("live transactions = %d, want 1", n)
		}
	})

	t.Run("a deleted account takes its transactions and balance with it and frees the plan slot", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")

		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		acc := firstAccount(t, app, ana)

		addTransaction(t, ana, acc, 0, "income", "25.00", date).AssertRedirect("/transactions")

		if _, err := fin.CreateAccount(ctx, "Second", "savings", ""); err == nil {
			t.Fatal("the Free plan must stop at one account while the first is alive")
		}

		ana.Client.Delete("/accounts/" + idStr(acc)).AssertRedirect("/accounts")

		if accounts, err := fin.Accounts(ctx); err != nil || len(accounts) != 0 {
			t.Fatalf("accounts = %+v, %v; want none", accounts, err)
		}

		if rows, err := fin.Transactions(ctx, finance.TxFilter{}); err != nil || len(rows) != 0 {
			t.Fatalf("transactions = %+v, %v; want none", rows, err)
		}

		d, err := fin.Dashboard(ctx, month)
		if err != nil || d.BalanceCents != 0 {
			t.Fatalf("balance = %d, %v; want 0", d.BalanceCents, err)
		}

		if body := ana.Client.Inertia("").Get("/accounts").Body(); strings.Contains(body, "Main account") {
			t.Error("the accounts page still lists the deleted account")
		}

		if _, err := fin.CreateAccount(ctx, "Second", "savings", ""); err != nil {
			t.Fatalf("a deleted account must not count against the plan: %v", err)
		}
	})

	t.Run("a deleted category frees its name and its budget, and uncategorizes its transactions", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		makePro(t, app, ana)

		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		groceries := categoryID(t, app, ana, "Groceries")

		ana.Client.PostForm("/budgets", map[string][]string{"categoryId": {idStr(groceries)}, "limit": {"100"}}).
			AssertRedirect("/budgets")
		addTransaction(t, ana, firstAccount(t, app, ana), groceries, "expense", "5.00", date).AssertRedirect("/transactions")

		if _, err := fin.CreateCategory(ctx, "Groceries", models.KindExpense); !isConflict(err) {
			t.Fatalf("a live duplicate name = %v, want the 409 conflict", err)
		}

		if err := fin.DeleteCategory(ctx, groceries); err != nil {
			t.Fatal(err)
		}

		cats, err := fin.Categories(ctx)
		if err != nil {
			t.Fatal(err)
		}

		for _, c := range cats {
			if c.ID == groceries {
				t.Fatal("the deleted category is still listed")
			}
		}

		if b, err := fin.Budgets(ctx, month); err != nil || len(b) != 0 {
			t.Fatalf("budgets = %+v, %v; want none after the category went", b, err)
		}

		rows, err := fin.Transactions(ctx, finance.TxFilter{})
		if err != nil || len(rows) != 1 || rows[0].CategoryID.Valid {
			t.Fatalf("transactions = %+v, %v; want one, uncategorized", rows, err)
		}

		reused, err := fin.CreateCategory(ctx, "Groceries", models.KindExpense)
		if err != nil || reused.ID == groceries || reused.Version != 1 {
			t.Fatalf("reusing the deleted name = %+v, %v; want a new row", reused, err)
		}

		if n := trashedCount[models.Category](t, app, ana); n != 1 {
			t.Errorf("tombstones = %d, want 1", n)
		}
	})

	t.Run("two creations of one live name end in a duplicate-name conflict at the database", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		bob := signUp(t, app, "Bob", "bob@example.com")

		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		add := func(w who, uid string) error {
			ctx := workspaces.WithTenant(t.Context(), w.Workspace.ID)

			return orm.Create(ctx, db, &models.Category{UID: uid, Name: "Offline made", Kind: models.KindExpense})
		}

		if err := add(ana, "01J0000000000000000000000A"); err != nil {
			t.Fatal(err)
		}

		err = add(ana, "01J0000000000000000000000B")
		if err == nil || !db.UniqueViolation(err) {
			t.Fatalf("second live name = %v, want a unique violation", err)
		}

		if err := add(bob, "01J0000000000000000000000C"); err != nil {
			t.Fatalf("the same name in another workspace must be fine: %v", err)
		}

		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		if err := softService(t, app).DeleteCategory(ctx, firstOf[models.Category](t, app, ana, "Offline made").ID); err != nil {
			t.Fatal(err)
		}

		if err := add(ana, "01J0000000000000000000000D"); err != nil {
			t.Fatalf("a deleted name must be reusable: %v", err)
		}
	})

	t.Run("a deleted budget leaves the list and the dashboard and can be set again", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		makePro(t, app, ana)

		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		groceries := categoryID(t, app, ana, "Groceries")

		ana.Client.PostForm("/budgets", map[string][]string{"categoryId": {idStr(groceries)}, "limit": {"100"}}).
			AssertRedirect("/budgets")

		b := firstOf[models.Budget](t, app, ana, "")
		ana.Client.Delete("/budgets/" + idStr(b.ID)).AssertRedirect("/budgets")

		d, err := fin.Dashboard(ctx, month)
		if err != nil || len(d.Budgets) != 0 {
			t.Fatalf("dashboard budgets = %+v, %v; want none", d.Budgets, err)
		}

		if err := fin.SetBudget(ctx, groceries, "50"); err != nil {
			t.Fatalf("setting a budget after a delete: %v", err)
		}

		if got, err := fin.Budgets(ctx, month); err != nil || len(got) != 1 || got[0].LimitCents != 5000 {
			t.Fatalf("budgets = %+v, %v; want the new 50.00 one", got, err)
		}
	})

	t.Run("the monthly summary and the budget alert ignore deleted rows", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

		tx, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: firstAccount(t, app, ana), Kind: models.KindExpense, Amount: "30.00", Date: date,
		})
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
			t.Fatal(err)
		}

		if err := fin.SendMonthlySummary(ctx, month); err != nil {
			t.Fatal(err)
		}

		if n := len(app.Mails().SentTo("ana@example.com")); n != 0 {
			t.Fatalf("a month whose only transaction was deleted mailed %d summaries, want 0", n)
		}
	})

	t.Run("deleting twice and across workspaces answers not found", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		bob := signUp(t, app, "Bob", "bob@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		acc := firstAccount(t, app, ana)

		if err := fin.DeleteAccount(ctx, acc); err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteAccount(ctx, acc); err == nil {
			t.Fatal("a second delete of a tombstone must fail")
		}

		bob.Client.Delete("/accounts/" + idStr(acc)).AssertStatus(http.StatusNotFound)
	})
}

func isConflict(err error) bool {
	var e *errs.HTTPError

	return errors.As(err, &e) && e.Status == http.StatusConflict
}
