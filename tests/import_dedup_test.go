package tests

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("fixtures", "ofx", name))
	if err != nil {
		t.Fatal(err)
	}

	return b
}

// uploadFile posts a statement with the file name and Content-Type a browser would send.
func uploadFile(t *testing.T, w who, account int64, filename, contentType string, content []byte) *vukemtest.Response {
	t.Helper()

	var body bytes.Buffer

	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("accountId", strconv.FormatInt(account, 10)); err != nil {
		t.Fatal(err)
	}

	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))

	if contentType != "" {
		hdr.Set("Content-Type", contentType)
	}

	part, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	return w.Client.Do(http.MethodPost, "/import", &body, mw.FormDataContentType())
}

func wsCtx(t *testing.T, w who) context.Context {
	t.Helper()

	return workspaces.WithTenant(t.Context(), w.Workspace.ID)
}

func firstOfWhere[T any](t *testing.T, app *vukemtest.App, w who, column string, value any) T {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	v, err := workspaces.Scoped[T](wsCtx(t, w), db).Where(column, value).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func latestImport(t *testing.T, app *vukemtest.App, w who) models.Import {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var id int64
	if err := db.QueryRowContext(t.Context(), "SELECT MAX(id) FROM imports WHERE workspace_id = ?", w.Workspace.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}

	return firstOfWhere[models.Import](t, app, w, "id", id)
}

// csvMapping maps the columns of the statements these tests write.
var csvMapping = finance.Mapping{Date: "Date", Description: "Description", Amount: "Amount", DateFormat: "2006-01-02"}

// runImport uploads, confirms and runs one statement through the service, the way the handlers and the
// job do, without the HTTP rate limit of the import routes.
func runImport(t *testing.T, app *vukemtest.App, w who, acc int64, filename string, content []byte, m finance.Mapping) models.Import {
	t.Helper()

	fin := softService(t, app)
	ctx := wsCtx(t, w)

	if _, err := fin.StartImport(ctx, acc, filename, content); err != nil {
		t.Fatalf("start %s: %v", filename, err)
	}

	imp := latestImport(t, app, w)

	err := fin.ConfirmImport(ctx, imp.ID, m, func(ctx context.Context, _, id int64) error { return fin.RunImport(ctx, id) })
	if err != nil {
		t.Fatalf("confirm %s: %v", filename, err)
	}

	return latestImport(t, app, w)
}

func importFixture(t *testing.T, app *vukemtest.App, w who, acc int64, name string, allow bool) models.Import {
	t.Helper()

	return runImport(t, app, w, acc, name, fixtureBytes(t, name), finance.Mapping{AllowDuplicates: allow})
}

func csvWith(allow bool) finance.Mapping {
	m := csvMapping
	m.AllowDuplicates = allow

	return m
}

func proWorkspace(t *testing.T) (*vukemtest.App, who, int64) {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	return app, ana, firstAccount(t, app, ana)
}

func TestImportSkipsDuplicates(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	first := importFixture(t, app, ana, acc, "sgml.ofx", false)
	if first.Status != models.ImportDone || first.Imported != 3 || first.Duplicates != 0 || first.Format != "ofx" {
		t.Fatalf("first import = %+v, want 3 imported", first)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{
		"description": "Corner & Market", "kind": "expense", "amount_cents": 8240, "occurred_on": "2026-03-03",
	})

	// The same file again: zero new rows, all reported as duplicates.
	second := importFixture(t, app, ana, acc, "sgml.ofx", false)
	if second.Status != models.ImportDone || second.Imported != 0 || second.Duplicates != 3 {
		t.Fatalf("second import = %+v, want 0 imported and 3 duplicates", second)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)

	// The key is the FITID, so a re-issued file with other text is a duplicate too.
	reissued := strings.ReplaceAll(string(fixtureBytes(t, "sgml.ofx")), "Rent March", "RENT   MARCH (edited)")
	if third := runImport(t, app, ana, acc, "again.ofx", []byte(reissued), finance.Mapping{}); third.Imported != 0 || third.Duplicates != 3 {
		t.Errorf("a re-issued file = %+v, want 3 duplicates by FITID", third)
	}

	// A CSV has no FITID: the key is account, date, signed amount and normalized description.
	csv := []byte("Date,Description,Amount\n2026-04-01,Coffee Shop,-3.50\n2026-04-02,Rent,-900.00\n")
	for i, want := range []int{2, 0} {
		if got := runImport(t, app, ana, acc, "april.csv", csv, csvWith(false)); got.Imported != want || got.Duplicates != 2-want {
			t.Errorf("csv run %d = %+v, want %d imported", i, got, want)
		}
	}

	spaced := []byte("Date,Description,Amount\n2026-04-01,  coffee   SHOP ,-3.50\n")
	if got := runImport(t, app, ana, acc, "spaced.csv", spaced, csvWith(false)); got.Imported != 0 || got.Duplicates != 1 {
		t.Errorf("a respelled description = %+v, want a duplicate", got)
	}

	// Another account is another namespace: the same row there is new.
	other, err := softService(t, app).CreateAccount(wsCtx(t, ana), "Second", "savings", "0")
	if err != nil {
		t.Fatal(err)
	}

	if got := runImport(t, app, ana, other.ID, "april.csv", csv, csvWith(false)); got.Imported != 2 {
		t.Errorf("the same file into another account = %+v, want 2 imported", got)
	}
}

func TestImportDuplicatesAnyway(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	importFixture(t, app, ana, acc, "xml.ofx", false)

	again := importFixture(t, app, ana, acc, "xml.ofx", true)
	if again.Imported != 2 || again.Duplicates != 0 {
		t.Fatalf("with the switch on = %+v, want the 2 rows imported again", again)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 4)

	if third := importFixture(t, app, ana, acc, "xml.ofx", false); third.Imported != 0 || third.Duplicates != 2 {
		t.Errorf("third run = %+v, want 2 duplicates", third)
	}
}

func TestImportIdenticalRowsInOneFileAreLegitimate(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	csv := []byte("Date,Description,Amount\n2026-05-01,Bus ticket,-2.00\n2026-05-01,Bus ticket,-2.00\n2026-05-01,Bus ticket,-2.00\n")

	for i, want := range []int{3, 0} {
		if got := runImport(t, app, ana, acc, "bus.csv", csv, csvWith(false)); got.Imported != want || got.Duplicates != 3-want {
			t.Errorf("run %d = %+v, want %d imported", i, got, want)
		}
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 3)

	// A repeated FITID is one bank transaction, not three purchases.
	dup := strings.Replace(string(fixtureBytes(t, "xml.ofx")), "<FITID>X-2</FITID>", "<FITID>X-1</FITID>", 1)
	if got := runImport(t, app, ana, acc, "same-fitid.ofx", []byte(dup), finance.Mapping{}); got.Imported != 1 || got.Duplicates != 1 {
		t.Errorf("a repeated FITID = %+v, want 1 imported and 1 duplicate", got)
	}
}

func TestImportReport(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	var b strings.Builder

	b.WriteString("Date,Description,Amount\n2026-06-01,Good one,-1.00\n")

	for i := range 150 {
		fmt.Fprintf(&b, "not a date %d,Bad %d,-1.00\n", i, i)
	}

	b.WriteString("2026-06-02,Zero,0.00\n2026-06-03,Good two,-2.00\n")

	// The upload of a CSV answers with its columns; nothing is created yet.
	uploadFile(t, ana, acc, "report.csv", "text/csv", []byte(b.String())).AssertOK().AssertInertia("Import").
		HasProp("preview.headers", 3).PropEquals("preview.format", "csv")

	imp := latestImport(t, app, ana)
	mapping := url.Values{"date": {"Date"}, "description": {"Description"}, "amount": {"Amount"}, "dateFormat": {"2006-01-02"}}

	// The preview shows the first rows as they will be stored and the totals of the file.
	pv := ana.Client.Inertia("").PostForm("/import/"+idString(imp.ID)+"/preview", mapping).AssertInertia("Import")
	pv.HasProp("preview.mapped.rows", 20).HasProp("preview.mapped.failures", 100).
		PropEquals("preview.mapped.read", float64(153)).PropEquals("preview.mapped.new", float64(2)).
		PropEquals("preview.mapped.failed", float64(151)).PropEquals("preview.mapped.duplicates", float64(0)).
		PropEquals("preview.mapped.from", "2026-06-01").PropEquals("preview.mapped.to", "2026-06-03")
	vukemtest.AssertDatabaseCount(t, app, "transactions", 0)

	ana.Client.PostForm("/import/"+idString(imp.ID), mapping).AssertRedirect("/import")

	got := latestImport(t, app, ana)
	if got.Status != models.ImportDone || got.Imported != 2 || got.Failed != 151 || got.Duplicates != 0 {
		t.Fatalf("import = %+v, want 2 imported and 151 failed", got)
	}

	// The page lists 100 failures with their lines and reasons, in file order.
	ana.Client.Inertia("").Get("/import").AssertInertia("Import").
		HasProp("imports.0.failures", 100).
		PropEquals("imports.0.failures.0.line", float64(3)).PropEquals("imports.0.failures.0.reason", "bad date").
		PropEquals("imports.0.failed", float64(151)).PropEquals("imports.0.imported", float64(2))

	// The report holds lines and reasons, never the file's own content.
	if strings.Contains(got.Report, "Bad 1") {
		t.Error("the report carries cell content")
	}
}

func TestImportPreviewOfAnOFXFileMarksDuplicates(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	importFixture(t, app, ana, acc, "sgml.ofx", false)

	uploadFile(t, ana, acc, "sgml.ofx", "application/x-ofx", fixtureBytes(t, "sgml.ofx")).AssertOK().AssertInertia("Import").
		PropEquals("preview.format", "ofx").HasProp("preview.mapped.rows", 3).
		PropEquals("preview.mapped.duplicates", float64(3)).PropEquals("preview.mapped.new", float64(0)).
		PropEquals("preview.mapped.rows.0.duplicate", true).PropEquals("preview.mapped.rows.0.cents", float64(250000))
}

func TestImportFailureLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	if _, err := fin.StartImport(ctx, acc, "march.csv", []byte(statement)); err != nil {
		t.Fatal(err)
	}

	imp := latestImport(t, app, ana)

	// The job is queued, then the account is archived while it waits: the run fails whole.
	if err := fin.ConfirmImport(ctx, imp.ID, csvMapping, func(context.Context, int64, int64) error { return nil }); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.ArchiveAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}

	if err := fin.RunImport(ctx, imp.ID); err != nil {
		t.Fatal(err)
	}

	got := latestImport(t, app, ana)
	if got.Status != models.ImportFailed || got.Imported != 0 || got.Error == "" || got.Content != "" {
		t.Errorf("import = %+v, want failed, empty and with the content dropped", got)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 0)

	// A second run of the same job does nothing.
	if err := fin.RunImport(ctx, imp.ID); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseCount(t, app, "transactions", 0)
}

