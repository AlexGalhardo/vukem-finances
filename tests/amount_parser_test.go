package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// There is one amount parser, finance.ParseAmount, and every path that reads money text goes through it: the
// forms, the CSV and OFX importers, and the filters. "1.234,56" and "1,234.56" are 123456 cents on all of them.
func TestOneAmountParser(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	for text, want := range map[string]int64{
		"1.234,56": 123456, "1,234.56": 123456, "1234.56": 123456, "1234,56": 123456, "12,5": 1250, "0.05": 5,
	} {
		if got, err := finance.ParseAmount(text); err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d", text, got, err, want)
		}
	}

	t.Run("the forms", func(t *testing.T) {
		for _, amount := range []string{"1.234,56", "1,234.56"} {
			ana.Client.PostJSON("/transactions", map[string]any{
				"accountId": idString(acc), "kind": "expense", "amount": amount, "description": "form " + amount, "date": "2026-07-01",
			}).AssertStatus(http.StatusCreated)
			vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": "form " + amount, "amount_cents": 123456})
		}

		ana.Client.PostJSON("/transactions", map[string]any{
			"accountId": idString(acc), "kind": "expense", "amount": "12.345", "description": "three decimals", "date": "2026-07-01",
		}).AssertStatus(http.StatusUnprocessableEntity)

		other, err := fin.CreateAccount(ctx, "Opening", "savings", "1.234,56")
		if err != nil || other.OpeningCents != 123456 {
			t.Errorf("opening balance = %+v, %v; want 123456", other, err)
		}

		cat := categoryID(t, app, ana, "Groceries")
		if err := fin.SetBudget(ctx, cat, "1.234,56"); err != nil {
			t.Fatal(err)
		}

		vukemtest.AssertDatabaseHas(t, app, "budgets", map[string]any{"category_id": cat, "limit_cents": 123456})

		tr, err := fin.CreateTransfer(ctx, finance.TransferInput{FromAccountID: acc, ToAccountID: other.ID, Amount: "1.234,56", Date: "2026-07-02"})
		if err != nil || tr.From.AmountCents != 123456 || tr.To.AmountCents != 123456 {
			t.Errorf("transfer = %+v, %v; want 123456 on both legs", tr, err)
		}
	})

	t.Run("CSV", func(t *testing.T) {
		csv := []byte("Data;Historico;Valor\n2026-07-03;pt-BR style;\"1.234,56\"\n2026-07-04;en style;\"1,234.56\"\n")
		m := finance.Mapping{Date: "Data", Description: "Historico", Amount: "Valor", Invert: true}

		if got := runImport(t, app, ana, acc, "amounts.csv", csv, m); got.Imported != 2 {
			t.Fatalf("import = %+v", got)
		}

		for _, d := range []string{"pt-BR style", "en style"} {
			vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": d, "amount_cents": 123456, "kind": "expense"})
		}
	})

	t.Run("OFX", func(t *testing.T) {
		ofx := "<OFX><STMTTRN><DTPOSTED>20260705<TRNAMT>-1.234,56<FITID>A1<NAME>ofx pt-BR</STMTTRN>" +
			"<STMTTRN><DTPOSTED>20260706<TRNAMT>-1,234.56<FITID>A2<NAME>ofx en</STMTTRN>" +
			"<STMTTRN><DTPOSTED>20260707<TRNAMT>-1234.56<FITID>A3<NAME>ofx spec</STMTTRN>" +
			"<STMTTRN><DTPOSTED>20260708<TRNAMT>-1.5e3<FITID>A4<NAME>ofx float</STMTTRN></OFX>"

		got := runImport(t, app, ana, acc, "amounts.ofx", []byte("OFXHEADER:100\n"+ofx), finance.Mapping{})
		if got.Imported != 3 || got.Failed != 1 {
			t.Fatalf("import = %+v, want 3 imported and the float refused", got)
		}

		for _, d := range []string{"ofx pt-BR", "ofx en", "ofx spec"} {
			vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": d, "amount_cents": 123456, "kind": "expense"})
		}

		vukemtest.AssertDatabaseMissing(t, app, "transactions", map[string]any{"description": "ofx float"})
	})

	t.Run("filters", func(t *testing.T) {
		for min, want := range map[string]int{"1.234,56": 1, "1,234.56": 1, "1.234,57": 0, "1234.57": 0} {
			rows, err := fin.Transactions(ctx, finance.TxFilter{Min: min, Q: "form"})
			if err != nil || len(rows) != want*2 {
				t.Errorf("min %q over the two form rows = %d, %v; want %d", min, len(rows), err, want*2)
			}
		}

		rows, err := fin.Transactions(ctx, finance.TxFilter{Min: "1.234,56", Max: "1,234.56", Q: "pt-BR"})
		if err != nil || len(rows) != 2 {
			t.Errorf("min and max at 123456 = %d rows, %v; want the CSV and OFX pt-BR rows", len(rows), err)
		}

		// Over HTTP, the query string is read the same way.
		ana.Client.Inertia("").Get("/transactions?q=form&min=1.234,56&max=1,234.56").AssertInertia("Transactions").HasProp("transactions", 2)
	})

	t.Run("a number beyond 2^53-1 cents is refused on every path", func(t *testing.T) {
		big := "90071992547409.92"

		if _, err := fin.CreateAccount(ctx, "Big", "savings", big); err == nil {
			t.Error("an opening balance over the cap was accepted")
		}

		if _, err := fin.Transactions(ctx, finance.TxFilter{Min: big}); err == nil {
			t.Error("a filter over the cap was accepted")
		}

		plain := browser(t, app, ana.User, ana.Workspace.ID) // the filters subtest turned ana.Client into an Inertia client
		res := plain.PostJSON("/transactions", map[string]any{
			"accountId": idString(acc), "kind": "expense", "amount": big, "description": "big", "date": "2026-07-01",
		})
		if res.Status() != http.StatusUnprocessableEntity {
			t.Errorf("a form amount over the cap = %d, want 422", res.Status())
		}

		csv := []byte("Date,Description,Amount\n2026-07-09," + strings.Repeat("big", 1) + "," + big + "\n2026-07-10,ok,1.00\n")
		if got := runImport(t, app, ana, acc, "big.csv", csv, csvMapping); got.Imported != 1 || got.Failed != 1 {
			t.Errorf("a CSV amount over the cap = %+v, want it reported as failed", got)
		}
	})

	if n := count[models.Transaction](t, app, ana); n == 0 {
		t.Fatal("nothing was recorded")
	}
}
