package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"

	vukem "github.com/AlexGalhardo/vukem"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
)

// mcpServeCommand serves MCP on the standard streams (epic 32). Stdout carries
// protocol messages and nothing else: logs go to stderr.
func mcpServeCommand() vukem.Command {
	var stdio *bool

	return vukem.CommandFunc{
		CommandName: "mcp:serve",
		Desc:        "Serve the finances to an AI client over MCP (--stdio, personal edition)",
		DefineFlags: func(fs *flag.FlagSet) { stdio = fs.Bool("stdio", false, "serve on the standard input and output") },
		Fn: func(ctx context.Context, app *vukem.App, _ []string) error {
			if !*stdio {
				return fmt.Errorf("%w: mcp:serve needs --stdio", vukem.ErrUsage)
			}

			srv, err := appmcp.NewStdio(app)
			if err != nil {
				return fmt.Errorf("mcp:serve: %w", err)
			}

			defer srv.Close()

			if err := srv.ServeStdio(ctx, app.Stdin(), app.Stdout()); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("mcp:serve: %w", err)
			}

			return nil
		},
	}
}
