package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/AlexGalhardo/vukem-finances/scripts/qa/qakit"
)

type harness struct {
	cfg  config
	base string
	ws   []qakit.Workspace
	bin  string
	env  []string

	http    *http.Client
	devices []string // per client, on the big workspace
}

func (h *harness) big() qakit.Workspace { return h.ws[0] }

// call sends one API request as client c of workspace w and returns the body.
func (h *harness) call(ctx context.Context, w qakit.Workspace, c int, method, path string, body any, headers map[string]string) ([]byte, error) {
	var rdr io.Reader

	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		rdr = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, h.base+path, rdr)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+w.Tokens[c%len(w.Tokens)])
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return h.do(req)
}

func (h *harness) do(req *http.Request) ([]byte, error) {
	res, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		snippet := string(raw)
		if len(snippet) > 160 {
			snippet = snippet[:160]
		}

		return nil, fmt.Errorf("%s %s: status %d: %s", req.Method, req.URL.Path, res.StatusCode, strings.TrimSpace(snippet))
	}

	return raw, nil
}

// measure runs total jobs over clients workers and times each one.
func (h *harness) measure(name string, total, clients int, job func(client, i int) error) result {
	rc := &recorder{r: result{Name: name}}
	jobs := make(chan int)

	var wg sync.WaitGroup

	start := time.Now()

	for c := range clients {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range jobs {
				t := time.Now()
				if err := job(c, i); err != nil {
					rc.fail(err.Error())
					continue
				}

				rc.ok(time.Since(t))
			}
		}()
	}

	for i := range total {
		jobs <- i
	}

	close(jobs)
	wg.Wait()

	rc.r.Elapsed = time.Since(start)
	fmt.Printf("  %-26s %5d requests in %s\n", name, total, rc.r.Elapsed.Round(time.Millisecond))

	return rc.r
}

func (h *harness) runAll(ctx context.Context) ([]result, error) {
	h.http = &http.Client{
		Timeout:   2 * time.Minute,
		Transport: &http.Transport{MaxIdleConnsPerHost: 64, MaxConnsPerHost: 64},
	}

	if err := h.registerDevices(ctx); err != nil {
		return nil, err
	}

	web, err := h.webLogin(ctx)
	if err != nil {
		return nil, err
	}

	n, cl := h.cfg.requests, h.cfg.clients
	big := h.big()
	now := time.Now().UTC()
	from12 := now.AddDate(-1, 0, 0).Format("2006-01-02")
	to := now.Format("2006-01-02")

	fmt.Println("measuring")

	var out []result

	out = append(out, h.measure("dashboard", n, cl, func(_, _ int) error {
		return web.get(ctx, "/dashboard")
	}))

	out = append(out, h.measure("transactions_list", n, cl, func(c, i int) error {
		_, err := h.call(ctx, big, c, "GET", fmt.Sprintf("/api/v1/transactions?page=%d", 1+i%10), nil, nil)
		return err
	}))

	out = append(out, h.measure("transactions_filtered", n, cl, func(c, i int) error {
		q := url.Values{"from": {from12}, "to": {to}, "kind": {"expense"}, "q": {fmt.Sprintf("shop %d", i%500)}, "account": {big.Accounts[i%3]}}
		_, err := h.call(ctx, big, c, "GET", "/api/v1/transactions?"+q.Encode(), nil, nil)

		return err
	}))

	reports := []string{"cash-flow", "categories", "accounts", "net-worth", "compare"}
	out = append(out, h.measure("reports", n, cl, func(c, i int) error {
		_, err := h.call(ctx, big, c, "GET", "/api/v1/reports/"+reports[i%len(reports)]+"?period=last_12_months", nil, nil)
		return err
	}))

	// Downloads share one limit of ExportLimit a minute per workspace.
	out = append(out, h.measure("export_csv_50k", 8, 8, func(c, _ int) error {
		raw, err := h.call(ctx, big, c, "GET", "/api/v1/export/transactions?from="+from12+"&to="+to, nil, nil)
		if err != nil {
			return err
		}

		if rows := bytes.Count(raw, []byte("\n")) - 1; rows < h.cfg.bigRows*45/100 {
			return fmt.Errorf("export holds %d rows, want about %d", rows, h.cfg.bigRows/2)
		}

		return nil
	}))

	out = append(out, h.measure("api_list_small", n, cl, func(c, i int) error {
		w := h.ws[1+i%h.cfg.workspaces]
		_, err := h.call(ctx, w, c, "GET", "/api/v1/transactions", nil, nil)

		return err
	}))

	out = append(out, h.measure("sync_pull", n, cl, func(c, i int) error {
		_, err := h.call(ctx, big, c, "GET", fmt.Sprintf("/api/v1/sync/pull?limit=500&cursor=%d", (i*1000)%h.cfg.bigRows), nil, h.syncHeaders(c))
		return err
	}))

	out = append(out, h.measure("sync_push", n, cl, func(c, _ int) error {
		return h.push(ctx, big, c, 20)
	}))

	mcp, err := h.mcpScenarios(ctx)
	if err != nil {
		return nil, err
	}

	out = append(out, mcp...)

	out = append(out, h.mixed(ctx, web))

	return out, nil
}

