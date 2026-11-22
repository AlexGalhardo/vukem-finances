package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	vukem "github.com/AlexGalhardo/vukem"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
)

// mcpReferenceCommand writes the MCP reference from the registry
// (docs/reference/mcp.md, committed; TestMCPReferenceIsFresh fails when stale).
type mcpReferenceCommand struct{ out *string }

func (c *mcpReferenceCommand) Name() string { return "mcp:reference" }

func (c *mcpReferenceCommand) Description() string {
	return "Write the MCP reference from the operation registry (docs/reference/mcp.md)"
}

func (c *mcpReferenceCommand) Flags(fs *flag.FlagSet) {
	c.out = fs.String("out", "docs/reference/mcp.md", "the file to write")
}

// NoBoot: the reference describes code, it opens no database.
func (c *mcpReferenceCommand) NoBoot() bool { return true }

func (c *mcpReferenceCommand) Run(_ context.Context, app *vukem.App, _ []string) error {
	body, err := MCPReference()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(*c.out), 0o755); err != nil {
		return fmt.Errorf("mcp:reference: %w", err)
	}

	if err := os.WriteFile(*c.out, body, 0o644); err != nil {
		return fmt.Errorf("mcp:reference: %w", err)
	}

	if app != nil {
		fmt.Fprintf(app.Stdout(), "mcp:reference: wrote %s\n", *c.out)
	}

	return nil
}

// MCPReference is the content of docs/reference/mcp.md.
func MCPReference() ([]byte, error) { return appmcp.Reference() }
