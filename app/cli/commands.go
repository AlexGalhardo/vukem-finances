package cli

import (
	"context"
	"flag"
	"os"

	vukem "github.com/AlexGalhardo/vukem"
)

// OSEnv is the environment of the real process.
func OSEnv(open OpenLocal) Env {
	return Env{
		Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Getenv: os.LookupEnv,
		StdinTTY: isTerminal(os.Stdin), StdoutTTY: isTerminal(os.Stdout),
		OpenLocal: LocalFrom(open),
	}
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()

	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

// Commands makes one Vukem command per resource of the tree (`transactions`,
// `accounts`, ...). They run before the app is built (NoBoot, no APP_KEY), so a
// machine that only talks to a server needs no database; local mode builds the
// app itself.
func Commands(c *CLI) []vukem.Command {
	var out []vukem.Command

	for _, r := range c.Tree.Resources() {
		out = append(out, &resourceCommand{cli: c, resource: r})
	}

	return out
}

type resourceCommand struct {
	cli      *CLI
	resource string
}

func (r *resourceCommand) Name() string { return r.resource }

func (r *resourceCommand) Description() string {
	return "The " + r.resource + " operations: `" + r.resource + " <verb>` (run it with no verb to list them)"
}

func (r *resourceCommand) Flags(*flag.FlagSet) {}

// NoBoot: remote mode needs no application, and local mode opens its own.
func (r *resourceCommand) NoBoot() bool { return true }

// RawArgs: the verb and its flags reach Run as typed.
func (r *resourceCommand) RawArgs() {}

func (r *resourceCommand) Run(ctx context.Context, _ *vukem.App, args []string) error {
	return exitError(r.cli.Run(ctx, r.resource, args))
}

// exitError wraps err so Vukem exits with the code of the CLI.
func exitError(err error) error {
	if err == nil {
		return nil
	}

	return &vukem.ExitError{Code: ExitOf(err), Err: err}
}
