package export_test

import (
	"bytes"
	"encoding/csv"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/export"
)

func seq(rows ...export.TransactionRow) iter.Seq2[export.TransactionRow, error] {
	return func(yield func(export.TransactionRow, error) bool) {
		for _, r := range rows {
			if !yield(r, nil) {
				return
			}
		}
	}
}

// Excel in a locale whose list separator is ";" splits a comma CSV at every
// semicolon (and the reverse), so the text after a separator can become a cell.
func TestCSVFormulaBehindSeparator(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	_, err := export.WriteCSV(&out, seq(export.TransactionRow{
		UID: "01J", Date: "2026-01-02", Account: "a, +1", Tags: []string{"ok", "=cmd"}, Kind: "expense",
		AmountCents: -1, Description: "paid;=HYPERLINK(1)\n=1+1\ny\xe2\x80\xa8@2\t-3",
	}), export.CSVOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		// encoding/csv writes a line end inside a quoted cell as \r\n under UseCRLF.
		"paid;'=HYPERLINK(1)", "\r\n'=1+1\r\ny", "\xe2\x80\xa8'@2", "\t'-3", "ok;'=cmd", "a,' +1",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("%q not in %q", want, out.String())
		}
	}
}

func TestCSVFormulaGuard(t *testing.T) {
	t.Parallel()

	nasty := []string{
		`=HYPERLINK("http://example.test")`, "+1", "-1", "@SUM(A1)", "\t=1+1", "\r=1", "line one\nline \"two\", with comma",
	}

	var rows []export.TransactionRow

	for _, text := range nasty {
		rows = append(rows, export.TransactionRow{
			UID: "01J", Date: "2026-01-02", Account: text, Category: text, Tags: []string{text}, Kind: "expense",
			AmountCents: -1234, Description: text, TransferUID: text,
		})
	}

	var out bytes.Buffer

	n, err := export.WriteCSV(&out, seq(rows...), export.CSVOptions{})
	if err != nil || n != len(nasty) {
		t.Fatalf("WriteCSV = %d, %v", n, err)
	}

	if !strings.Contains(out.String(), "'=HYPERLINK(") {
		t.Errorf("the formula is not neutralized in %q", out.String())
	}

	if !strings.HasPrefix(out.String(), "uid,date,account,category,tags,kind,amount,description,transfer_uid\r\n") {
		t.Errorf("header or line ending wrong: %q", out.String())
	}

	records, err := csv.NewReader(&out).ReadAll()
	if err != nil {
		t.Fatalf("the output is not valid CSV: %v", err)
	}

	for i, rec := range records[1:] {
		want := nasty[i]
		if strings.ContainsRune("=+-@\t\r", rune(want[0])) {
			want = "'" + want
		}

		// encoding/csv drops carriage returns inside a quoted cell on reading: compare without them.
		for col, cell := range rec {
			switch col {
			case 6:
				if cell != "-12.34" {
					t.Errorf("amount = %q, want the plain signed decimal", cell)
				}
			case 0, 1, 5:
			default:
				if strings.ReplaceAll(cell, "\r", "") != strings.ReplaceAll(want, "\r", "") {
					t.Errorf("row %d column %d = %q, want %q", i, col, cell, want)
				}
			}

			if col != 6 && col > 1 && col != 5 && strings.ContainsRune("=+-@\t\r", rune(cell[0])) {
				t.Errorf("row %d column %d starts a formula: %q", i, col, cell)
			}
		}
	}
}

func TestCSVOptions(t *testing.T) {
	t.Parallel()

	row := export.TransactionRow{UID: "01J", Date: "2026-01-02", Account: "Main", Description: "a;b", Kind: "income", AmountCents: 5}

	var out bytes.Buffer

	if _, err := export.WriteCSV(&out, seq(row), export.CSVOptions{BOM: true, Semicolon: true}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "\xEF\xBB\xBFuid;date;account;") {
		t.Errorf("want a BOM and semicolons: %q", got)
	}

	if !strings.Contains(got, `;income;0.05;"a;b";`) {
		t.Errorf("a cell holding the delimiter is quoted and the amount is machine decimal: %q", got)
	}
}

// pulled counts how many rows the exporter asked for and how many bytes
// had reached the writer at that moment.
type probe struct {
	bytes int
	log   []int
}

func (p *probe) Write(b []byte) (int, error) { p.bytes += len(b); return len(b), nil }

func TestCSVStreams(t *testing.T) {
	t.Parallel()

	p := &probe{}

	const total = 1500

	rows := func(yield func(export.TransactionRow, error) bool) {
		for i := range total {
			p.log = append(p.log, p.bytes)

			if !yield(export.TransactionRow{UID: "u", Date: "2026-01-01", Kind: "expense", AmountCents: -int64(i)}, nil) {
				return
			}
		}
	}

	if _, err := export.WriteCSV(p, rows, export.CSVOptions{}); err != nil {
		t.Fatal(err)
	}

	if p.log[0] == 0 {
		t.Error("the header was not written before the first row was read")
	}

	if p.log[total-1] == p.log[0] {
		t.Error("nothing was written while the rows were still being read: the export is buffered whole")
	}
}

func TestCSVCap(t *testing.T) {
	t.Parallel()

	rows := func(yield func(export.TransactionRow, error) bool) {
		for range export.MaxCSVRows + 1 {
			if !yield(export.TransactionRow{UID: "u", Date: "2026-01-01", Kind: "expense", AmountCents: -1}, nil) {
				return
			}
		}
	}

	var out bytes.Buffer

	n, err := export.WriteCSV(&out, rows, export.CSVOptions{})
	if !errors.Is(err, export.ErrTooManyRows) || n != export.MaxCSVRows {
		t.Errorf("WriteCSV = %d, %v, want %d rows and ErrTooManyRows", n, err, export.MaxCSVRows)
	}
}
