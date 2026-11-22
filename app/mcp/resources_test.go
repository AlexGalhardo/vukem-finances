package mcp_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func (f *fixture) read(t *testing.T, uri string) (string, bool) {
	t.Helper()

	rep := f.client.Call("resources/read", map[string]any{"uri": uri})
	if rep.Error != nil {
		return rep.Error.Message, false
	}

	var res struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(rep.Result, &res); err != nil || len(res.Contents) != 1 {
		t.Fatalf("read %s: %v %s", uri, err, rep.Result)
	}

	if res.Contents[0].URI != uri || res.Contents[0].MIMEType != "application/json" {
		t.Fatalf("read %s: %+v", uri, res.Contents[0])
	}

	return res.Contents[0].Text, true
}

func TestResources(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)
	f.seedTransaction(t)

	var list struct {
		Resources []struct {
			URI string `json:"uri"`
		} `json:"resources"`
	}

	if err := f.client.Call("resources/list", nil).Decode(&list); err != nil {
		t.Fatal(err)
	}

	var uris []string
	for _, r := range list.Resources {
		uris = append(uris, r.URI)
	}

	slices.Sort(uris)

	if want := []string{"finances://accounts", "finances://categories", "finances://workspace"}; !slices.Equal(uris, want) {
		t.Fatalf("resources = %v, want %v", uris, want)
	}

	var tpl struct {
		Templates []struct {
			URITemplate string `json:"uriTemplate"`
		} `json:"resourceTemplates"`
	}

	if err := f.client.Call("resources/templates/list", nil).Decode(&tpl); err != nil {
		t.Fatal(err)
	}

	var templates []string
	for _, r := range tpl.Templates {
		templates = append(templates, r.URITemplate)
	}

	slices.Sort(templates)

	if want := []string{"finances://budgets/{month}", "finances://reports/{kind}/{period}"}; !slices.Equal(templates, want) {
		t.Fatalf("templates = %v, want %v", templates, want)
	}

	readable := []struct{ uri, contains string }{
		{"finances://workspace", "net_worth"},
		{"finances://accounts", "Wallet"},
		{"finances://categories", "data"},
		{"finances://budgets/2026-11", "data"},
		{"finances://reports/cashflow/this_month", "totals"},
		{"finances://reports/categories/last_3_months", "rows"},
		{"finances://reports/compare/2026-01-01_2026-03-31", "period"},
	}

	for _, r := range readable {
		text, ok := f.read(t, r.uri)
		if !ok || !strings.Contains(text, r.contains) {
			t.Errorf("%s: ok=%v, want %q in %.200s", r.uri, ok, r.contains, text)
		}
	}

	for _, uri := range []string{
		"finances://budgets/november",
		"finances://budgets/2026-13",
		"finances://reports/bogus/this_month",
		"finances://reports/cashflow/forever",
		"finances://reports/cashflow/2026-03-31_2026-01-01x",
		"finances://elsewhere",
	} {
		if text, ok := f.read(t, uri); ok {
			t.Errorf("%s was served: %.200s", uri, text)
		}
	}
}

// A resource is a read operation: a toolset or a switch never gives it write.
func TestResourcesAreReadOnly(t *testing.T) {
	t.Parallel()

	f := newFixture(t, everything(true), nil)

	for _, method := range []string{"resources/write", "resources/update", "resources/delete"} {
		if rep := f.client.Call(method, map[string]any{"uri": "finances://accounts"}); rep.Error == nil {
			t.Errorf("%s answered: %s", method, rep.Result)
		}
	}
}
