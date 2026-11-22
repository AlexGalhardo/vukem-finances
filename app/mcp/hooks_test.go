package mcp_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/mcp/mcptest"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

type hookLog struct {
	mu     sync.Mutex
	before []string
	done   []string
	refuse error
}

func (h *hookLog) Before(_ context.Context, m operations.Meta, _ operations.Actor) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.before = append(h.before, m.Name)

	return h.refuse
}

func (h *hookLog) Done(_ context.Context, m operations.Meta, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	outcome := "ok"
	if err != nil {
		outcome = "refused"
	}

	h.done = append(h.done, m.Name+":"+outcome)
}

func (h *hookLog) snapshot() (before, done []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]string(nil), h.before...), append([]string(nil), h.done...)
}

func (h *hookLog) set(err error) {
	h.mu.Lock()
	h.refuse = err
	h.mu.Unlock()
}

// hooked is a server over the fixture's catalog with recording hooks.
func hooked(t *testing.T, f *fixture) (*hookLog, *mcptest.Client) {
	t.Helper()

	ws, err := workspaces.New(f.app.App)
	if err != nil {
		t.Fatal(err)
	}

	h := &hookLog{}

	srv, err := appmcp.New(appmcp.Options{
		Catalog: f.cat, Actor: appmcp.LocalActor(ws, f.cat, readOnly()), Config: readOnly(), PageSize: 500,
		Before: h.Before, Done: h.Done,
	})
	if err != nil {
		t.Fatal(err)
	}

	return h, mcptest.NewClient(t, srv)
}

// Every path that reaches Invoke passes the rate limit and the audit hooks: a
// tool call, a resource read and a completion (a prompt only builds text).
func TestEveryEntryPointPassesBothHooks(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)
	h, c := hooked(t, f)

	if rep := c.CallTool("accounts_list", map[string]any{}); rep.Err != nil || rep.IsError {
		t.Fatalf("tool: %v %s", rep.Err, rep.Text())
	}

	if rep := c.Call("resources/read", map[string]any{"uri": "finances://accounts"}); rep.Error != nil {
		t.Fatalf("resource: %v", rep.Error)
	}

	if rep := c.Call("resources/read", map[string]any{"uri": "finances://reports/cashflow/this_month"}); rep.Error != nil {
		t.Fatalf("report resource: %v", rep.Error)
	}

	if rep := c.Call("completion/complete", map[string]any{
		"ref":      map[string]any{"type": "ref/prompt", "name": "import-statement"},
		"argument": map[string]any{"name": "account", "value": ""},
	}); rep.Error != nil {
		t.Fatalf("completion: %v", rep.Error)
	}

	if rep := c.Call("prompts/get", map[string]any{"name": "budget-plan", "arguments": map[string]any{}}); rep.Error != nil {
		t.Fatalf("prompt: %v", rep.Error)
	}

	before, done := h.snapshot()

	// The prompt reads nothing, so it adds no entry: tool, two resources, one completion.
	wantBefore := []string{"accounts.list", "accounts.list", "reports.cashflow", "accounts.list"}
	wantDone := []string{"accounts.list:ok", "accounts.list:ok", "reports.cashflow:ok", "accounts.list:ok"}

	if strings.Join(before, ",") != strings.Join(wantBefore, ",") || strings.Join(done, ",") != strings.Join(wantDone, ",") {
		t.Fatalf("before = %v, done = %v", before, done)
	}
}

func TestResourceReadIsRateLimitedAndAudited(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)
	h, c := hooked(t, f)

	h.set(&operations.Error{Code: operations.CodeRateLimited, Message: "Too many calls: try again in 30 seconds."})

	t.Run("a rate limit surfaces as one", func(t *testing.T) {
		rep := c.Call("resources/read", map[string]any{"uri": "finances://accounts"})
		if rep.Error != nil && strings.Contains(rep.Error.Message, "not found") {
			t.Fatalf("a rate limit was reported as not found: %v", rep.Error)
		}

		msg := string(rep.Result)
		if rep.Error != nil {
			msg += rep.Error.Message
		}

		if !strings.Contains(msg, "rate_limited") {
			t.Fatalf("the refusal does not say rate_limited: %s %v", rep.Result, rep.Error)
		}

		if rep := c.CallTool("accounts_list", map[string]any{}); !strings.Contains(rep.Text(), "rate_limited") {
			t.Fatalf("the tool is not limited the same way: %q", rep.Text())
		}
	})

	t.Run("any other refusal looks like a missing resource", func(t *testing.T) {
		h.set(&operations.Error{Code: operations.CodeForbidden, Message: "no"})

		rep := c.Call("resources/read", map[string]any{"uri": "finances://accounts"})
		if rep.Error == nil || !strings.Contains(rep.Error.Message, "not found") {
			t.Fatalf("a forbidden read answered %s %v", rep.Result, rep.Error)
		}
	})

	t.Run("a refusal is audited like an allowed read", func(t *testing.T) {
		_, done := h.snapshot()

		refused := 0

		for _, d := range done {
			if d == "accounts.list:refused" {
				refused++
			}
		}

		if refused != 3 {
			t.Fatalf("done = %v, want three refused accounts.list entries (resource, tool, resource)", done)
		}
	})
}
