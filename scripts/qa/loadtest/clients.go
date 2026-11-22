package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem-finances/scripts/qa/qakit"
)

// webClient is a signed-in browser session: the dashboard is a web page, not
// an API route.
type webClient struct {
	h    *harness
	http *http.Client
}

// webLogin signs in as one user of the big workspace: the owner for -1, or
// member c. The per-user web rate limit (120 reads a minute) then applies to
// each client alone, which is how 20 people would meet it.
func (h *harness) webLogin(ctx context.Context, member int) (*webClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &http.Client{Jar: jar, Timeout: 2 * time.Minute, Transport: h.http.Transport}
	email := h.big().Email
	if member >= 0 {
		email = h.big().Members[member]
	}

	form := url.Values{"email": {email}, "password": {qakit.SeedPass}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.base+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	w := &webClient{h: h, http: c}
	if err := w.get(ctx, "/dashboard"); err != nil {
		return nil, fmt.Errorf("web login did not reach the dashboard: %w", err)
	}

	return w, nil
}

func (w *webClient) get(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.h.base+path, nil)
	if err != nil {
		return err
	}

	res, err := w.http.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK || strings.HasSuffix(res.Request.URL.Path, "/login") {
		return fmt.Errorf("GET %s: status %d at %s: %.120s", path, res.StatusCode, res.Request.URL.Path, strings.TrimSpace(string(raw)))
	}

	return nil
}

// mcpClient is one MCP stdio server process of the personal edition.
type mcpClient struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
	id  int
}

func (h *harness) startMCP(ctx context.Context) (*mcpClient, error) {
	cmd := exec.CommandContext(ctx, h.bin, "mcp:serve", "--stdio")
	cmd.Env = h.env
	cmd.Stderr = io.Discard

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	m := &mcpClient{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<20)}

	_, err = m.rpc("initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "loadtest", "version": "1"},
	})
	if err != nil {
		m.stop()
		return nil, err
	}

	return m, nil
}

func (m *mcpClient) rpc(method string, params any) (json.RawMessage, error) {
	m.id++

	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": m.id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}

	if _, err := m.in.Write(append(raw, '\n')); err != nil {
		return nil, err
	}

	line, err := m.out.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var res struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(line, &res); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}

	if res.Error != nil {
		return nil, fmt.Errorf("%s: %s", method, res.Error.Message)
	}

	var tool struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if json.Unmarshal(res.Result, &tool) == nil && tool.IsError && len(tool.Content) > 0 {
		return nil, fmt.Errorf("%s: %.160s", method, tool.Content[0].Text)
	}

	return res.Result, nil
}

func (m *mcpClient) stop() {
	_ = m.in.Close()
	_ = m.cmd.Process.Kill()
	_ = m.cmd.Wait()
}

// mcpScenarios times the list tools of the stdio server, one process per
// client, the way twenty assistants would hold it.
func (h *harness) mcpScenarios(ctx context.Context) ([]result, error) {
	clients := make([]*mcpClient, 0, h.cfg.clients)

	defer func() {
		for _, c := range clients {
			c.stop()
		}
	}()

	for range h.cfg.clients {
		c, err := h.startMCP(ctx)
		if err != nil {
			return nil, fmt.Errorf("start mcp: %w", err)
		}

		clients = append(clients, c)
	}

	// The clients share the personal owner's workspace: the big one.
	n, cl := h.cfg.requests, h.cfg.clients
	tools := []struct{ name, tool string }{{"mcp_tools_list", ""}, {"mcp_transactions_list", "transactions_list"}}

	var out []result

	for _, t := range tools {
		out = append(out, h.measure(t.name, n, cl, func(c, _ int) error {
			// A client is one process and one request at a time: the pool gives each worker its own.
			if t.tool == "" {
				_, err := clients[c].rpc("tools/list", map[string]any{})
				return err
			}

			_, err := clients[c].rpc("tools/call", map[string]any{"name": t.tool, "arguments": map[string]any{}})

			return err
		}))
	}

	return out, nil
}
