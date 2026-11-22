package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// telegramBot builds the bot the commands talk through, refusing when it is off.
func telegramBot(app *vukem.App) (*telegram.Bot, error) {
	ws, err := workspaces.New(app)
	if err != nil {
		return nil, err
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return nil, err
	}

	bot, err := telegram.New(app, ws, fin)
	if err != nil {
		return nil, err
	}

	if bot == nil {
		return nil, errors.New("the Telegram bot is off: set TELEGRAM_BOT_TOKEN")
	}

	return bot, nil
}

// telegramSetWebhookCommand registers the webhook of the bot at Telegram: the
// URL is TELEGRAM_WEBHOOK_URL plus the secret path, and the secret header is
// TELEGRAM_WEBHOOK_SECRET. It prints the host only: the path is a secret.
func telegramSetWebhookCommand() vukem.Command {
	return vukem.CommandFunc{
		CommandName: "telegram:set-webhook",
		Desc:        "Register the Telegram webhook (webhook mode)",
		Fn: func(ctx context.Context, app *vukem.App, _ []string) error {
			bot, err := telegramBot(app)
			if err != nil {
				return err
			}

			if bot.Cfg.Mode != config.TelegramWebhook {
				return errors.New("TELEGRAM_MODE is not webhook: polling needs no webhook")
			}

			if bot.Cfg.WebhookURL == "" {
				return errors.New("set TELEGRAM_WEBHOOK_URL to the public https address of the app")
			}

			hook := bot.Cfg.WebhookURL + config.WebhookPath + "/" + bot.Cfg.WebhookPathSecret()
			if err := bot.Client.SetWebhook(ctx, hook, bot.Cfg.WebhookSecret.Reveal()); err != nil {
				return fmt.Errorf("telegram:set-webhook: %w", err)
			}

			host := bot.Cfg.WebhookURL
			if u, perr := url.Parse(host); perr == nil {
				host = u.Host
			}

			fmt.Fprintf(app.Stdout(), "telegram:set-webhook: registered the webhook at %s (the path stays secret)\n", host)

			return nil
		},
	}
}

// telegramInfoCommand checks the token with getMe and shows the settings. The
// token and the secrets are never printed.
func telegramInfoCommand() vukem.Command {
	return vukem.CommandFunc{
		CommandName: "telegram:info",
		Desc:        "Check the Telegram bot token and show its settings",
		Fn: func(ctx context.Context, app *vukem.App, _ []string) error {
			bot, err := telegramBot(app)
			if err != nil {
				return err
			}

			me, err := bot.Client.GetMe(ctx)
			if err != nil {
				return fmt.Errorf("telegram:info: %w", err)
			}

			out := app.Stdout()
			fmt.Fprintf(out, "Bot:        @%s\n", me.Username)
			fmt.Fprintf(out, "Mode:       %s\n", bot.Cfg.Mode)
			fmt.Fprintf(out, "Token:      %s\n", bot.Cfg.Token.String())
			fmt.Fprintf(out, "Webhook secret set: %t\n", !bot.Cfg.WebhookSecret.IsZero())

			if bot.Cfg.Username != me.Username {
				fmt.Fprintf(out, "Warning: TELEGRAM_BOT_USERNAME is %q but the token belongs to @%s\n", bot.Cfg.Username, me.Username)
			}

			return nil
		},
	}
}
