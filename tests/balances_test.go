package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// insertAccount stores an account directly, outside the plan quota, so a test
// can hold as many as it needs.
func insertAccount(t *testing.T, app *vukemtest.App, w who, name string, opening int64) models.Account {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	a := models.Account{Name: name, Kind: models.AccountChecking, OpeningCents: opening}
	if err := orm.Create(workspaces.WithTenant(t.Context(), w.Workspace.ID), db, &a); err != nil {
		t.Fatal(err)
	}

	return a
}

func insertTx(t *testing.T, app *vukemtest.App, w who, account int64, kind string, cents int64, date string) models.Transaction {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	tx := models.Transaction{AccountID: account, Kind: kind, AmountCents: cents, Description: "t", OccurredOn: date}
	if err := orm.Create(workspaces.WithTenant(t.Context(), w.Workspace.ID), db, &tx); err != nil {
		t.Fatal(err)
	}

	return tx
}

func balanceOf(t *testing.T, bs []finance.AccountBalance, account int64) finance.AccountBalance {
	t.Helper()

	for _, b := range bs {
		if b.AccountID == account {
			return b
		}
	}

	t.Fatalf("no balance for account %d in %+v", account, bs)

	return finance.AccountBalance{}
}

func daysAgo(n int) string { return time.Now().UTC().AddDate(0, 0, -n).Format("2006-01-02") }

func TestBalances(t *testing.T) {
	t.Parallel()

	t.Run("a balance is the opening plus income minus expense up to the as-of date", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		acc := insertAccount(t, app, ana, "Main", 10000)

		insertTx(t, app, ana, acc.ID, models.KindIncome, 5000, daysAgo(10))
		insertTx(t, app, ana, acc.ID, models.KindExpense, 1234, daysAgo(5))
		insertTx(t, app, ana, acc.ID, models.KindExpense, 100, daysAgo(0))

		for asOf, want := range map[string]int64{
			daysAgo(11): 10000,
			daysAgo(10): 15000,
			daysAgo(6):  15000,
			daysAgo(5):  13766,
			"":          13666,
			daysAgo(0):  13666,
		} {
			bs, err := fin.AccountBalances(ctx, asOf)
			if err != nil {
				t.Fatal(err)
			}

			if got := balanceOf(t, bs, acc.ID); got.BalanceCents != want || got.OpeningCents != 10000 {
				t.Errorf("as of %q: %+v, want balance %d", asOf, got, want)
			}
		}

		if _, err := fin.AccountBalances(ctx, "yesterday"); err == nil {
			t.Error("an as-of date that is not an ISO date was accepted")
		}
	})

	t.Run("archived and deleted accounts and transactions", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		seed := firstOf[models.Account](t, app, ana, "")
		live := insertAccount(t, app, ana, "Live", 1000)
		old := insertAccount(t, app, ana, "Old", 7000)
		gone := insertAccount(t, app, ana, "Gone", 99999)
		removed := insertTx(t, app, ana, live.ID, models.KindIncome, 400, daysAgo(1))
		insertTx(t, app, ana, old.ID, models.KindExpense, 500, daysAgo(1))

		if err := fin.DeleteTransaction(ctx, removed.ID); err != nil {
			t.Fatal(err)
		}

		if err := fin.DeleteAccount(ctx, gone.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := fin.ArchiveAccount(ctx, old.ID); err != nil {
			t.Fatal(err)
		}

		bs, err := fin.AccountBalances(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		for _, b := range bs {
			if b.AccountID == gone.ID {
				t.Errorf("a deleted account has a balance: %+v", b)
			}
		}

		if b := balanceOf(t, bs, live.ID); b.BalanceCents != 1000 || b.Archived {
			t.Errorf("live = %+v, want 1000 (the deleted income must not count)", b)
		}

		// An archived account keeps its history and its balance, but is not net worth.
		if b := balanceOf(t, bs, old.ID); b.BalanceCents != 6500 || !b.Archived {
			t.Errorf("archived = %+v, want 6500 and the archived flag", b)
		}

		net, err := fin.NetWorth(ctx, "")
		if err != nil || net != 1000+seed.OpeningCents {
			t.Errorf("NetWorth = %d, %v; want the live accounts only", net, err)
		}
	})

	t.Run("the dashboard total is the net worth", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		seed := firstOf[models.Account](t, app, ana, "")
		a := insertAccount(t, app, ana, "A", 2500)
		b := insertAccount(t, app, ana, "B", -300)

		insertTx(t, app, ana, a.ID, models.KindIncome, 1000, daysAgo(3))
		insertTx(t, app, ana, b.ID, models.KindExpense, 450, daysAgo(2))
		insertTx(t, app, ana, b.ID, models.KindIncome, 90, daysAgo(0))

		// A future entry is not part of today's balance.
		insertTx(t, app, ana, a.ID, models.KindExpense, 777, daysAgo(-30))

		if _, err := fin.ArchiveAccount(ctx, b.ID); err != nil {
			t.Fatal(err)
		}

		d, err := fin.Dashboard(ctx, fin.Month(ctx))
		if err != nil {
			t.Fatal(err)
		}

		net, err := fin.NetWorth(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		if want := seed.OpeningCents + 2500 + 1000; d.BalanceCents != net || net != want {
			t.Errorf("dashboard %d, net worth %d; want both %d (non-archived accounts only)", d.BalanceCents, net, want)
		}
	})

	t.Run("balances equal a naive reference over random data", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		rng := rand.New(rand.NewPCG(7, 11))

		type entry struct {
			account int64
			income  bool
			cents   int64
			date    string
		}

		var accounts []models.Account

		for i := range 5 {
			accounts = append(accounts, insertAccount(t, app, ana, fmt.Sprintf("Acc %d", i), rng.Int64N(2_000_000)-1_000_000))
		}

		var entries []entry

		for range 400 {
			e := entry{
				account: accounts[rng.IntN(len(accounts))].ID, income: rng.IntN(2) == 0,
				cents: 1 + rng.Int64N(500_000), date: daysAgo(rng.IntN(120)),
			}
			entries = append(entries, e)

			kind := models.KindExpense
			if e.income {
				kind = models.KindIncome
			}

			insertTx(t, app, ana, e.account, kind, e.cents, e.date)
		}

		for _, asOf := range []string{"", daysAgo(0), daysAgo(30), daysAgo(77), daysAgo(200)} {
			cutoff := asOf
			if cutoff == "" {
				cutoff = daysAgo(0)
			}

			bs, err := fin.AccountBalances(ctx, asOf)
			if err != nil {
				t.Fatal(err)
			}

			for _, a := range accounts {
				want := a.OpeningCents

				for _, e := range entries {
					if e.account != a.ID || e.date > cutoff {
						continue
					}

					if e.income {
						want += e.cents
					} else {
						want -= e.cents
					}
				}

				if got := balanceOf(t, bs, a.ID).BalanceCents; got != want {
					t.Errorf("as of %q, %s: balance %d, reference %d", asOf, a.Name, got, want)
				}
			}
		}
	})

	t.Run("the largest amounts keep the sums exact", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		fin := softService(t, app)
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
		acc := insertAccount(t, app, ana, "Big", models.MaxCents)

		insertTx(t, app, ana, acc.ID, models.KindExpense, models.MaxCents, daysAgo(1))
		insertTx(t, app, ana, acc.ID, models.KindIncome, 1, daysAgo(1))

		bs, err := fin.AccountBalances(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		if got := balanceOf(t, bs, acc.ID).BalanceCents; got != 1 {
			t.Errorf("balance = %d, want 1", got)
		}
	})
}
