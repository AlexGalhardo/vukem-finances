package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

const mcpReferenceFile = "../docs/reference/mcp.md"

// The MCP reference is committed and checked: a new, changed or removed tool,
// resource or prompt fails here until `go run ./cmd/app mcp:reference` writes
// the file.
func TestMCPReferenceIsFresh(t *testing.T) {
	t.Parallel()

	body, err := bootstrap.MCPReference()
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, mcpReferenceFile, body, "go run ./cmd/app mcp:reference")
}
