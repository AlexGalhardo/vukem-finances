package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

const cliReferenceFile = "../docs/reference/cli.md"

// The CLI reference is committed and checked: a new, changed or removed
// operation fails here until `go run ./cmd/app cli:reference` writes the file.
func TestCLIReferenceIsFresh(t *testing.T) {
	t.Parallel()

	body, err := bootstrap.CLIReference()
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, cliReferenceFile, body, "go run ./cmd/app cli:reference")
}
