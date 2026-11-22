package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"os"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// inventoryCommand writes the inventory of the operation registry: every
// operation with its scope, kind, surfaces, input and output, sorted so the
// same registry always writes the same bytes (resources/operations.json,
// committed; TestOperationInventoryIsFresh fails when it is stale).
type inventoryCommand struct{ out *string }

func newInventoryCommand() *inventoryCommand { return &inventoryCommand{} }

func (c *inventoryCommand) Name() string { return "operations:inventory" }

func (c *inventoryCommand) Description() string {
	return "Write the inventory of the operation registry (resources/operations.json)"
}

func (c *inventoryCommand) Flags(fs *flag.FlagSet) {
	c.out = fs.String("out", "resources/operations.json", "the file to write")
}

// NoBoot: the inventory describes code, it opens no database.
func (c *inventoryCommand) NoBoot() bool { return true }

func (c *inventoryCommand) Run(_ context.Context, app *vukem.App, _ []string) error {
	body, err := InventoryJSON()
	if err != nil {
		return err
	}

	if err := os.WriteFile(*c.out, body, 0o644); err != nil {
		return fmt.Errorf("operations:inventory: %w", err)
	}

	if app != nil {
		fmt.Fprintf(app.Stdout(), "operations:inventory: wrote %s\n", *c.out)
	}

	return nil
}

// InventoryJSON is the content of resources/operations.json.
func InventoryJSON() ([]byte, error) {
	rest := func(name string) (operations.Route, bool) {
		o, ok := api.OperationNamed(name)

		return operations.Route{Method: o.Method, Path: api.BasePath + o.Path}, ok
	}

	return operations.BuildInventory(api.Describe().Registry, rest).JSON()
}
