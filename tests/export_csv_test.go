package tests

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/export"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

// record adds a transaction through the service, with a description of choice.
func record(t *testing.T, app *vukemtest.App, w who, account, category int64, kind, amount, date, description string, tags ...string) models.Transaction {
	t.Helper()

	row, err := softService(t, app).CreateTransaction(wsCtx(t, w), finance.TxInput{
		AccountID: account, CategoryID: category, Kind: kind, Amount: amount, Description: description, Date: date, Tags: tags,
	})
	if err != nil {
		t.Fatalf("record %s %s: %v", kind, amount, err)
	}

	return row
}

func categoryOf(t *testing.T, app *vukemtest.App, w who, kind string) models.Category {
	t.Helper()

	return firstOfWhere[models.Category](t, app, w, "kind", kind)
}

func csvRecords(t *testing.T, body string) [][]string {
	t.Helper()

	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(body, "\xEF\xBB\xBF")))
	r.FieldsPerRecord = -1

	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("not CSV: %v\n%s", err, body)
	}

	return rows
}

func TestCSVExportDownload(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	bob := signUp(t, app, "Bob", "bob@example.com")
	bobAcc := firstAccount(t, app, bob)
	food := categoryOf(t, app, ana, models.KindExpense)

	record(t, app, ana, acc, food.ID, "expense", "12.34", "2026-02-03", `=HYPERLINK("http://example.test")`, "trip")
	record(t, app, ana, acc, 0, "income", "100.00", "2026-02-04", "Pay, with \"quotes\"")
	record(t, app, ana, acc, 0, "expense", "1.00", "2026-03-04", "-5 refund")
	record(t, app, bob, bobAcc, 0, "expense", "777.77", "2026-02-05", "BOB SECRET")

	res := ana.Client.Get("/transactions/export.csv").AssertOK()
	res.AssertHeaderContains("Content-Type", "text/csv")
	res.AssertHeader("X-Content-Type-Options", "nosniff")
	res.AssertHeaderContains("Content-Disposition", "attachment")
	res.AssertHeader("Content-Disposition", `attachment; filename="vukem-finances-transactions-`+today()+`.csv"`)

	body := res.Body()
	if !strings.Contains(body, "\r\n") || strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") && !strings.Contains(body, "\"") {
		t.Errorf("lines must end with \\r\\n: %q", body)
	}

	if strings.Contains(body, "BOB SECRET") || strings.Contains(body, "777.77") {
		t.Fatal("the export of one workspace holds the data of another")
	}

	rows := csvRecords(t, body)
	if len(rows) != 4 || strings.Join(rows[0], ",") != "uid,date,account,category,tags,kind,amount,description,transfer_uid" {
		t.Fatalf("rows = %v", rows)
	}

	first := rows[1]
	if first[1] != "2026-02-03" || first[3] != food.Name || first[4] != "trip" || first[5] != "expense" || first[6] != "-12.34" ||
		first[7] != `'=HYPERLINK("http://example.test")` || first[8] != "" || len(first[0]) != 26 {
		t.Errorf("first row = %q", first)
	}

	if rows[2][6] != "100.00" || rows[2][7] != `Pay, with "quotes"` {
		t.Errorf("second row = %q", rows[2])
	}

	if rows[3][7] != "'-5 refund" {
		t.Errorf("the description starting with a minus is not guarded: %q", rows[3][7])
	}

	t.Run("it applies the filters of the list", func(t *testing.T) {
		got := csvRecords(t, ana.Client.Get("/transactions/export.csv?kind=income").AssertOK().Body())
		if len(got) != 2 || got[1][5] != "income" {
			t.Errorf("kind=income gave %v", got)
		}

		got = csvRecords(t, ana.Client.Get("/transactions/export.csv?from=2026-03-01&to=2026-03-31").AssertOK().Body())
		if len(got) != 2 || got[1][1] != "2026-03-04" {
			t.Errorf("a period gave %v", got)
		}

		got = csvRecords(t, ana.Client.Get("/transactions/export.csv?q=quotes").AssertOK().Body())
		if len(got) != 2 {
			t.Errorf("a search gave %v", got)
		}
	})

	t.Run("a deleted transaction is not exported", func(t *testing.T) {
		gone := record(t, app, ana, acc, 0, "expense", "9.99", "2026-04-01", "DELETED ROW")
		if err := softService(t, app).DeleteTransaction(wsCtx(t, ana), gone.ID); err != nil {
			t.Fatal(err)
		}

		ana.Client.Get("/transactions/export.csv").AssertOK().AssertDontSee("DELETED ROW")
	})

	t.Run("bom and delimiter options", func(t *testing.T) {
		got := ana.Client.Get("/transactions/export.csv?bom=1&delimiter=semicolon").AssertOK().Body()
		if !strings.HasPrefix(got, "\xEF\xBB\xBFuid;date;account;") {
			t.Errorf("want a BOM and semicolons: %q", got[:40])
		}

		ana.Client.Get("/transactions/export.csv?delimiter=tab").AssertUnprocessable()
	})

	t.Run("a transfer exports both legs with the same transfer uid", func(t *testing.T) {
		extra := insertAccount(t, app, ana, "Extra", 0)

		if _, err := softService(t, app).CreateTransfer(wsCtx(t, ana), finance.TransferInput{
			FromAccountID: acc, ToAccountID: extra.ID, Amount: "7.00", Description: "move", Date: "2026-05-01",
		}); err != nil {
			t.Fatal(err)
		}

		got := csvRecords(t, ana.Client.Get("/transactions/export.csv?from=2026-05-01&to=2026-05-01").AssertOK().Body())
		if len(got) != 3 || got[1][8] == "" || got[1][8] != got[2][8] || got[1][6] != "-7.00" && got[2][6] != "-7.00" {
			t.Errorf("transfer rows = %v", got)
		}
	})
}

