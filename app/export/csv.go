// Package export writes the data of a workspace in formats other programs
// read: a CSV of transactions that opens in a spreadsheet and imports back,
// and a JSON document of the whole workspace. Text cells are neutralized
// against spreadsheet formulas, nothing is buffered whole, and no spreadsheet
// library is involved.
package export

import (
	"bufio"
	"encoding/csv"
	"io"
	"iter"
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// MaxCSVRows bounds one CSV export. A larger result is refused with a 413
// rather than cut: a silent truncation would look like a complete backup.
const MaxCSVRows = 200000

// ErrTooManyRows is the answer to an export of more than MaxCSVRows rows.
var ErrTooManyRows = errs.New(http.StatusRequestEntityTooLarge,
	"The export has more than 200000 transactions: narrow the period or the filters and try again.")

// CSVHeader is the first row of the transaction CSV.
var CSVHeader = []string{"uid", "date", "account", "category", "tags", "kind", "amount", "description", "transfer_uid"}

// CSVOptions are the spreadsheet quirks the person can ask for: BOM for old
// versions of Excel that need it to read UTF-8, Semicolon for locales whose
// decimal separator is a comma and whose list separator is a semicolon.
type CSVOptions struct {
	BOM       bool
	Semicolon bool
}

// TransactionRow is one transaction as the CSV shows it. AmountCents is
// signed: negative for an expense.
type TransactionRow struct {
	UID         string
	Date        string
	Account     string
	Category    string
	Tags        []string
	Kind        string
	AmountCents int64
	Description string
	TransferUID string
}

const utf8BOM = "\xEF\xBB\xBF"

// flushEvery is how many rows are written before the buffer is flushed to
// the response, so a long export streams.
const flushEvery = 500

// WriteCSV writes the header and then the rows to w, UTF-8 with \r\n line
// ends and RFC 4180 quoting, and returns the number of rows written. Every
// text cell goes through finance.SpreadsheetSafe (a cell starting with =, +,
// -, @, a tab or a carriage return gets a leading quote, so a spreadsheet does
// not evaluate it); the amount is the one cell not guarded, because it is
// built from an integer and its minus sign is data. A row past MaxCSVRows
// stops the export with ErrTooManyRows.
func WriteCSV(w io.Writer, rows iter.Seq2[TransactionRow, error], opt CSVOptions) (int, error) {
	buf := bufio.NewWriter(w)

	if opt.BOM {
		if _, err := buf.WriteString(utf8BOM); err != nil {
			return 0, err
		}
	}

	cw := csv.NewWriter(buf)
	cw.UseCRLF = true

	if opt.Semicolon {
		cw.Comma = ';'
	}

	if err := cw.Write(CSVHeader); err != nil {
		return 0, err
	}

	// The header leaves at once: the person sees the download start.
	cw.Flush()

	if err := buf.Flush(); err != nil {
		return 0, err
	}

	n := 0

	for r, err := range rows {
		if err != nil {
			return n, err
		}

		if n >= MaxCSVRows {
			return n, ErrTooManyRows
		}

		if err := cw.Write(csvRecord(r)); err != nil {
			return n, err
		}

		n++

		if n%flushEvery == 0 {
			cw.Flush()

			if err := buf.Flush(); err != nil {
				return n, err
			}
		}
	}

	cw.Flush()

	if err := cw.Error(); err != nil {
		return n, err
	}

	return n, buf.Flush()
}

func csvRecord(r TransactionRow) []string {
	return []string{
		textCell(r.UID),
		textCell(r.Date),
		textCell(r.Account),
		textCell(r.Category),
		textCell(strings.Join(r.Tags, ";")),
		textCell(r.Kind),
		finance.FormatDecimalCents(r.AmountCents),
		textCell(r.Description),
		textCell(r.TransferUID),
	}
}

// textCell guards every piece of s between commas and semicolons. Excel in a
// locale whose list separator is the other one (pt-BR reads ";") splits the
// row there without honoring the quoting, so each piece can become a cell.
func textCell(s string) string {
	var b strings.Builder

	for {
		i := strings.IndexAny(s, ",;")
		if i < 0 {
			b.WriteString(finance.SpreadsheetSafe(s))

			return b.String()
		}

		b.WriteString(finance.SpreadsheetSafe(s[:i]))
		b.WriteByte(s[i])
		s = s[i+1:]
	}
}
