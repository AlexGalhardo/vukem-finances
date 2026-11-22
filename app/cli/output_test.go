package cli_test

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

func TestOutput(t *testing.T) {
	t.Parallel()

	app := newEdition(t, "personal")
	signUp(t, app, "Ana", "ana@example.com")

	c, tm := newCLI(t, nil, localOf(app))

	create := func(name string, extra ...string) {
		t.Helper()

		args := append([]string{"accounts", "create", "--name", name, "--kind", "checking"}, extra...)
		if code := run(t, c, tm, args...); code != 0 {
			t.Fatalf("create exit %d: %s", code, tm.err.String())
		}
	}

	create("Checking", "--opening", "1.234,56")
	create("=cmd|' /C calc'!A0", "--opening", "-10.50")

	t.Run("json keeps the exact decimal and the integer cents", func(t *testing.T) {
		if code := run(t, c, tm, "accounts", "list"); code != 0 {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}

		out := tm.out.String()
		if !strings.Contains(out, `"opening_cents": 123456`) || !strings.Contains(out, `"opening_cents": -1050`) {
			t.Fatalf("cents not kept as integers:\n%s", out)
		}
	})

	t.Run("table shows 1.234,56 as 1234.56 and negative amounts as they are", func(t *testing.T) {
		c.Env.StdoutTTY = true

		t.Cleanup(func() { c.Env.StdoutTTY = false })

		if code := run(t, c, tm, "accounts", "list"); code != 0 {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}

		out := tm.out.String()
		if !strings.Contains(out, "OPENING") || !strings.Contains(out, "1234.56") || !strings.Contains(out, "-10.50") {
			t.Fatalf("table:\n%s", out)
		}

		if strings.Contains(out, "{") {
			t.Fatalf("table is JSON:\n%s", out)
		}
	})

	t.Run("csv neutralizes formulas and leaves numbers alone", func(t *testing.T) {
		if code := run(t, c, tm, "accounts", "list", "--output", "csv"); code != 0 {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}

		rows, err := csv.NewReader(strings.NewReader(tm.out.String())).ReadAll()
		if err != nil || len(rows) != 4 {
			t.Fatalf("csv: %v, %d rows:\n%s", err, len(rows), tm.out.String())
		}

		col := map[string]int{}
		for i, h := range rows[0] {
			col[h] = i
		}

		var sawFormula, sawNegative bool

		for _, r := range rows[1:] {
			switch r[col["name"]] {
			case "'=cmd|' /C calc'!A0":
				sawFormula = true
			case "Checking", "Main account":
			default:
				t.Fatalf("name %q is not neutralized", r[col["name"]])
			}

			if r[col["opening_cents"]] == "-1050" {
				sawNegative = true
			}
		}

		if !sawFormula || !sawNegative {
			t.Fatalf("formula neutralized %v, negative kept %v:\n%s", sawFormula, sawNegative, tm.out.String())
		}
	})

	t.Run("a bad format is a usage error", func(t *testing.T) {
		if code := run(t, c, tm, "accounts", "list", "--output", "xml"); code != cli.ExitUsage {
			t.Fatalf("exit %d", code)
		}
	})
}

func TestOutputCells(t *testing.T) {
	t.Parallel()

	tm := &term{}
	c := &cli.CLI{Env: cli.Env{Stdout: &tm.out}}

	val := func(raw string) any {
		var v any

		d := json.NewDecoder(strings.NewReader(raw))
		d.UseNumber()

		if err := d.Decode(&v); err != nil {
			t.Fatal(err)
		}

		return v
	}

	t.Run("csv", func(t *testing.T) {
		tm.out.Reset()

		v := val(`{"data":[{"a":"=1+1","b":"+1","c":"@x","d":"-1+2","e":"-10.50","f":-10.5,"g":"\tx","h":"plain","i":"-1.","j":"-abc"}]}`)
		if err := c.WriteForTest(v, "csv"); err != nil {
			t.Fatal(err)
		}

		want := "a,b,c,d,e,f,g,h,i,j\n'=1+1,'+1,'@x,'-1+2,-10.50,-10.5,'\tx,plain,'-1.,'-abc\n"
		if tm.out.String() != want {
			t.Fatalf("got:\n%q\nwant:\n%q", tm.out.String(), want)
		}
	})

	t.Run("table strips terminal control and format characters", func(t *testing.T) {
		tm.out.Reset()

		v := val(`[{"name":"a\u001b[2Jb\u202Ec\r\nd","amount_cents":-5}]`)
		if err := c.WriteForTest(v, "table"); err != nil {
			t.Fatal(err)
		}

		out := tm.out.String()
		for _, bad := range []string{"\x1b", "\u202e", "\r"} {
			if strings.Contains(out, bad) {
				t.Fatalf("table has %q:\n%q", bad, out)
			}
		}

		if strings.Count(out, "\n") != 2 || !strings.Contains(out, "-0.05") {
			t.Fatalf("table = %q", out)
		}
	})
}

func TestOutputFiles(t *testing.T) {
	t.Parallel()

	app := newEdition(t, "personal")
	signUp(t, app, "Ana", "ana@example.com")

	c, tm := newCLI(t, nil, localOf(app))

	t.Run("a download is refused on a terminal without --out", func(t *testing.T) {
		c.Env.StdoutTTY = true

		t.Cleanup(func() { c.Env.StdoutTTY = false })

		if code := run(t, c, tm, "export", "transactions"); code != cli.ExitUsage || !strings.Contains(tm.err.String(), "--out") {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}

		if tm.out.Len() != 0 {
			t.Fatalf("bytes reached the terminal: %q", tm.out.String())
		}
	})

	t.Run("piped output is the file itself", func(t *testing.T) {
		if code := run(t, c, tm, "export", "transactions"); code != 0 || !strings.Contains(tm.out.String(), ",") {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}
	})

	t.Run("--out replaces the target through a private temp file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "out.csv")

		if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		if code := run(t, c, tm, "export", "transactions", "--out", target); code != 0 {
			t.Fatalf("exit %d: %s", code, tm.err.String())
		}

		got, err := os.ReadFile(target)
		if err != nil || string(got) == "old" || len(got) == 0 {
			t.Fatalf("target = %q, %v", got, err)
		}

		if left, _ := filepath.Glob(filepath.Join(dir, ".download-*")); len(left) != 0 {
			t.Fatalf("temp files left: %v", left)
		}

		if runtime.GOOS != "windows" {
			if st, err := os.Stat(target); err != nil || st.Mode().Perm() != 0o600 {
				t.Fatalf("mode %v, %v", st.Mode(), err)
			}
		}
	})

	t.Run("a failed download keeps the old file and leaves no temp file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "keep.csv")

		if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}

		bad := &cli.CLI{Env: cli.Env{Stdout: &tm.out}}

		if err := bad.WriteFileForTest(target, func(w io.Writer) error {
			_, _ = w.Write([]byte("partial"))

			return errors.New("connection cut")
		}); err == nil {
			t.Fatal("no error")
		}

		if got, _ := os.ReadFile(target); string(got) != "old" {
			t.Fatalf("target = %q, want the old file", got)
		}

		if left, _ := filepath.Glob(filepath.Join(dir, ".download-*")); len(left) != 0 {
			t.Fatalf("temp files left: %v", left)
		}
	})
}
