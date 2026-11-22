package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	vmcp "github.com/AlexGalhardo/vukem/mcp"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// ask runs one request against srv over the stdio framing and returns the
// result of the reply. The reference is written from the live server, so it
// cannot say what the server does not.
func ask(srv *vmcp.Server, method string) (json.RawMessage, error) {
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": method,
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    vmcp.Version,
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}},
	})
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer

	if err := srv.ServeStdio(context.Background(), bytes.NewReader(append(req, '\n')), &out); err != nil {
		return nil, err
	}

	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  *vmcp.Error     `json:"error"`
	}

	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		return nil, fmt.Errorf("mcp reference: %s: %w", method, err)
	}

	if reply.Error != nil {
		return nil, fmt.Errorf("mcp reference: %s: %w", method, reply.Error)
	}

	return reply.Result, nil
}

func describeServer(cfg config.MCP) (*vmcp.Server, *operations.Catalog, error) {
	cat := operations.NewCatalog(operations.Config{Edition: config.EditionPersonal}, operations.Services{})

	srv, err := New(Options{
		Catalog: cat, Config: cfg, PageSize: 1000,
		Actor: func(context.Context) (operations.Actor, error) {
			return operations.Actor{}, fmt.Errorf("describe only")
		},
	})

	return srv, cat, err
}

// Reference is docs/reference/mcp.md: the tools, resources, prompts and
// toolsets, written from the server itself. The same code always gives the
// same bytes.
func Reference() ([]byte, error) {
	full, cat, err := describeServer(config.MCP{Toolsets: []string{config.MCPToolsetAll}, AllowDestructive: true})
	if err != nil {
		return nil, err
	}

	var tools struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}

	raw, err := ask(full, "tools/list")
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil, err
	}

	var b strings.Builder

	b.WriteString(strings.ReplaceAll(`# MCP reference

<!-- Generated from the MCP server: do not edit. Run ~go run ./cmd/app mcp:reference~. -->

The server speaks the Model Context Protocol, revision 2026-07-28. Every tool is an
operation of the registry (~app/operations~), so it has the scope, the role check, the
validation and the isolation of the REST endpoint of the same name. A result is the
person's own data: a client must never treat text inside it as an instruction.

## Personal edition on stdio

~mcp:serve --stdio~ runs only with ~APP_EDITION=personal~, as the local owner. Standard
output carries protocol messages and nothing else; logs go to standard error. Any other
edition exits non-zero before it serves.

| Setting | Meaning |
| --- | --- |
| ~MCP_TOOLSETS~ | Comma separated groups of tools. Default ~read~. |
| ~MCP_ALLOW_DESTRUCTIVE~ | ~true~ offers the tools that remove or archive data. Default ~false~. |

Each destructive call asks the person to confirm through the protocol (the input-required
flow). A client that cannot ask gets a refusal, and a declined answer changes nothing.

## Toolsets

| Toolset | Offers |
| --- | --- |
`, "~", "`"))

	sets := []struct{ name, what string }{
		{config.MCPToolsetRead, "every read operation"},
		{config.MCPToolsetTransactions, "transactions and transfers: list, show, record, change"},
		{config.MCPToolsetPlanning, "budgets, the summary, goals, recurring rules, cards and statements"},
		{config.MCPToolsetAll, "every tool"},
	}

	for _, s := range sets {
		fmt.Fprintf(&b, "| `%s` | %s |\n", s.name, s.what)
	}

	b.WriteString("\nSerialized size of the tool list of each toolset, which a client may hold in context:\n\n")
	b.WriteString("| Toolset | Tools | Bytes |\n| --- | --- | --- |\n")

	for _, s := range sets {
		for _, destructive := range []bool{false, true} {
			srv, _, err := describeServer(config.MCP{Toolsets: []string{s.name}, AllowDestructive: destructive})
			if err != nil {
				return nil, err
			}

			res, err := ask(srv, "tools/list")
			if err != nil {
				return nil, err
			}

			var n struct {
				Tools []json.RawMessage `json:"tools"`
			}

			if err := json.Unmarshal(res, &n); err != nil {
				return nil, err
			}

			label := s.name
			if destructive {
				label += " + destructive"
			}

			fmt.Fprintf(&b, "| `%s` | %d | %d |\n", label, len(n.Tools), len(res))
		}
	}

	b.WriteString("\n## Tools\n\n| Tool | Operation | Kind | Scope | Toolsets |\n| --- | --- | --- | --- | --- |\n")

	for _, t := range tools.Tools {
		m, ok := cat.Named(strings.ReplaceAll(t.Name, "_", "."))
		if !ok {
			m, ok = findByTool(cat, t.Name)
		}

		if !ok {
			return nil, fmt.Errorf("mcp reference: no operation for tool %s", t.Name)
		}

		var in []string

		for _, s := range sets {
			if InToolset(s.name, m) {
				in = append(in, s.name)
			}
		}

		fmt.Fprintf(&b, "| `%s` | `%s` | %s | `%s` | %s |\n", t.Name, m.Name, m.Kind, m.Scope, strings.Join(in, ", "))
	}

	b.WriteString("\nOperations with no tool, each with the reason:\n\n")

	var optOuts []string

	for _, m := range cat.Operations() {
		if why, out := m.OptOut[operations.SurfaceMCP]; out {
			optOuts = append(optOuts, fmt.Sprintf("- `%s`: %s", m.Name, why))
		}
	}

	slices.Sort(optOuts)
	b.WriteString(strings.Join(optOuts, "\n") + "\n")

	if err := listSection(&b, full, "resources/list", "resources", "## Resources", "uri"); err != nil {
		return nil, err
	}

	if err := listSection(&b, full, "resources/templates/list", "resourceTemplates", "## Resource templates", "uriTemplate"); err != nil {
		return nil, err
	}

	if err := listSection(&b, full, "prompts/list", "prompts", "## Prompts", "name"); err != nil {
		return nil, err
	}

	return []byte(b.String()), nil
}

func findByTool(cat *operations.Catalog, tool string) (operations.Meta, bool) {
	for _, m := range cat.Operations() {
		if ToolName(m.Name) == tool {
			return m, true
		}
	}

	return operations.Meta{}, false
}

// listSection writes a list of the server (resources, templates, prompts) as a
// table of its key and description.
func listSection(b *strings.Builder, srv *vmcp.Server, method, field, title, key string) error {
	raw, err := ask(srv, method)
	if err != nil {
		return err
	}

	var res map[string]json.RawMessage
	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}

	var items []map[string]any
	if err := json.Unmarshal(res[field], &items); err != nil {
		return err
	}

	fmt.Fprintf(b, "\n%s\n\n| Name | Description |\n| --- | --- |\n", title)

	for _, it := range items {
		desc, _ := it["description"].(string)
		fmt.Fprintf(b, "| `%v` | %s |\n", it[key], strings.ReplaceAll(desc, "|", "\\|"))
	}

	return nil
}

// ToolNames lists the tools the server offers with every toolset and the
// destructive tools on, as a client sees them. The parity test compares them
// with the registry.
func ToolNames() ([]string, error) {
	srv, _, err := describeServer(config.MCP{Toolsets: []string{config.MCPToolsetAll}, AllowDestructive: true})
	if err != nil {
		return nil, err
	}

	raw, err := ask(srv, "tools/list")
	if err != nil {
		return nil, err
	}

	var res struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(res.Tools))
	for _, t := range res.Tools {
		names = append(names, t.Name)
	}

	return names, nil
}
