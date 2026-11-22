package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

// cliReferenceCommand writes the CLI reference from the registry
// (docs/reference/cli.md, committed; TestCLIReferenceIsFresh fails when stale).
type cliReferenceCommand struct{ out *string }

func (c *cliReferenceCommand) Name() string { return "cli:reference" }

func (c *cliReferenceCommand) Description() string {
	return "Write the CLI reference from the operation registry (docs/reference/cli.md)"
}

func (c *cliReferenceCommand) Flags(fs *flag.FlagSet) {
	c.out = fs.String("out", "docs/reference/cli.md", "the file to write")
}

// NoBoot: the reference describes code, it opens no database.
func (c *cliReferenceCommand) NoBoot() bool { return true }

func (c *cliReferenceCommand) Run(_ context.Context, app *vukem.App, _ []string) error {
	body, err := CLIReference()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(*c.out), 0o755); err != nil {
		return fmt.Errorf("cli:reference: %w", err)
	}

	if err := os.WriteFile(*c.out, body, 0o644); err != nil {
		return fmt.Errorf("cli:reference: %w", err)
	}

	if app != nil {
		fmt.Fprintf(app.Stdout(), "cli:reference: wrote %s\n", *c.out)
	}

	return nil
}

// CLIReference is the content of docs/reference/cli.md.
func CLIReference() ([]byte, error) {
	tree, err := cli.DefaultTree()
	if err != nil {
		return nil, err
	}

	return cli.Reference(tree), nil
}
