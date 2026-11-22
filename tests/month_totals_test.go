package tests

import (
	"context"
	"fmt"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// monthTotalsSnapshot renders the summary of one workspace as the reader sees it.
func monthTotalsSnapshot(t *testing.T, fin *finance.Service, ws int64) string {
	t.Helper()

	rows, err := fin.MonthTotals(workspaces.WithTenant(t.Context(), ws), finance.FirstSummaryDate, "9999-12-31", finance.ByMonth|finance.ByAccount|finance.ByCategory)
	if err != nil {
		t.Fatal(err)
	}

	var out string

	for _, r := range rows {
		out += fmt.Sprintln(r.Month, r.AccountID, r.CategoryID, r.Kind, r.Transfer, r.N, r.Sum())
	}

	return out
}

// TestMonthTotalsEqualARecomputation is QA-503: after random creates, edits,
// soft deletes, restores, transfers, installments, imports and hard deletes in
// one workspace, the summary equals a recomputation from the transactions, the
// range reader equals a direct sum for random ranges (whole months and
// edges), and another workspace's summary did not move.
func TestMonthTotalsEqualARecomputation(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	bobCtx := workspaces.WithTenant(t.Context(), bob.Workspace.ID)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	accounts := []int64{firstAccount(t, app, ana)}
	for _, name := range []string{"Savings", "Card", "Cash"} {
		accounts = append(accounts, insertAccount(t, app, ana, name, 1000).ID)
	}

	var categories []int64

	for i, name := range []string{"Food", "Rent", "Fun"} {
		kind := models.KindExpense
		if i == 2 {
			kind = models.KindIncome
		}

		c, err := fin.CreateCategory(ctx, name, kind)
		if err != nil {
			t.Fatal(err)
		}

		categories = append(categories, c.ID)
	}

	bobAcc := firstAccount(t, app, bob)
	for i := range 5 {
		insertTx(t, app, bob, bobAcc, models.KindExpense, int64(100+i), fmt.Sprintf("2026-0%d-10", 1+i))
	}

	bobBefore := monthTotalsSnapshot(t, fin, bob.Workspace.ID)
	if bobBefore == "" {
		t.Fatal("bob has no summary to compare")
	}

	rng := rand.New(rand.NewPCG(3, 5))
	pick := func(ids []int64) int64 { return ids[rng.IntN(len(ids))] }

	date := func() string {
		return fmt.Sprintf("%d-%02d-%02d", 2025+rng.IntN(2), 1+rng.IntN(12), 1+rng.IntN(28))
	}

	amount := func() string {
		if rng.IntN(15) == 0 {
			return "90071992547409.91"
		}

		return fmt.Sprintf("%d.%02d", 1+rng.IntN(5000), rng.IntN(100))
	}

	category := func() int64 {
		if rng.IntN(3) == 0 {
			return 0
		}

		return pick(categories)
	}

	randomID := func(where string) int64 {
		var id int64

		row := db.QueryRowContext(t.Context(),
			"SELECT id FROM transactions WHERE workspace_id = ? AND "+where+" ORDER BY random() LIMIT 1", ana.Workspace.ID)
		if row.Scan(&id) != nil {
			return 0
		}

		return id
	}

	ran := map[string]int{}

	try := func(name string, err error) {
		t.Helper()

		if err == nil {
			ran[name]++
		}
	}

	noDispatch := func(c context.Context, _, id int64) error { return fin.RunImport(c, id) }

	for step := range 400 {
		kind := [2]string{models.KindIncome, models.KindExpense}[rng.IntN(2)]

		switch op := rng.IntN(11); op {
		case 0, 1, 2:
			_, err := fin.CreateTransaction(ctx, finance.TxInput{
				AccountID: pick(accounts), CategoryID: category(), Kind: kind, Amount: amount(), Description: "x", Date: date(),
			})
			try("create", err)
		case 3:
			if id := randomID("deleted_at IS NULL AND transfer_uid IS NULL"); id != 0 {
				_, err := fin.UpdateTransaction(ctx, id, finance.TxInput{
					AccountID: pick(accounts), CategoryID: category(), Kind: kind, Amount: amount(), Description: "y", Date: date(),
				})
				try("edit", err)
			}
		case 4:
			if id := randomID("deleted_at IS NULL AND transfer_uid IS NULL"); id != 0 {
				try("delete", fin.DeleteTransaction(ctx, id))
			}
		case 5:
			if id := randomID("deleted_at IS NOT NULL"); id != 0 {
				_, err := fin.RestoreTransaction(ctx, id)
				try("restore", err)
			}
		case 6:
			from, to := pick(accounts), pick(accounts)
			_, err := fin.CreateTransfer(ctx, finance.TransferInput{FromAccountID: from, ToAccountID: to, Amount: amount(), Date: date()})
			try("transfer", err)
		case 7:
			if id := randomID("deleted_at IS NULL AND transfer_uid IS NOT NULL AND kind = 'expense'"); id != 0 {
				try("delete transfer", fin.DeleteTransfer(ctx, id))
			}
		case 8:
			_, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
				AccountID: pick(accounts), CategoryID: category(), Total: "300.00", Parts: 2 + rng.IntN(4), FirstDate: date(), Description: "tv",
			})
			try("installments", err)
		case 9:
			if id := randomID("deleted_at IS NULL AND installment_group_uid IS NOT NULL"); id != 0 {
				_, err := fin.DeleteInstallments(ctx, id, rng.IntN(2) == 0, true)
				try("delete installments", err)
			}
		default:
			csv := fmt.Sprintf("date,description,amount\n%s,Shop %d,%d.50\n%s,Shop %d,%d.25\n", date(), step, step+1, date(), step, step+2)

			if p, err := fin.StartImport(ctx, pick(accounts), "bank.csv", []byte(csv)); err == nil {
				err = fin.ConfirmImport(ctx, p.ImportID, finance.Mapping{Date: "date", Description: "description", Amount: "amount"}, noDispatch)
				try("import", err)
			}

			if id := randomID("deleted_at IS NOT NULL"); id != 0 && rng.IntN(2) == 0 {
				if _, err := db.ExecContext(t.Context(), "DELETE FROM transactions WHERE id = ?", id); err == nil {
					ran["hard delete"]++
				}
			}
		}

		if step%20 == 0 {
			if err := finance.CheckMonthTotals(t.Context(), db); err != nil {
				t.Fatalf("step %d: %v", step, err)
			}
		}
	}

	for _, name := range []string{"create", "edit", "delete", "restore", "transfer", "delete transfer", "installments", "delete installments", "import", "hard delete"} {
		if ran[name] == 0 {
			t.Errorf("the random run never completed a %q: the test would prove nothing for it (%v)", name, ran)
		}
	}

	if err := finance.CheckMonthTotals(t.Context(), db); err != nil {
		t.Fatal(err)
	}

	if got := monthTotalsSnapshot(t, fin, bob.Workspace.ID); got != bobBefore {
		t.Errorf("another workspace's summary moved:\n%s\nwas\n%s", got, bobBefore)
	}

	if _, err := fin.MonthTotals(bobCtx, "2026-01-01", "2026-01-31", 0); err != nil {
		t.Fatal(err)
	}

	checkRanges(t, app, fin, ana, rng)
}

