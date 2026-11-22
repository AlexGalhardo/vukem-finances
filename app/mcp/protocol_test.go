package mcp_test

import (
	"context"
	"encoding/json"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

type slowOut struct {
	Waited bool `json:"waited"`
}

// slowOperation is a read that waits for its context, so a test can cancel it.
func slowOperation(started chan<- struct{}, stopped *atomic.Bool) func(*operations.Catalog) {
	return func(cat *operations.Catalog) {
		operations.Define(cat.Registry, operations.Operation[operations.NoInput, slowOut]{
			Name: "slow.wait", Summary: "Wait until cancelled", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead,
			Run: func(ctx context.Context, _ operations.Actor, _ operations.NoInput) (slowOut, error) {
				started <- struct{}{}

				select {
				case <-ctx.Done():
					stopped.Store(true)

					return slowOut{}, ctx.Err()
				case <-time.After(10 * time.Second):
					return slowOut{Waited: true}, nil
				}
			},
		})
	}
}

func TestPagination(t *testing.T) {
	t.Parallel()

	f := newSizedFixture(t, everything(true), 7, nil)

	var all []string

	cursor := ""
	pages := 0

	for {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}

		var res struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
			Next string `json:"nextCursor"`
		}

		if err := f.client.Call("tools/list", params).Decode(&res); err != nil {
			t.Fatal(err)
		}

		if len(res.Tools) > 7 {
			t.Fatalf("a page has %d tools, want at most 7", len(res.Tools))
		}

		for _, tl := range res.Tools {
			all = append(all, tl.Name)
		}

		pages++

		if res.Next == "" {
			break
		}

		cursor = res.Next
	}

	if pages < 2 {
		t.Fatalf("%d pages: the list was not paged", pages)
	}

	if !slices.IsSorted(all) || len(slices.Compact(slices.Clone(all))) != len(all) {
		t.Fatalf("pages overlap or are out of order: %v", all)
	}

	if len(all) <= 7 {
		t.Fatalf("paged only %d tools", len(all))
	}

	t.Run("a cursor that is not ours is refused", func(t *testing.T) {
		for _, bad := range []string{"!!!", "not base64 at all", "%%"} {
			if rep := f.client.Call("tools/list", map[string]any{"cursor": bad}); rep.Error == nil {
				t.Errorf("cursor %q was accepted", bad)
			}
		}
	})
}

func TestProgress(t *testing.T) {
	t.Parallel()

	f := newFixture(t, readOnly(), nil)

	rep := f.client.Call("tools/call", map[string]any{
		"name": "reports_cashflow", "arguments": map[string]any{},
		"_meta": map[string]any{"progressToken": "report-1"},
	})
	if rep.Error != nil {
		t.Fatal(rep.Error)
	}

	var progress []float64

	for _, n := range f.client.Notifications() {
		if n.Method != "notifications/progress" {
			continue
		}

		var p struct {
			Token    string  `json:"progressToken"`
			Progress float64 `json:"progress"`
		}

		if err := json.Unmarshal(n.Params, &p); err != nil || p.Token != "report-1" {
			t.Fatalf("progress notification %s (%v)", n.Params, err)
		}

		progress = append(progress, p.Progress)
	}

	if len(progress) < 2 || !slices.IsSorted(progress) || progress[0] >= progress[len(progress)-1] {
		t.Fatalf("progress = %v, want it to increase", progress)
	}

	t.Run("a call with no token sends none", func(t *testing.T) {
		before := len(f.client.Notifications())

		if rep := f.client.CallTool("reports_cashflow", map[string]any{}); rep.Err != nil {
			t.Fatal(rep.Err)
		}

		if after := len(f.client.Notifications()); after != before {
			t.Fatalf("%d notifications with no progress token", after-before)
		}
	})
}

func TestCancel(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 1)

	var stopped atomic.Bool

	f := newFixture(t, readOnly(), slowOperation(started, &stopped))

	id := f.client.Start("tools/call", map[string]any{"name": "slow_wait", "arguments": map[string]any{}})

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the operation never started")
	}

	f.client.Cancel(id)

	deadline := time.Now().Add(5 * time.Second)
	for !stopped.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if !stopped.Load() {
		t.Fatal("cancelling the request did not cancel the context of the operation")
	}
}

// The server keeps serving after a cancelled call.
func TestCancelLeavesTheServerWorking(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 1)

	var stopped atomic.Bool

	f := newFixture(t, readOnly(), slowOperation(started, &stopped))

	id := f.client.Start("tools/call", map[string]any{"name": "slow_wait", "arguments": map[string]any{}})
	<-started
	f.client.Cancel(id)

	if rep := f.client.CallTool("accounts_list", map[string]any{}); rep.Err != nil || rep.IsError {
		t.Fatalf("after a cancel: %v %s", rep.Err, rep.Text())
	}
}
