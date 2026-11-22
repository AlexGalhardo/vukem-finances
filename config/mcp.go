package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// The toolsets of the MCP server (MCP_TOOLSETS), each a group of tools an agent
// is offered. Read is the default, so write access is always a choice.
const (
	MCPToolsetRead         = "read"
	MCPToolsetTransactions = "transactions"
	MCPToolsetPlanning     = "planning"
	MCPToolsetAll          = "all"
)

// The limits of the remote MCP endpoint and of its authorization server
// (docs/mcp.md). A grant is counted apart for reads and for writes; the address
// is counted for every request, and the OAuth endpoints per address.
const (
	MCPReadLimit  = 120
	MCPWriteLimit = 60
	MCPIPLimit    = 300
	// MCPOAuthLimit is the requests of one address per window to each of the
	// authorize, token and revoke endpoints.
	MCPOAuthLimit = 30
	MCPWindow     = time.Minute
)

// MCPToolsets lists the toolsets.
func MCPToolsets() []string {
	return []string{MCPToolsetRead, MCPToolsetTransactions, MCPToolsetPlanning, MCPToolsetAll}
}

// MCP configures the MCP server (docs/mcp.md).
type MCP struct {
	// Toolsets are the groups of tools offered (MCP_TOOLSETS, comma separated).
	Toolsets []string
	// AllowDestructive offers the tools that remove or archive data, each of
	// which still asks the person for a confirmation (MCP_ALLOW_DESTRUCTIVE).
	AllowDestructive bool
	// OAuthClients are the AI clients registered ahead of time
	// (MCP_OAUTH_CLIENTS, a JSON list); any other client is identified by a
	// Client ID Metadata Document.
	OAuthClients []OAuthClient
}

// OAuthClient is a pre-registered public client of the remote endpoint: it has
// no secret and relies on PKCE. Redirect URIs are matched exactly.
type OAuthClient struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
}

// LoadMCP reads the settings of the MCP server from src. An unknown toolset or
// a value that is not true or false is an error, so a typo never widens what an
// agent can do.
func LoadMCP(src vconfig.Lookuper) (MCP, error) {
	cfg := MCP{Toolsets: []string{MCPToolsetRead}}

	if v, _ := src.Lookup("MCP_TOOLSETS"); strings.TrimSpace(v) != "" {
		cfg.Toolsets = nil

		for part := range strings.SplitSeq(v, ",") {
			name := strings.ToLower(strings.TrimSpace(part))

			if !slices.Contains(MCPToolsets(), name) {
				return MCP{}, fmt.Errorf("MCP_TOOLSETS has %q: use %s", name, strings.Join(MCPToolsets(), ", "))
			}

			if !slices.Contains(cfg.Toolsets, name) {
				cfg.Toolsets = append(cfg.Toolsets, name)
			}
		}
	}

	if v, _ := src.Lookup("MCP_ALLOW_DESTRUCTIVE"); strings.TrimSpace(v) != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true":
			cfg.AllowDestructive = true
		case "false":
		default:
			return MCP{}, fmt.Errorf("MCP_ALLOW_DESTRUCTIVE=%q is invalid: use true or false", v)
		}
	}

	if v, _ := src.Lookup("MCP_OAUTH_CLIENTS"); strings.TrimSpace(v) != "" {
		dec := json.NewDecoder(strings.NewReader(v))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&cfg.OAuthClients); err != nil {
			return MCP{}, fmt.Errorf("MCP_OAUTH_CLIENTS is not a JSON list of clients: %w", err)
		}

		for _, c := range cfg.OAuthClients {
			if c.ID == "" || c.Name == "" || len(c.RedirectURIs) == 0 {
				return MCP{}, fmt.Errorf("MCP_OAUTH_CLIENTS: every client needs id, name and redirect_uris")
			}
		}
	}

	return cfg, nil
}
