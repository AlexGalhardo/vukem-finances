package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/openapi"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// These tests compare generated text with files committed in this repository.
// The text carries the application name (tool descriptions, schema names built
// from the module path), so the scaffold export leaves this file and the
// files it checks out: a project generated under another name regenerates
// them instead (docs/scaffold.md).

const (
	clientFile       = "../clients/typescript/api.ts"
	mcpReferenceFile = "../docs/reference/mcp.md"
	updateHow        = "UPDATE_OPENAPI=1 go test ./tests/... -run 'TestOpenAPISnapshot|TestTypeScriptClientIsFresh'"
)

func TestOpenAPISnapshot(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	body, err := api.RenderDocument(apiDocument(t, app))
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, openAPIFile, body, updateHow)
}

func TestTypeScriptClientIsFresh(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	ts, err := openapi.TypeScriptClientWith(apiDocument(t, app), openapi.ClientOptions{Args: "(see docs/api.md)"})
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, clientFile, []byte(ts), updateHow)
}

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