func (h *harness) syncHeaders(c int) map[string]string {
	return map[string]string{"X-Sync-Protocol": "1", "X-Device-Id": h.devices[c]}
}

func (h *harness) registerDevices(ctx context.Context) error {
	big := h.big()

	for c := range h.cfg.clients {
		raw, err := h.call(ctx, big, c, "POST", "/api/v1/sync/devices", map[string]string{"name": fmt.Sprintf("load %d", c), "platform": "other"},
			map[string]string{"X-Sync-Protocol": "1"})
		if err != nil {
			return fmt.Errorf("register device: %w", err)
		}

		var d struct {
			DeviceID string `json:"device_id"`
		}

		if err := json.Unmarshal(raw, &d); err != nil || d.DeviceID == "" {
			return fmt.Errorf("register device: no device_id in %s", raw)
		}

		h.devices = append(h.devices, d.DeviceID)
	}

	return nil
}

// push sends count new transactions as one sync batch.
func (h *harness) push(ctx context.Context, w qakit.Workspace, c, count int) error {
	muts := make([]map[string]any, 0, count)

	for i := range count {
		muts = append(muts, map[string]any{
			"op_id": qakit.ULID(time.Now()), "entity": "transaction", "uid": qakit.ULID(time.Now()), "op": "upsert", "base_seq": 0,
			"fields": map[string]any{
				"account_uid": w.Accounts[i%3], "category_uid": w.Cats[2+i%10], "kind": "expense",
				"amount_cents": 1000 + i, "description": "pushed by the load test", "occurred_on": time.Now().UTC().Format("2006-01-02"),
			},
		})
	}

	raw, err := h.call(ctx, w, c, "POST", "/api/v1/sync/push", map[string]any{"mutations": muts}, h.syncHeaders(c))
	if err != nil {
		return err
	}

	var res struct {
		Results []struct {
			Status string `json:"status"`
			Error  struct {
				Code string `json:"code"`
			} `json:"error"`
		} `json:"results"`
	}

	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}

	for _, r := range res.Results {
		if r.Status == "rejected" {
			return fmt.Errorf("push rejected a mutation: %s", r.Error.Code)
		}
	}

	return nil
}

// mixed is every kind of work at once: reads, a report, a pull, a push and a
// create, with no error and no lock allowed.
func (h *harness) mixed(ctx context.Context, web *webClient) result {
	big := h.big()

	return h.measure("mixed_load", h.cfg.requests, h.cfg.clients, func(c, i int) error {
		switch i % 6 {
		case 0:
			return web.get(ctx, "/dashboard")
		case 1:
			_, err := h.call(ctx, big, c, "GET", "/api/v1/transactions?page=2", nil, nil)
			return err
		case 2:
			_, err := h.call(ctx, big, c, "GET", "/api/v1/reports/categories?period=last_3_months", nil, nil)
			return err
		case 3:
			_, err := h.call(ctx, big, c, "GET", "/api/v1/sync/pull?limit=200&cursor=0", nil, h.syncHeaders(c))
			return err
		case 4:
			return h.push(ctx, big, c, 5)
		default:
			_, err := h.call(ctx, big, c, "POST", "/api/v1/transactions", map[string]any{
				"uid": qakit.ULID(time.Now()), "account_uid": big.Accounts[0], "kind": "expense", "amount_cents": 1234, "description": "mixed", "date": time.Now().UTC().Format("2006-01-02"),
			}, nil)

			return err
		}
	})
}
