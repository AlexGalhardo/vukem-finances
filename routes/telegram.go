package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Telegram registers the bot (docs/telegram.md). With no TELEGRAM_BOT_TOKEN it
// registers nothing: every bot route is 404 and no worker starts. In webhook
// mode it adds POST /telegram/webhook/{secret}, which the web middleware
// skips (the secret header is the credential); in polling mode it adds the
// polling worker to the server.
func Telegram(r *router.Router, app *vukem.App) error {
	ws, err := workspaces.New(app)
	if err != nil {
		return err
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return err
	}

	bot, err := telegram.New(app, ws, fin)
	if err != nil || bot == nil {
		return err
	}

	if bot.Cfg.Mode == config.TelegramWebhook {
		r.Handle("POST "+config.WebhookPath+"/{secret}", bot.WebhookHandler()).SecretParams("secret")

		return nil
	}

	if app.Kind == vukem.KindWeb {
		app.AddRunner("telegram-poller", bot)
	}

	return nil
}
