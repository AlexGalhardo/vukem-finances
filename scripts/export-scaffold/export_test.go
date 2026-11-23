package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

type data struct{ Module, Name string }

func render(t *testing.T, src []byte, d data) string {
	t.Helper()

	tpl, err := template.New("f").Option("missingkey=error").Parse(string(src))
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, d); err != nil {
		t.Fatal(err)
	}

	return b.String()
}

func TestPlaceholdersSurviveRendering(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"import":          `import "github.com/AlexGalhardo/vukem-finances/app"`,
		"go template":     `x := "{{ .Title }}" + "github.com/AlexGalhardo/vukem-finances"`,
		"jsx":             `export const a = <div style={{ color: "red" }}>vukem-finances</div>`,
		"brace by action": `{vukem-finances} {github.com/AlexGalhardo/vukem-finances}`,
		"triple":          `{{{ vukem-finances }}}`,
		"trailing brace":  "vukem-finances {{",
		"leading braces":  "}}{{ vukem-finances",
		"comment action":  `{{/* c */}} vukem-finances {{- end }}`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, changed := Placeholders([]byte(src))
			if !changed {
				t.Fatal("no placeholder written")
			}

			want := strings.ReplaceAll(strings.ReplaceAll(src, modulePath, "example.test/money"), appName, "money")
			if got := render(t, out, data{"example.test/money", "money"}); got != want {
				t.Errorf("rendered %q, want %q", got, want)
			}
		})
	}
}

func TestPlaceholdersKeepOwnReferences(t *testing.T) {
	t.Parallel()

	src := "see https://github.com/AlexGalhardo/vukem-finances/releases and ghcr.io/alexgalhardo/vukem-finances:1 {{ x }}"

	out, changed := Placeholders([]byte(src))
	if changed || string(out) != src {
		t.Errorf("got %q changed=%v, want the text untouched", out, changed)
	}
}

func TestInclude(t *testing.T) {
	t.Parallel()

	for p, want := range map[string]bool{
		"app/finance/summary.go":        true,
		"README.md":                     true,
		"package.json":                  true,
		"bun.lock":                      true,
		"mobile/app.json":               false,
		"desktop/main.go":               false,
		"site/index.html":               false,
		".github/workflows/ci.yml":      false,
		".agents/rules/ci.md":           false,
		".claude/agents/x.md":           false,
		"development/STATE.md":          false,
		"docs/qa/round.md":              false,
		"docs/qa/inventory.md":          true,
		"site/generated/plans.json":     true,
		"docs/offline.md":               true,
		"CHANGELOG.md":                  false,
		"go.mod":                        false,
		"go.sum":                        false,
		"LICENSE":                       false,
		"scripts/qa/round.sh":           false,
		"resources/public/build/app.js": false,
		"resources/public/robots.txt":   true,
		"storage/.gitkeep":              false,
		"resources/images/logo.png":     false,
		"resources/images/LOGO.SVG":     false,
	} {
		if got := Include(p); got != want {
			t.Errorf("Include(%q) = %v, want %v", p, got, want)
		}
	}
}

func sample() []File {
	return []File{
		{Path: "go.mod", Mode: 0o644, Data: []byte("module " + modulePath + "\n")},
		{Path: "LICENSE", Mode: 0o644, Data: []byte("MIT vukem-finances\n")},
		{Path: "app/a.go", Mode: 0o644, Data: []byte("package a\n\nimport _ \"" + modulePath + "/b\"\n")},
		{Path: "app/b.go", Mode: 0o644, Data: []byte("package b\n\nvar s = \"{{ keep }}\"\n")},
		{Path: "scripts/x.sh", Mode: 0o755, Data: []byte("#!/bin/sh\n")},
		{Path: "docs/guide.md", Mode: 0o644, Data: []byte("# vukem-finances\n")},
		{Path: "run.sh", Mode: 0o755, Data: []byte("#!/bin/sh\necho vukem-finances\n")},
		{Path: "img.bin", Mode: 0o644, Data: []byte("vukem-finances\x00")},
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()

	files, err := Build(sample(), "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	byPath := map[string]File{}
	for _, f := range files {
		byPath[f.Path] = f
	}

	for _, p := range []string{"app/a.go.tmpl", "app/b.go", "docs/guide.md", "run.sh.tmpl", "img.bin", "VERSION", "LICENSE.vukem-finances", "MANIFEST.sha256"} {
		if _, ok := byPath[p]; !ok {
			t.Errorf("%s missing from the tree", p)
		}
	}

	for _, p := range []string{"go.mod", "LICENSE", "scripts/x.sh"} {
		if _, ok := byPath[p]; ok {
			t.Errorf("%s must not be exported", p)
		}
	}

	if byPath["run.sh.tmpl"].Mode != 0o755 {
		t.Errorf("run.sh lost its executable bit")
	}

	if string(byPath["VERSION"].Data) != "v1.2.3\n" {
		t.Errorf("VERSION = %q", byPath["VERSION"].Data)
	}

	var want bytes.Buffer

	for _, f := range files {
		if f.Path != manifestName {
			want.WriteString(hashHex(f.Data) + "  " + f.Path + "\n")
		}
	}

	if got := byPath[manifestName].Data; !bytes.Equal(got, want.Bytes()) {
		t.Errorf("manifest does not list the files:\n%s", got)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	t.Parallel()

	var archives [2][]byte

	for i := range archives {
		in := sample()
		// Input order must not matter.
		in[0], in[len(in)-1] = in[len(in)-1], in[0]

		files, err := Build(in, "v1.2.3")
		if err != nil {
			t.Fatal(err)
		}

		var b bytes.Buffer
		if err := WriteTar(&b, files); err != nil {
			t.Fatal(err)
		}

		archives[i] = b.Bytes()
	}

	if !bytes.Equal(archives[0], archives[1]) {
		t.Error("two builds gave different archives")
	}
}

func TestBuildNeedsLicense(t *testing.T) {
	t.Parallel()

	if _, err := Build([]File{{Path: "a.go", Data: []byte("package a")}}, "v1"); err == nil {
		t.Error("expected an error without a LICENSE")
	}
}
