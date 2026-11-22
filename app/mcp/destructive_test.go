package mcp_test

import (
	"strings"
	"testing"

	vmcp "github.com/AlexGalhardo/vukem/mcp"
	"github.com/AlexGalhardo/vukem/mcp/mcptest"
)

// liveTransactions counts the transactions that are not deleted.
func (f *fixture) liveTransactions(t *testing.T) int {
	t.Helper()

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM transactions WHERE deleted_at IS NULL").Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

// seedTransaction records one expense and returns its uid.
func (f *fixture) seedTransaction(t *testing.T) string {
	t.Helper()

	acc := uidOf(t, f.seed(t, "accounts.create", map[string]any{"name": "Wallet", "kind": "cash"}))

	return uidOf(t, f.seed(t, "transactions.create", map[string]any{
		"account_uid": acc, "kind": "expense", "amount_cents": 1250, "date": "2026-11-20", "description": "Lunch",
	}))
}

func refused(rep mcptest.ToolReply) bool { return rep.Err != nil || rep.IsError }

func TestDestructiveGate(t *testing.T) {
	t.Parallel()

	answer := func(action string, asked *[]string) mcptest.Option {
		return mcptest.WithElicitation(func(message string) vmcp.ElicitResult {
			*asked = append(*asked, message)

			return vmcp.ElicitResult{Action: action}
		})
	}

	t.Run("off: the tool is not offered and a call changes nothing", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t, everything(false), nil)
		uid := f.seedTransaction(t)

		for _, n := range f.toolNames(t) {
			if n == "transactions_delete" {
				t.Fatal("transactions_delete is offered with the switch off")
			}
		}

		if rep := f.client.CallTool("transactions_delete", map[string]any{"uid": uid}); !refused(rep) {
			t.Fatalf("a destructive call was served: %s", rep.Text())
		}

		if n := f.liveTransactions(t); n != 1 {
			t.Fatalf("%d live transactions, want 1", n)
		}
	})

	t.Run("on and confirmed: the person is asked, then it runs", func(t *testing.T) {
		t.Parallel()

		var asked []string

		f := newFixture(t, everything(true), nil, answer("accept", &asked))
		uid := f.seedTransaction(t)

		rep := f.client.CallTool("transactions_delete", map[string]any{"uid": uid})
		if refused(rep) {
			t.Fatalf("a confirmed call failed: %v %s", rep.Err, rep.Text())
		}

		if len(asked) != 1 || !strings.Contains(asked[0], "transactions_delete") || !strings.Contains(asked[0], uid) {
			t.Fatalf("the person was asked %q, want one question naming the tool and the input", asked)
		}

		if n := f.liveTransactions(t); n != 0 {
			t.Fatalf("%d live transactions after a confirmed delete, want 0", n)
		}
	})

	t.Run("declined: nothing changes", func(t *testing.T) {
		t.Parallel()

		var asked []string

		f := newFixture(t, everything(true), nil, answer("decline", &asked))
		uid := f.seedTransaction(t)

		rep := f.client.CallTool("transactions_delete", map[string]any{"uid": uid})
		if !refused(rep) || !strings.Contains(rep.Text(), "declined") {
			t.Fatalf("a declined call: err=%v text=%q", rep.Err, rep.Text())
		}

		if n := f.liveTransactions(t); n != 1 {
			t.Fatalf("%d live transactions after a decline, want 1", n)
		}
	})

	t.Run("cancelled counts as no", func(t *testing.T) {
		t.Parallel()

		var asked []string

		f := newFixture(t, everything(true), nil, answer("cancel", &asked))
		uid := f.seedTransaction(t)

		if rep := f.client.CallTool("transactions_delete", map[string]any{"uid": uid}); !refused(rep) {
			t.Fatalf("a cancelled confirmation ran the tool: %s", rep.Text())
		}

		if n := f.liveTransactions(t); n != 1 {
			t.Fatalf("%d live transactions, want 1", n)
		}
	})

	t.Run("unsupported: a client that cannot ask gets a clear refusal", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t, everything(true), nil)
		uid := f.seedTransaction(t)

		rep := f.client.CallTool("transactions_delete", map[string]any{"uid": uid})
		if !refused(rep) {
			t.Fatalf("a client with no elicitation could delete: %s", rep.Text())
		}

		if !strings.Contains(rep.Text(), "cannot ask") {
			t.Fatalf("the refusal is %+v / %q, want it to say the client cannot ask", rep.Err, rep.Text())
		}

		if n := f.liveTransactions(t); n != 1 {
			t.Fatalf("%d live transactions, want 1", n)
		}
	})
}

func TestWritesAreRefusedByDefault(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)
	accounts := f.count(t, "accounts")

	for tool, args := range map[string]map[string]any{
		"accounts_create": {"name": "Sneaky", "kind": "cash"},
		"transactions_create": {
			"account_uid": "01JD5Z8K3M4N5P6Q7R8S9T0V1W", "kind": "expense", "amount_cents": 100, "date": "2026-11-20",
		},
		"transactions_delete": {"uid": "01JD5Z8K3M4N5P6Q7R8S9T0V1W"},
		"budgets_set":         {"category_uid": "01JD5Z8K3M4N5P6Q7R8S9T0V1W", "limit_cents": 100},
	} {
		if rep := f.client.CallTool(tool, args); !refused(rep) {
			t.Errorf("%s was served with the default settings: %s", tool, rep.Text())
		}
	}

	if n := f.count(t, "accounts"); n != accounts {
		t.Fatalf("accounts went from %d to %d", accounts, n)
	}
}
