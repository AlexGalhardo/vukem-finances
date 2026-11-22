package reports_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

const perfRows = 100000

// TestReportsPerformance runs every report over 100000 transactions spread
// over two years, fifty categories, tags and several accounts, and wants each
// under a second. The race detector slows the run by an order of magnitude, so
// there only the answers are checked.
func TestReportsPerformance(t *testing.T) {
	t.Parallel()

	f := newFx(t)
	savings := f.newAccount("Savings", models.AccountSavings, "0")
	trip := mustTag(t, f, "trip")

	var cats []int64

	for i := range 50 {
		cats = append(cats, f.category(fmt.Sprintf("Perf %02d", i), "expense", 0).ID)
	}

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db.SQL().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	stmt, err := tx.PrepareContext(t.Context(), `INSERT INTO transactions
		(workspace_id, uid, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'perf', ?, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	var wantExpense int64

	for i := range perfRows {
		account, kind, cents := f.account.ID, models.KindExpense, int64(100+i%900)

		if i%2 == 1 {
			account = savings.ID
		}

		var category any = cats[i%len(cats)]

		if i%10 == 0 {
			kind, category = models.KindIncome, nil
		} else {
			wantExpense += cents
		}

		date := fmt.Sprintf("%d-%02d-%02d", 2025+i%2, 1+i%12, 1+i%28)

		if _, err := stmt.ExecContext(t.Context(), f.ws.ID, fmt.Sprintf("PERF%022d", i), account, category, kind, cents, date); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := tx.ExecContext(t.Context(), `INSERT INTO transaction_tags (workspace_id, transaction_id, tag_id, created_at, updated_at)
		SELECT workspace_id, id, ?, datetime('now'), datetime('now') FROM transactions WHERE workspace_id = ? AND id % 5 = 0`,
		trip, f.ws.ID); err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	p := f.period("2025-01-01", "2026-12-31")

	timed := func(name string, run func() error) {
		t.Helper()

		start := time.Now()
		if err := run(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		took := time.Since(start)
		t.Logf("%s: %s", name, took)

		if !raceEnabled && took > time.Second {
			t.Errorf("%s took %s, the limit is 1s", name, took)
		}
	}

	timed("CashFlow", func() error {
		got, err := f.rep.CashFlow(f.ctx, p)
		if err == nil && (len(got.Months) != 24 || got.Totals.ExpenseCents != wantExpense) {
			t.Errorf("CashFlow expense = %d over %d months, want %d over 24", got.Totals.ExpenseCents, len(got.Months), wantExpense)
		}

		return err
	})
	timed("ByCategory", func() error {
		got, err := f.rep.ByCategory(f.ctx, p, "expense")
		if err == nil && (got.TotalCents != wantExpense || len(got.Rows) != 45) {
			t.Errorf("ByCategory total = %d in %d rows, want %d in 45 (the multiples of ten are income)", got.TotalCents, len(got.Rows), wantExpense)
		}

		return err
	})
	timed("ByAccount", func() error {
		got, err := f.rep.ByAccount(f.ctx, p)
		if err == nil && len(got.Rows) != 2 {
			t.Errorf("ByAccount rows = %d", len(got.Rows))
		}

		return err
	})
	timed("ByTag", func() error {
		got, err := f.rep.ByTag(f.ctx, p)
		if err == nil && (len(got.Rows) != 1 || got.Rows[0].ExpenseCents == 0) {
			t.Errorf("ByTag = %+v", got.Rows)
		}

		return err
	})
	timed("NetWorthSeries", func() error {
		got, err := f.rep.NetWorthSeries(f.ctx, p)
		if err == nil && len(got.Points) != 24 {
			t.Errorf("NetWorthSeries points = %d", len(got.Points))
		}

		return err
	})
	timed("Compare", func() error {
		_, err := f.rep.Compare(f.ctx, reports.Period{Preset: reports.Custom, From: "2026-01-01", To: "2026-12-31"})

		return err
	})
}

func mustTag(t *testing.T, f *fx, name string) int64 {
	t.Helper()

	tag, err := f.fin.CreateTag(f.ctx, name, "")
	if err != nil {
		t.Fatal(err)
	}

	return tag.ID
}
