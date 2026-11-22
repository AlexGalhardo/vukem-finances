package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

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
		return writeCSV(c.Env.Stdout, res.Value, c.Env.StdoutTTY)
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

// maxColumns bounds how wide a table or CSV may get: a server decides the keys,
// and every distinct key is a column of every row.
const maxColumns = 256

func gridOf(v any) (grid, error) {
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
				if len(g.cols) >= maxColumns {
					return grid{}, &Failure{Code: ExitFailure, Msg: "the answer has too many columns for table/csv; use --output json"}
				}

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

	return g, nil
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
	g, err := gridOf(v)
	if err != nil {
		return err
	}

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

// writeCSV writes the answer as CSV. On a terminal (tty) the text also goes
// through cleanText; redirected output is byte-exact apart from the formula
// guard.
func writeCSV(w io.Writer, v any, tty bool) error {
	g, err := gridOf(v)
	if err != nil {
		return err
	}

	text := func(s string) string {
		if tty {
			return cleanText(s)
		}

		return s
	}

	cw := csv.NewWriter(w)

	head := make([]string, len(g.cols))
	for i, c := range g.cols {
		head[i] = csvSafe(text(c), false)
	}

	if err := cw.Write(head); err != nil {
		return err
	}

	for _, row := range g.rows {
		cells := make([]string, len(g.cols))

		for i, c := range g.cols {
			_, numeric := row[c].(json.Number)
			cells[i] = csvSafe(text(cellText(row[c])), numeric)
		}

		if err := cw.Write(cells); err != nil {
			return err
		}
	}

	cw.Flush()

	return cw.Error()
}

// csvSafe neutralizes formula injection: a spreadsheet runs a cell whose first
// meaningful character is = + - @ (or its full-width form), and it skips the
// whitespace, control and format characters in front of it, so those are
// skipped here too before the first rune is judged. Such a cell gets a leading
// quote. A real number is not text and stays as it is, so -10.50 is still a
// number in the sheet; a text value is exempt only when the whole of it is a
// plain negative decimal ("-10.50"), which no spreadsheet reads as a formula.
func csvSafe(s string, numeric bool) string {
	if s == "" || numeric || plainNegative(s) {
		return s
	}

	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}

		switch r {
		case '=', '+', '-', '@', '＝', '＋', '－', '＠':
			return "'" + s
		}

		break
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
		if c.Env.StdoutTTY {
			return usagef("this answer is a file: pass --out FILE, or pipe it (| or >)")
		}

		return res.Stream(c.Env.Stdout)
	}

	return writeAtomic(outFile, res.Stream)
}

// writeAtomic writes through a private temp file in the folder of the target
// (CreateTemp makes it 0600) and renames it over the target: a failed download never truncates an
// existing file, and a symlink or a looser mode at the target is replaced, not
// reused.
func writeAtomic(target string, stream func(io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(target), ".download-*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}

	defer func() {
		if err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
	}()

	if err = stream(f); err != nil {
		return err
	}

	if err = f.Sync(); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}

	if err = os.Rename(f.Name(), target); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}

	return nil
}
