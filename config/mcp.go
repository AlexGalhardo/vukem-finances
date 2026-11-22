package config

import (
	"fmt"
	"slices"
	"strings"

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

	return cfg, nil
}
