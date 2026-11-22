package cli

import (
	"context"
	"flag"
	"os"

	vukem "github.com/AlexGalhardo/vukem"
)

// OSEnv is the environment of the real process.
func OSEnv(open OpenLocal) Env {
	var store CredentialStore
	if s, err := DefaultStore(); err == nil {
		store = s
	}

	return Env{
		Store: store, HTTP: NewHTTPClient(),
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

	return append(out, &authCommand{cli: c}, &completionCommand{cli: c})
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

type authCommand struct{ cli *CLI }

func (a *authCommand) Name() string { return "auth" }

func (a *authCommand) Description() string {
	return "Log in to a server (`auth login --server URL`), log out, or show the login"
}

func (a *authCommand) Flags(*flag.FlagSet) {}
func (a *authCommand) NoBoot() bool        { return true }
func (a *authCommand) RawArgs()            {}

func (a *authCommand) Run(ctx context.Context, _ *vukem.App, args []string) error {
	return exitError(a.cli.Run(ctx, "auth", args))
}

type completionCommand struct{ cli *CLI }

func (c *completionCommand) Name() string { return "completion" }

func (c *completionCommand) Description() string {
	return "Print the shell completion script: `completion bash|zsh|fish|powershell [program]`"
}

func (c *completionCommand) Flags(*flag.FlagSet) {}
func (c *completionCommand) NoBoot() bool        { return true }
func (c *completionCommand) RawArgs()            {}

func (c *completionCommand) Run(ctx context.Context, _ *vukem.App, args []string) error {
	return exitError(c.cli.Run(ctx, "completion", args))
}