func TestCSVStreamsAllChunks(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	insertRows(t, app, ana, acc, 2500)

	fin := softService(t, app)

	rows, err := export.Transactions(wsCtx(t, ana), fin, finance.TxFilter{})
	if err != nil {
		t.Fatal(err)
	}

	n, last := 0, ""

	for r, err := range rows {
		if err != nil {
			t.Fatal(err)
		}

		key := r.Date + r.UID
		if key <= last {
			t.Fatalf("row %d (%s) is not after %s: the order is date then id", n, key, last)
		}

		last = key
		n++
	}

	if n != 2500 {
		t.Errorf("%d rows over three reads of 1000, want 2500", n)
	}
}

// insertRows adds n expenses straight to the table, fast, for the volume tests.
func insertRows(t *testing.T, app *vukemtest.App, w who, account int64, n int) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db.SQL().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	stmt, err := tx.PrepareContext(t.Context(), `INSERT INTO transactions
		(workspace_id, uid, account_id, kind, amount_cents, description, occurred_on, created_at, updated_at)
		VALUES (?, ?, ?, 'expense', 100, 'bulk', ?, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	for i := range n {
		date := fmt.Sprintf("2025-%02d-%02d", 1+i%12, 1+i%28)
		if _, err := stmt.ExecContext(t.Context(), w.Workspace.ID, fmt.Sprintf("BULK%022d", i), account, date); err != nil {
			t.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestCSVCap(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	insertRows(t, app, ana, acc, export.MaxCSVRows+1)

	res := ana.Client.Get("/transactions/export.csv").AssertStatus(http.StatusRequestEntityTooLarge)
	if strings.Contains(res.Body(), "uid,date,account") {
		t.Error("a refused export started to stream")
	}

	// Narrowing the filter brings the result under the cap.
	ana.Client.Get("/transactions/export.csv?from=2025-01-01&to=2025-01-31").AssertOK()
}

func TestCSVRoundTrip(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	food := categoryOf(t, app, ana, models.KindExpense)
	pay := categoryOf(t, app, ana, models.KindIncome)

	record(t, app, ana, acc, pay.ID, "income", "2500.00", "2026-01-05", "Salary")
	record(t, app, ana, acc, food.ID, "expense", "82.40", "2026-01-06", "Market")
	record(t, app, ana, acc, food.ID, "expense", "0.05", "2026-01-07", "Gum, \"mint\"")
	record(t, app, ana, acc, 0, "expense", "10.00", "2026-02-01", "-dash first")
	record(t, app, ana, acc, 0, "income", "1234567.89", "2026-02-02", "Big one")

	wide := reports.Period{Preset: reports.Custom, From: "2020-01-01", To: "2030-12-31"}
	rep := reports.New(softService(t, app))

	want, err := rep.CashFlow(wsCtx(t, ana), wide)
	if err != nil {
		t.Fatal(err)
	}

	wantMove := movementOf(t, app, ana, acc)

	for i, query := range []string{"", "?bom=1&delimiter=semicolon"} {
		csvText := ana.Client.Get("/transactions/export.csv" + query).AssertOK().Body()

		ben := signUp(t, app, "Ben", fmt.Sprintf("ben%d@example.com", i))
		makePro(t, app, ben)

		mapping := finance.Mapping{
			Date: "date", Description: "description", Amount: "amount", Category: "category", DateFormat: "2006-01-02",
		}

		imp := runImport(t, app, ben, firstAccount(t, app, ben), "export.csv", []byte(csvText), mapping)
		if imp.Status != models.ImportDone || imp.Imported != 5 || imp.Failed != 0 {
			t.Fatalf("variant %q: import = %+v", query, imp)
		}

		got, err := reports.New(softService(t, app)).CashFlow(wsCtx(t, ben), wide)
		if err != nil {
			t.Fatal(err)
		}

		if got.Totals != want.Totals {
			t.Errorf("variant %q: totals = %+v, want %+v", query, got.Totals, want.Totals)
		}

		if move := movementOf(t, app, ben, firstAccount(t, app, ben)); move != wantMove {
			t.Errorf("variant %q: balance movement = %d, want %d", query, move, wantMove)
		}

		// Categories came across by name.
		if n := countWhere(t, app, ben, "category_id IS NOT NULL"); n != 3 {
			t.Errorf("variant %q: %d categorized rows, want 3", query, n)
		}
	}
}

func movementOf(t *testing.T, app *vukemtest.App, w who, account int64) int64 {
	t.Helper()

	balances, err := softService(t, app).AccountBalances(wsCtx(t, w), "2030-12-31")
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range balances {
		if b.AccountID == account {
			return b.MovementCents
		}
	}

	t.Fatal("account not found")

	return 0
}

func countWhere(t *testing.T, app *vukemtest.App, w who, where string) int64 {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int64
	if err := db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM transactions WHERE workspace_id = ? AND deleted_at IS NULL AND "+where, w.Workspace.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}
