package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// plansExportCommand writes the plan limits the static site reads
// (site/generated/plans.json, committed). With -check it writes nothing and
// fails when the file differs from workspaces.Free and workspaces.Pro.
type plansExportCommand struct {
	out   *string
	check *bool
}

func (c *plansExportCommand) Name() string { return "plans:export" }

func (c *plansExportCommand) Description() string {
	return "Write the Free and Pro limits for the static site (site/generated/plans.json); -check only compares"
}

func (c *plansExportCommand) Flags(fs *flag.FlagSet) {
	c.out = fs.String("out", "site/generated/plans.json", "the file to write or compare")
	c.check = fs.Bool("check", false, "fail when the file is stale instead of writing it")
}

// NoBoot: the limits are constants, nothing opens a database.
func (c *plansExportCommand) NoBoot() bool { return true }

func (c *plansExportCommand) Run(_ context.Context, app *vukem.App, _ []string) error {
	body, err := PlansJSON()
	if err != nil {
		return err
	}

	if *c.check {
		have, err := os.ReadFile(*c.out)
		if err != nil {
			return fmt.Errorf("plans:export: %w (run go run ./cmd/app plans:export)", err)
		}

		// Git on Windows may check the file out with CRLF.
		if !bytes.Equal(bytes.ReplaceAll(have, []byte("\r\n"), []byte("\n")), body) {
			return errors.New("plans:export: " + *c.out + " is stale: run go run ./cmd/app plans:export")
		}

		return nil
	}

	if err := os.WriteFile(*c.out, body, 0o644); err != nil {
		return fmt.Errorf("plans:export: %w", err)
	}

	if app != nil {
		fmt.Fprintf(app.Stdout(), "plans:export: wrote %s\n", *c.out)
	}

	return nil
}

// PlansJSON is the content of site/generated/plans.json: the limits and
// features of the two plans and nothing else (no price).
func PlansJSON() ([]byte, error) {
	body, err := json.MarshalIndent(struct {
		Free workspaces.Limits `json:"free"`
		Pro  workspaces.Limits `json:"pro"`
	}{workspaces.Free, workspaces.Pro}, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(body, '\n'), nil
}