// checkRanges compares the range reader with a direct sum of the live rows,
// for ranges that are whole months, partial months and both.
func checkRanges(t *testing.T, app *vukemtest.App, fin *finance.Service, w who, rng *rand.Rand) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), w.Workspace.ID)

	ranges := [][2]string{
		{"2025-01-01", "2026-12-31"}, {"2025-03-01", "2025-05-31"}, {"2025-03-15", "2025-05-31"}, {"2025-03-01", "2025-05-14"},
		{"2025-03-15", "2025-03-20"}, {"2025-02-10", "2025-04-20"}, {"2025-12-31", "2026-01-01"}, {"2026-02-01", "2026-02-28"},
	}

	for range 20 {
		a, b := fmt.Sprintf("202%d-%02d-%02d", 5+rng.IntN(2), 1+rng.IntN(12), 1+rng.IntN(28)), fmt.Sprintf("202%d-%02d-%02d", 5+rng.IntN(2), 1+rng.IntN(12), 1+rng.IntN(28))
		if a > b {
			a, b = b, a
		}

		ranges = append(ranges, [2]string{a, b})
	}

	for _, r := range ranges {
		got, err := fin.MonthTotals(ctx, r[0], r[1], 0)
		if err != nil {
			t.Fatal(err)
		}

		gotSum, gotN := map[string]*big.Int{}, map[string]int64{}

		for _, row := range got {
			k := fmt.Sprint(row.Kind, row.Transfer)
			if gotSum[k] == nil {
				gotSum[k] = new(big.Int)
			}

			gotSum[k].Add(gotSum[k], row.Sum())
			gotN[k] += row.N
		}

		rows, err := db.QueryContext(t.Context(),
			"SELECT kind, transfer_uid IS NOT NULL, COUNT(*), SUM(amount_cents) FROM transactions "+
				"WHERE workspace_id = ? AND deleted_at IS NULL AND occurred_on BETWEEN ? AND ? GROUP BY 1, 2", w.Workspace.ID, r[0], r[1])
		if err != nil {
			t.Fatal(err)
		}

		seen := 0

		for rows.Next() {
			var (
				kind        string
				transfer, n int64
				cents       int64
			)

			if err := rows.Scan(&kind, &transfer, &n, &cents); err != nil {
				t.Fatal(err)
			}

			k := fmt.Sprint(kind, transfer)
			seen++

			if gotN[k] != n || gotSum[k] == nil || gotSum[k].Cmp(big.NewInt(cents)) != 0 {
				t.Errorf("range %v %s: reader %v rows %v cents, direct %d rows %d cents", r, k, gotN[k], gotSum[k], n, cents)
			}
		}

		_ = rows.Close()

		if seen != len(gotSum) {
			t.Errorf("range %v: reader has %d groups, direct %d", r, len(gotSum), seen)
		}
	}
}
