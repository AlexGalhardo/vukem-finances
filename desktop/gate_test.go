package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func shellServer(t *testing.T, choose func(string) (string, error), open func(string) error) (*Server, *http.Client) {
	t.Helper()

	paths, err := ResolvePaths(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	sh := &shell{tmpDir: paths.Data, openURL: open, chooseSavePath: choose}
	s := startTestServer(t, paths, startOptions{shell: sh})

	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: noRedirect, Timeout: 20 * time.Second}

	if got := code(t, c, s.EntryURL()); got != http.StatusSeeOther {
		t.Fatalf("entry = %d, want 303", got)
	}

	return s, c
}

func post(t *testing.T, c *http.Client, url string, body []byte) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	return res
}

func TestDesktopSaveRoute(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "chosen.csv")

	var asked string

	s, c := shellServer(t, func(name string) (string, error) { asked = name; return dest, nil }, nil)

	data := []byte("date,amount\n2026-01-01,10.00\n")

	res := post(t, c, s.Origin()+"/desktop/save?name=../../etc/transactions.csv", data)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save = %d, want 200", res.StatusCode)
	}

	if asked != "transactions.csv" {
		t.Errorf("suggested name = %q, want a plain file name", asked)
	}

	if got, _ := os.ReadFile(dest); !bytes.Equal(got, data) {
		t.Errorf("saved file = %q, want the posted bytes", got)
	}

	// No temp file is left behind.
	if m, _ := filepath.Glob(filepath.Join(filepath.Dir(s.paths.Key), "save-*")); len(m) != 0 {
		t.Errorf("left behind: %v", m)
	}

	// Without the cookie, or from another origin, the route is closed.
	if res := post(t, &http.Client{Timeout: 10 * time.Second}, s.Origin()+"/desktop/save?name=x.csv", data); res.StatusCode != http.StatusForbidden {
		t.Errorf("save without the cookie = %d, want 403", res.StatusCode)
	}

	// GET is not an action.
	if got := code(t, c, s.Origin()+"/desktop/save"); got != http.StatusNotFound {
		t.Errorf("GET /desktop/save = %d, want 404", got)
	}
}

func TestDesktopSaveRouteCancelledWritesNothing(t *testing.T) {
	s, c := shellServer(t, func(string) (string, error) { return "", nil }, nil)

	if res := post(t, c, s.Origin()+"/desktop/save?name=x.csv", []byte("a")); res.StatusCode != http.StatusNoContent {
		t.Fatalf("cancelled save = %d, want 204", res.StatusCode)
	}
}

func TestDesktopOpenExternalOnlyWebAddresses(t *testing.T) {
	var opened []string

	s, c := shellServer(t, nil, func(u string) error { opened = append(opened, u); return nil })

	for _, ok := range []string{"https://example.com/docs", "http://example.com", "mailto:ana@example.com"} {
		if res := post(t, c, s.Origin()+"/desktop/open-external", []byte(ok)); res.StatusCode != http.StatusNoContent {
			t.Errorf("open %s = %d, want 204", ok, res.StatusCode)
		}
	}

	for _, bad := range []string{"file:///c:/windows/system32/calc.exe", "javascript:alert(1)", "ms-msdt:/id", "calc.exe", strings.Repeat("a", 5000)} {
		if res := post(t, c, s.Origin()+"/desktop/open-external", []byte(bad)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("open %.20q = %d, want 400", bad, res.StatusCode)
		}
	}

	if len(opened) != 3 {
		t.Errorf("opened = %v, want only the 3 web addresses", opened)
	}
}

func TestSafeFileName(t *testing.T) {
	for in, want := range map[string]string{
		"a.csv": "a.csv", "../../x.csv": "x.csv", `C:\temp\y.csv`: "y.csv", "": "download", "..": "download", "/": "download",
	} {
		if got := safeFileName(in); got != want {
			t.Errorf("safeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestShellInjectsTheScriptIntoHTMLPages(t *testing.T) {
	s, c := shellServer(t, nil, nil)

	get := func(path, accept string) (*http.Response, string) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.Origin()+path, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Accept", accept)

		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)

		return res, string(body)
	}

	res, body := get("/setup", "text/html")
	if !strings.Contains(body, scriptTag+"</head>") {
		t.Errorf("the setup page has no shell script tag before </head>")
	}

	if res.Header.Get("Content-Length") != strconv.Itoa(len(body)) {
		t.Errorf("Content-Length = %q, body is %d bytes", res.Header.Get("Content-Length"), len(body))
	}

	if res.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", res.Header.Get("Referrer-Policy"))
	}

	if _, js := get("/desktop/shell.js", "*/*"); !strings.Contains(js, "__VF_DESKTOP__") {
		t.Error("/desktop/shell.js does not serve the shell script")
	}

	if _, asset := get("/build/manifest.json", "*/*"); strings.Contains(asset, scriptTag) {
		t.Error("a non-HTML response was rewritten")
	}

	if _, page := get("/setup", "application/json"); strings.Contains(page, scriptTag) {
		t.Error("a response to a non-HTML request was rewritten")
	}
}
