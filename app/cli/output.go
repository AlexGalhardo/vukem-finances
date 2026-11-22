package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

func (c *CLI) write(_ Command, res Response, format, outFile string) error {
	if res.Stream != nil {
		return c.writeFile(res, outFile)
	}

	switch format {
	case "table":
		return writeTable(c.Env.Stdout, res.Value)
	case "csv":
		return writeCSV(c.Env.Stdout, res.Value)
	}

	return writeJSON(c.Env.Stdout, res.Value)
}

// writeJSON keeps every number as the text the operation produced, so an
// amount is the exact decimal of the answer next to its integer cents.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(v)
}

// grid is the answer as rows: a list answer ({"data": [...]} or an array) has
// one row per item, anything else one row.
type grid struct {
	cols []string
	rows []map[string]any
}

func gridOf(v any) grid {
	var items []any

	switch x := v.(type) {
	case []any:
		items = x
	case map[string]any:
		if data, ok := x["data"].([]any); ok {
			items = data
		} else {
			items = []any{x}
		}
	default:
		items = []any{map[string]any{"value": v}}
	}

	g := grid{}
	seen := map[string]bool{}

	for _, it := range items {
		row, ok := it.(map[string]any)
		if !ok {
			row = map[string]any{"value": it}
		}

		g.rows = append(g.rows, row)

		for k := range row {
			if !seen[k] {
				seen[k] = true
				g.cols = append(g.cols, k)
			}
		}
	}

	sort.Slice(g.cols, func(i, j int) bool {
		a, b := g.cols[i], g.cols[j]
		if (a == "uid") != (b == "uid") {
			return a == "uid"
		}

		return a < b
	})

	return g
}

const centsSuffix = "_cents"

// cellText is a value as the person reads it: nested values as compact JSON.
func cellText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return string(raw)
}

// tableCell shows whole cents as a decimal amount.
func tableCell(col string, v any) string {
	if n, ok := v.(json.Number); ok && strings.HasSuffix(col, centsSuffix) {
		if cents, err := strconv.ParseInt(n.String(), 10, 64); err == nil {
			return finance.FormatDecimalCents(cents)
		}
	}

	return cellText(v)
}

func writeTable(w io.Writer, v any) error {
	g := gridOf(v)
	if len(g.rows) == 0 {
		_, err := fmt.Fprintln(w, "No results.")

		return err
	}

	has := map[string]bool{}
	for _, c := range g.cols {
		has[c] = true
	}

	var cols, heads []string

	for _, c := range g.cols {
		head := c

		if strings.HasSuffix(c, centsSuffix) {
			// "amount" next to "amount_cents" already shows the decimal.
			base := strings.TrimSuffix(c, centsSuffix)
			if has[base] {
				continue
			}

			head = base
		}

		cols, heads = append(cols, c), append(heads, strings.ToUpper(cleanText(head)))
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprintln(tw, strings.Join(heads, "\t"))

	for _, row := range g.rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = cleanText(tableCell(c, row[c]))
		}

		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}

	return tw.Flush()
}

func writeCSV(w io.Writer, v any) error {
	g := gridOf(v)
	cw := csv.NewWriter(w)

	head := make([]string, len(g.cols))
	for i, c := range g.cols {
		head[i] = csvSafe(c, false)
	}

	if err := cw.Write(head); err != nil {
		return err
	}

	for _, row := range g.rows {
		cells := make([]string, len(g.cols))

		for i, c := range g.cols {
			_, numeric := row[c].(json.Number)
			cells[i] = csvSafe(cellText(row[c]), numeric)
		}

		if err := cw.Write(cells); err != nil {
			return err
		}
	}

	cw.Flush()

	return cw.Error()
}

// csvSafe neutralizes formula injection: a spreadsheet runs a cell that starts
// with = + - @ (or a tab or CR that hides them), so such a cell gets a leading
// quote. A real number is not text and stays as it is, so -10.50 is still a
// number in the sheet; a text value is exempt only when the whole of it is a
// plain negative decimal ("-10.50"), which no spreadsheet reads as a formula.
func csvSafe(s string, numeric bool) string {
	if s == "" || numeric || plainNegative(s) {
		return s
	}

	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}

	return s
}

// plainNegative matches -123 and -123.45 and nothing else.
func plainNegative(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}

	digits, dot := 0, false

	for _, r := range s[1:] {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '.' && !dot && digits > 0:
			dot = true
			digits = 0

			continue
		default:
			return false
		}
	}

	return digits > 0
}

func (c *CLI) writeFile(res Response, outFile string) error {
	if outFile == "" {
		return res.Stream(c.Env.Stdout)
	}

	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}

	if err := res.Stream(f); err != nil {
		_ = f.Close()

		return err
	}

	return f.Close()
}
