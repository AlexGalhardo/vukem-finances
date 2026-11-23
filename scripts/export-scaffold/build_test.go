package main

import (
	"bytes"
	"flag"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

// TestExportedTreeBuilds renders the exported tree the way the Vukem CLI
// does, under another module path and name, and builds, vets and tests it.
func TestExportedTreeBuilds(t *testing.T) {
	// A nested go test beside every other package of the repository failed under
	// load on Windows, so CI and the QA round run this one on its own, by name.
	if run := flag.Lookup("test.run"); run == nil || !strings.Contains(run.Value.String(), "TestExportedTreeBuilds") {
		t.Skip("builds a whole project: run it with -run TestExportedTreeBuilds")
	}

	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	tracked, err := readTree(repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	files, err := Build(tracked, "v0.0.0-test")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()

	// SCAFFOLD_RENDER_DIR keeps the rendered project for inspection.
	if keep := os.Getenv("SCAFFOLD_RENDER_DIR"); keep != "" {
		dir = keep
	}
	render := data{Module: "example.test/money", Name: "money"}

	for _, f := range files {
		body := f.Data

		if stub, ok := strings.CutSuffix(f.Path, ".tmpl"); ok {
			f.Path = stub

			tpl, err := template.New(f.Path).Option("missingkey=error").Parse(string(body))
			if err != nil {
				t.Fatalf("%s: %v", f.Path, err)
			}

			var b bytes.Buffer
			if err := tpl.Execute(&b, render); err != nil {
				t.Fatalf("%s: %v", f.Path, err)
			}

			body = b.Bytes()

			if strings.HasSuffix(f.Path, ".go") {
				if body, err = format.Source(body); err != nil {
					t.Fatalf("%s: %v", f.Path, err)
				}
			}
		}

		dst := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(dst, body, f.Mode); err != nil {
			t.Fatal(err)
		}
	}

	appMod, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	goVersion := firstMatch(t, appMod, `(?m)^go (\S+)$`)
	require := "require (\n"

	for _, mod := range []string{"github.com/AlexGalhardo/vukem", "github.com/AlexGalhardo/vukem/contrib/stripe"} {
		require += "\t" + mod + " " + firstMatch(t, appMod, `(?m)^\s*`+regexp.QuoteMeta(mod)+` (v\S+)`) + "\n"
	}

	gomod := "module " + render.Module + "\n\ngo " + goVersion + "\n\n" + require + ")\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}

	// The checksums only seed the module cache lookups; tidy rewrites them.
	if sum, err := os.ReadFile(filepath.Join(repo, "go.sum")); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// -p 4: the full parallelism of a Windows machine trips over SQLite temp
	// directories that are still being released when the test cleans up.
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-p", "4", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, tail(out))
		}
	}
}

func firstMatch(t *testing.T, src []byte, expr string) string {
	t.Helper()

	m := regexp.MustCompile(expr).FindSubmatch(src)
	if m == nil {
		t.Fatalf("go.mod has no match for %s", expr)
	}

	return string(m[1])
}

func tail(b []byte) string {
	const keep = 6000
	if len(b) > keep {
		b = b[len(b)-keep:]
	}

	return string(b)
}
