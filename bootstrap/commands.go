package bootstrap

import vukem "github.com/AlexGalhardo/vukem"

// Commands lists the application commands; vukem make:command appends here.
func Commands() []vukem.Command {
	return []vukem.Command{aboutCommand(), resetPasswordCommand(), telegramSetWebhookCommand(), telegramInfoCommand(), aiEvalCommand()}
}

// EarlyCommands run before the app is built, so they need no APP_KEY and open
// no database: cmd/app registers them with vukem.WithCommands.
func EarlyCommands() []vukem.Command {
	return append([]vukem.Command{newInventoryCommand()}, cliCommands()...)
}
