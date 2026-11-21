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
}

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

		if len(rows) >= maxImportRows+1 {
			break
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

	headers, rows, err := readCSV(string(content))
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
