package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodPage = `<!doctype html><html lang="en"><head><title>t</title>` +
	`<link rel="canonical" href="https://finances.vukem.dev/">` +
	`<link rel="stylesheet" href="/_astro/a.css"></head>` +
	`<body><a href="/docs/">Docs</a><a href="https://github.com/AlexGalhardo/vukem-finances">GitHub</a>` +
	`<img src="/x.png" alt="x"><p>Self-hosted and open source.</p></body></html>`

// site writes a clean site, applies change, and returns the problems found.
func site(t *testing.T, change func(dir string)) string {
	t.Helper()

	dir := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.html", goodPage)
	write("CNAME", "finances.vukem.dev\n")
	write("_astro/a.css", "body{background:url(/bg.png)}")
	write("_astro/a.js", "console.log(1)")
	// Pagefind reads its own static index: its folder is the allowlist entry.
	write("pagefind/pagefind.js", "fetch('/pagefind/index/x.pf_index')")
	write("pagefind/pagefind-worker.js", "")

	if change != nil {
		change(dir)
	}

	problems, err := Scan(dir, Options{Origin: "https://finances.vukem.dev", CNAME: "finances.vukem.dev"})
	if err != nil {
		t.Fatal(err)
	}

	return strings.Join(problems, "\n")
}

func edit(t *testing.T, rel, old, repl string) func(string) {
	t.Helper()

	return func(dir string) {
		p := filepath.Join(dir, rel)

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, repl, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func put(t *testing.T, rel, body string) func(string) {
	t.Helper()

	return func(dir string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCleanSiteHasNoProblem(t *testing.T) {
	t.Parallel()

	if got := site(t, nil); got != "" {
		t.Fatalf("a clean site has problems:\n%s", got)
	}
}

func TestPlantedFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		change func(*testing.T) func(string)
		want   string
	}{
		{"a form", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</body>", `<form action="/x"></form></body>`)
		}, "<form"},
		{"an external script", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</head>", `<script src="https://cdn.example.net/a.js"></script></head>`)
		}, "another origin"},
		{"an external stylesheet", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</head>", `<link rel="stylesheet" href="//fonts.example.net/a.css"></head>`)
		}, "another origin"},
		{"an external image", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</body>", `<img src="http://img.example.net/a.png" alt="x"></body>`)
		}, "another origin"},
		{"a CSS url to another origin", func(t *testing.T) func(string) {
			return put(t, "_astro/a.css", "body{background:url(https://x.example.net/a.png)}")
		}, "another origin"},
		{"a fetch to an API path", func(t *testing.T) func(string) {
			return put(t, "_astro/a.js", "fetch('/api/v1/me')")
		}, "network API"},
		{"an inline fetch", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</body>", `<script>fetch('/api/x')</script></body>`)
		}, "network API"},
		{"a WebSocket", func(t *testing.T) func(string) {
			return put(t, "_astro/a.js", "new WebSocket('wss://x')")
		}, "network API"},
		{"a login link", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</body>", `<a href="/login">Sign in</a></body>`)
		}, "page of an app"},
		{"an absolute login link to the own origin", func(t *testing.T) func(string) {
			return edit(t, "index.html", "</body>", `<a href="https://finances.vukem.dev/register/">x</a></body>`)
		}, "page of an app"},
		{"a hosted-service phrase", func(t *testing.T) func(string) {
			return edit(t, "index.html", "open source.", "open source. Start free today.")
		}, "start free"},
		{"a worker file", func(t *testing.T) func(string) { return put(t, "_worker.js", "") }, "server or worker"},
		{"a server entry", func(t *testing.T) func(string) { return put(t, "_astro/server.mjs", "") }, "server or worker"},
		{"a missing CNAME", func(t *testing.T) func(string) {
			return func(dir string) { _ = os.Remove(filepath.Join(dir, "CNAME")) }
		}, "CNAME: missing"},
		{"a wrong CNAME", func(t *testing.T) func(string) { return put(t, "CNAME", "other.example.com") }, "want"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := site(t, tc.change(t)); !strings.Contains(got, tc.want) {
				t.Fatalf("want a problem containing %q, got:\n%s", tc.want, got)
			}
		})
	}
}

func TestDocsMayMentionLoginAndPagefindMayFetch(t *testing.T) {
	t.Parallel()

	got := site(t, put(t, "docs/cli/index.html", `<html lang="en"><body><p>Run the login command.</p></body></html>`))
	if got != "" {
		t.Fatalf("docs describe the CLI login command: %s", got)
	}
}

func TestRunExitCodes(t *testing.T) {
	t.Parallel()

	var out, errb bytes.Buffer
	empty := t.TempDir()
	if code := run([]string{empty}, &out, &errb); code != 1 {
		t.Fatalf("a directory without CNAME: code %d, want 1", code)
	}

	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("no argument: code %d, want 2", code)
	}

	if code := run([]string{filepath.Join(t.TempDir(), "missing")}, &out, &errb); code != 2 {
		t.Fatalf("missing dir: code %d, want 2", code)
	}
}
