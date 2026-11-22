package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatches(t *testing.T) {
	t.Parallel()

	h := hole

	cases := []struct {
		route, tmpl string
		want        bool
	}{
		{"/dashboard", "/dashboard", true},
		{"/dashboard", "/dashboard/", true},
		{"/{$}", "/", true},
		{"/cards/{card}", "/cards/" + h, true},
		{"/cards/{card}", "/cards/42", true},
		{"/cards/{card}/archive", "/cards/" + h, false},
		{"/cards/{card}/archive", "/cards/" + h + "/archive", true},
		{"/cards/{card}", "/cards/" + h + "/archive", false},
		{"/api/v1/accounts", "/api/v1/" + h, false},
		{"/api/v1/{rest...}", "/api/v1/nothing/here", true},
		{"/transactions/export.csv", "/transactions/export" + h, true},
		{"/transactions/export.csv", "/transactions/" + h, false},
		{"/a/{x}", "/b/1", false},
	}

	for _, c := range cases {
		if got := matches(c.route, c.tmpl); got != c.want {
			t.Errorf("matches(%q, %q) = %v, want %v", c.route, c.tmpl, got, c.want)
		}
	}
}

const sample = `package tests

import (
	"fmt"
	"net/http"
	"testing"
)

var table = []string{"accounts.archive"}

func TestSample(t *testing.T) {
	base := "/cards/" + id
	c.PostForm(fmt.Sprintf("/goals/%s/archive", gid), nil)
	c.PostForm(base+"/archive", nil)
	c.Do(http.MethodPut, "/workspace", nil, "")
	putForm(c, "/ai/settings", v)
	_ = "POST /import"
	_ = "DELETE /tags/" + id + "/x?y=1"
}
`

func TestIndexFindsRequestsAndStrings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample_test.go"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	ix, err := buildIndex(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct{ method, route string }{
		{"POST", "/cards/{card}/archive"}, {"POST", "/goals/{goal}/archive"}, {"PUT", "/workspace"},
		{"PUT", "/ai/settings"}, {"POST", "/import"}, {"DELETE", "/tags/{tag}/x"},
	} {
		if len(ix.coverRequest(c.method, c.route)) == 0 {
			t.Errorf("%s %s is not covered", c.method, c.route)
		}
	}

	if len(ix.coverRequest("DELETE", "/cards/{card}/archive")) != 0 {
		t.Error("a POST covered a DELETE")
	}

	if len(ix.coverString("accounts.archive")) == 0 {
		t.Error("a string in a package level table was not indexed")
	}

	if us := ix.coverString("accounts.archive"); us[0].name() != "sample_test.go" {
		t.Errorf("a table outside a test is named %q, want its file", us[0].name())
	}
}

func TestHiddenTestIsGone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample_test.go"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	ix, err := buildIndex(dir, map[string]bool{"TestSample": true})
	if err != nil {
		t.Fatal(err)
	}

	if len(ix.coverRequest("PUT", "/workspace")) != 0 || ix.tests["TestSample"] {
		t.Error("a hidden test still covers")
	}
}
