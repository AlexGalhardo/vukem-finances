package finance

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"time"
)

// DateFormats are the formats a statement's dates may use (Go layouts).
var DateFormats = []string{"2006-01-02", "02/01/2006", "01/02/2006", "02.01.2006"}

// Mapping says which CSV column holds what. Columns are header names. A
// negative amount is an expense, a positive one income, unless Invert.
type Mapping struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Category    string `json:"category"`
	DateFormat  string `json:"dateFormat"`
	Invert      bool   `json:"invert"`
	// AllowDuplicates imports the rows that an earlier import already brought
	// in; it is a setting of the import, not a column.
	AllowDuplicates bool `json:"allowDuplicates"`
}

const (
	// maxCSVColumns bounds the columns of a record.
	maxCSVColumns = 200
	// maxCSVCell bounds a cell the importer reads, in bytes.
	maxCSVCell = 1000
)

// CSVParser reads a CSV statement through the column Mapping the user chose.
type CSVParser struct {
	Mapping Mapping
}

func readCSV(content string) ([]string, [][]string, error) {
	// Statements from European banks use ";": pick it when the header has more of it.
	head, _, _ := strings.Cut(content, "\n")
	r := csv.NewReader(strings.NewReader(content))
	r.Comma = ','

	if strings.Count(head, ";") > strings.Count(head, ",") {
		r.Comma = ';'
	}

	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var rows [][]string

	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, errImportCSV
		}

		// The header is one record: more than maxImportRows after it is a file to split, not to cut.
		if len(rows) >= maxImportRows+1 {
			return nil, nil, errImportRows
		}

		if len(rec) > maxCSVColumns {
			return nil, nil, errImportCSV
		}

		rows = append(rows, rec)
	}

	if len(rows) < 2 {
		return nil, nil, errImportCSV
	}

	headers := rows[0]
	for i := range headers {
		headers[i] = strings.TrimSpace(strings.TrimPrefix(headers[i], string(rune(0xFEFF))))
	}

	return headers, rows[1:], nil
}

// Parse implements StatementParser.
func (p CSVParser) Parse(content []byte) ([]StatementRow, error) {
	m := p.Mapping

	// A bank file is often Windows-1252 and may start with a BOM: read it as text, never keep invalid UTF-8.
	headers, rows, err := readCSV(decodeOFXText(content))
	if err != nil {
		return nil, err
	}

	di, ni, ai, ci := indexOf(headers, m.Date), indexOf(headers, m.Description), indexOf(headers, m.Amount),
		indexOf(headers, m.Category)
	if di < 0 || ni < 0 || ai < 0 {
		return nil, errMapping
	}

	out := make([]StatementRow, len(rows))
	for i, row := range rows {
		out[i] = csvRow(row, i+2, di, ni, ai, ci, m)
	}

	return out, nil
}

func csvRow(row []string, line, di, ni, ai, ci int, m Mapping) StatementRow {
	r := StatementRow{Line: line}

	if len(row) <= max(di, ni, ai) {
		r.Problem = "too few columns"
		return r
	}

	for _, i := range []int{di, ni, ai, ci} {
		if i >= 0 && i < len(row) && len(row[i]) > maxCSVCell {
			r.Problem = "field too long"
			return r
		}
	}

	date, err := time.Parse(m.DateFormat, strings.TrimSpace(row[di]))
	if err != nil {
		r.Problem = "bad date"
		return r
	}

	cents, err := ParseAmount(row[ai])
	if err != nil || cents == 0 {
		r.Problem = "bad amount"
		return r
	}

	if m.Invert {
		cents = -cents
	}

	r.Date, r.Description, r.Cents = date.Format(isoDate), strings.TrimSpace(row[ni]), cents

	if ci >= 0 && ci < len(row) {
		r.Category = strings.TrimSpace(row[ci])
	}

	return r
}
