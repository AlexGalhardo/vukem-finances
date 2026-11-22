package bootstrap

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

// cliCommands are the commands of the CLI over the operation registry (epic 31).
func cliCommands() []vukem.Command {
	tree, err := cli.DefaultTree()
	if err != nil {
		panic("bootstrap: the CLI tree: " + err.Error())
	}

	return cli.Commands(&cli.CLI{Tree: tree, Env: cli.OSEnv(openLocalApp)})
}

// openLocalApp builds and boots the application of this installation for the
// local mode of the CLI.
func openLocalApp(ctx context.Context) (*vukem.App, func() error, error) {
	app, err := New(vukem.WithKind(vukem.KindConsole))
	if err != nil {
		return nil, nil, err
	}

	if err := app.Boot(ctx); err != nil {
		return nil, nil, err
	}

	return app, func() error { return app.Shutdown(context.WithoutCancel(ctx)) }, nil
}
