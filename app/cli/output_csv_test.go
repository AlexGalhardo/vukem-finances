package cli_test

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

// startsFormula is what a spreadsheet reads as a formula: the first meaningful
// character, skipping whitespace, control, format characters and a quote.
func startsFormula(cell string) bool {
	for _, r := range cell {
		if r == '"' || unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}

		return strings.ContainsRune("=+-@＝＋－＠", r)
	}

	return false
}

func TestOutputCSVDelimiterGuard(t *testing.T) {
	t.Parallel()

	payloads := map[string]string{
		"x;=1+1":     "x;'=1+1",
		"a\n=1+1;":   "a\n'=1+1;",
		"a;\"=1+1;":  "a;'\"=1+1;",
		"a\nb,=1+1,": "a\nb,'=1+1,",
		"a\t+b":      "a\t'+b",
		"x;  @y":     "x;'  @y",
		"a;＝1":       "a;'＝1",
		"a;​-1":      "a;'​-1",
		"=1":         "'=1",
		"a;b":        "a;b",
		"a\tb":       "a\tb",
		"-10.50":     "-10.50",
		"a-b;c+d":    "a-b;c+d",
		"a,b":        "a,b",
		"a,-b":       "a,-b",
		"say \"hi\"": "say \"hi\"",
	}

	for cell, want := range payloads {
		tm := &term{}
		c := &cli.CLI{Env: cli.Env{Stdout: &tm.out}}

		if err := c.WriteForTest([]any{map[string]any{"k": cell, "n": json.Number("-10.5")}}, "csv"); err != nil {
			t.Fatal(err)
		}

		for _, comma := range []rune{',', ';', '\t'} {
			r := csv.NewReader(strings.NewReader(tm.out.String()))
			r.Comma = comma
			r.FieldsPerRecord = -1
			r.LazyQuotes = comma != ','

			rows, err := r.ReadAll()
			if err != nil {
				t.Fatalf("%q with %q: %v: %q", cell, comma, err, tm.out.String())
			}

			for _, row := range rows {
				for _, f := range row {
					if comma != ',' && startsFormula(f) && !strings.HasPrefix(f, "'") && cell != "-10.50" {
						t.Errorf("%q with %q: the cell %q starts with a trigger", cell, comma, f)
					}
				}
			}

			if comma == ',' && (len(rows) != 2 || len(rows[1]) != 2 || rows[1][0] != want || rows[1][1] != "-10.5") {
				t.Errorf("%q: rows %q, want the cell %q and a bare -10.5", cell, rows, want)
			}
		}
	}
}

func TestOutputCSVNumbersAreBare(t *testing.T) {
	t.Parallel()

	for number, bare := range map[string]bool{
		"-1050": true, "12.50": true, "1e3": true, "1;=2": false, "1\t2": false, "=1": false, "1,5": false, "-": false,
	} {
		tm := &term{}
		c := &cli.CLI{Env: cli.Env{Stdout: &tm.out}}

		if err := c.WriteForTest([]any{map[string]any{"n": json.Number(number)}}, "csv"); err != nil {
			t.Fatal(err)
		}

		out := tm.out.String()
		if got := !strings.Contains(strings.SplitN(out, "\n", 2)[1], "\""); got != bare {
			t.Errorf("%q: bare = %v, want %v: %q", number, got, bare, out)
		}
	}
}
