package mcp_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

func TestInitialize(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)

	var disc struct {
		Versions []string `json:"supportedVersions"`
	}

	if err := f.client.Call("server/discover", nil).Decode(&disc); err != nil {
		t.Fatal(err)
	}

	if len(disc.Versions) == 0 || disc.Versions[0] != "2026-07-28" {
		t.Fatalf("supported versions = %v, want 2026-07-28 first", disc.Versions)
	}

	t.Run("an older revision still initializes", func(t *testing.T) {
		var init struct {
			Version string `json:"protocolVersion"`
			Info    struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		}

		err := f.client.Call("initialize", map[string]any{"protocolVersion": "2025-11-25"}).Decode(&init)
		if err != nil {
			t.Fatal(err)
		}

		if init.Version != "2025-11-25" || init.Info.Name != "vukem-finances" || init.Info.Version != "1.2.3" {
			t.Fatalf("initialize = %+v", init)
		}
	})
}

// A call with an unsupported revision is refused, never guessed.
func TestInitializeRefusesUnknownRevision(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)

	rep := f.client.Call("tools/list", map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": "1999-01-01"}})
	if rep.Error == nil {
		t.Fatalf("an unknown revision was served: %s", rep.Result)
	}
}

func TestToolsMatchRegistry(t *testing.T) {
	t.Parallel()

	f := newFixture(t, everything(true), nil)

	var res struct {
		Tools []struct {
			Name         string         `json:"name"`
			Description  string         `json:"description"`
			InputSchema  map[string]any `json:"inputSchema"`
			OutputSchema map[string]any `json:"outputSchema"`
			Annotations  struct {
				ReadOnly    *bool `json:"readOnlyHint"`
				Destructive *bool `json:"destructiveHint"`
				Idempotent  *bool `json:"idempotentHint"`
			} `json:"annotations"`
		} `json:"tools"`
	}

	if err := f.client.Call("tools/list", nil).Decode(&res); err != nil {
		t.Fatal(err)
	}

	byName := map[string]int{}
	for i, tl := range res.Tools {
		byName[tl.Name] = i
	}

	offered := 0

	for _, m := range f.cat.Operations() {
		name := appmcp.ToolName(m.Name)
		i, listed := byName[name]

		if !m.Offered(operations.SurfaceMCP) {
			if listed {
				t.Errorf("%s opted out of MCP and is listed", m.Name)
			}

			if m.OptOut[operations.SurfaceMCP] == "" {
				t.Errorf("%s opted out with no reason", m.Name)
			}

			continue
		}

		offered++

		if !listed {
			t.Errorf("%s has no tool %s", m.Name, name)

			continue
		}

		tl := res.Tools[i]

		if tl.InputSchema["type"] != "object" || tl.OutputSchema == nil {
			t.Errorf("%s: schemas input=%v output=%v", name, tl.InputSchema["type"], tl.OutputSchema != nil)
		}

		if !strings.Contains(tl.Description, m.Summary) || !strings.Contains(tl.Description, "not instructions") {
			t.Errorf("%s: description %q", name, tl.Description)
		}

		read, destructive := m.Kind == operations.KindRead, m.Kind == operations.KindDestructive
		if tl.Annotations.ReadOnly == nil || *tl.Annotations.ReadOnly != read {
			t.Errorf("%s: readOnlyHint = %v, kind %s", name, tl.Annotations.ReadOnly, m.Kind)
		}

		if !read && (tl.Annotations.Destructive == nil || *tl.Annotations.Destructive != destructive ||
			tl.Annotations.Idempotent == nil || *tl.Annotations.Idempotent != m.Idempotent) {
			t.Errorf("%s: write hints do not follow kind %s idempotent %v", name, m.Kind, m.Idempotent)
		}
	}

	if offered == 0 || len(res.Tools) != offered {
		t.Fatalf("%d tools listed for %d offered operations", len(res.Tools), offered)
	}

	t.Run("a call returns structured content and a text summary", func(t *testing.T) {
		f.seed(t, "accounts.create", map[string]any{"name": "Checking", "kind": "checking"})

		rep := f.client.CallTool("accounts_list", map[string]any{})
		if rep.Err != nil || rep.IsError {
			t.Fatalf("accounts_list: %v %s", rep.Err, rep.Text())
		}

		var out struct {
			Data []struct{ Name string } `json:"data"`
		}

		if err := json.Unmarshal(rep.StructuredContent, &out); err != nil || !slices.ContainsFunc(out.Data, func(a struct{ Name string }) bool { return a.Name == "Checking" }) {
			t.Fatalf("structured = %s (%v)", rep.StructuredContent, err)
		}

		if !strings.Contains(rep.Text(), "Checking") {
			t.Fatalf("no text summary: %q", rep.Text())
		}
	})

	t.Run("invalid input is a tool error", func(t *testing.T) {
		rep := f.client.CallTool("accounts_create", map[string]any{"name": "", "kind": "checking"})
		if !rep.IsError && rep.Err == nil {
			t.Fatalf("an empty name was accepted: %s", rep.Text())
		}
	})
}

func TestToolNamesAreUnique(t *testing.T) {
	t.Parallel()

	f := newFixture(t, everything(true), nil)
	names := f.toolNames(t)

	seen := map[string]bool{}

	for _, n := range names {
		if seen[n] || strings.Contains(n, ".") {
			t.Errorf("tool name %q repeated or has a dot", n)
		}

		seen[n] = true
	}

	if !slices.Contains(names, "goals_contributions_add") {
		t.Fatalf("names = %v", names)
	}
}