func TestImportNeverMakesTransferLegs(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	fin := softService(t, app)

	other, err := fin.CreateAccount(wsCtx(t, ana), "Savings", "savings", "0")
	if err != nil {
		t.Fatal(err)
	}

	// A transfer of 82.40 on 2026-03-03 sits in both accounts; the statement has the same row.
	if _, err := fin.CreateTransfer(wsCtx(t, ana), finance.TransferInput{
		FromAccountID: acc, ToAccountID: other.ID, Amount: "82.40", Date: "2026-03-03",
	}); err != nil {
		t.Fatal(err)
	}

	legs := count[models.Transaction](t, app, ana)

	imp := importFixture(t, app, ana, acc, "sgml.ofx", false)
	if imp.Imported != 3 || imp.Duplicates != 0 {
		t.Fatalf("import = %+v, want 3 imported next to the transfer", imp)
	}

	if got := count[models.Transaction](t, app, ana); got != legs+3 {
		t.Errorf("%d transactions, want the %d legs and 3 new ones", got, legs)
	}

	// Every imported row is an ordinary entry; the two legs are still the only paired rows, with no import key.
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var paired, pairedImported, importedPlain int
	if err := db.QueryRowContext(t.Context(),
		"SELECT COUNT(*), COALESCE(SUM(import_hash IS NOT NULL OR import_id IS NOT NULL), 0) FROM transactions WHERE transfer_uid IS NOT NULL").
		Scan(&paired, &pairedImported); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM transactions WHERE import_id IS NOT NULL AND transfer_uid IS NULL AND import_hash IS NOT NULL").
		Scan(&importedPlain); err != nil {
		t.Fatal(err)
	}

	if paired != 2 || pairedImported != 0 || importedPlain != 3 {
		t.Errorf("paired legs %d (imported %d), plain imported rows %d; want 2, 0 and 3", paired, pairedImported, importedPlain)
	}
}
