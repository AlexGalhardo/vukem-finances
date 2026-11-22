package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/controllers/account"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
)

// financeTelegram registers the Telegram settings page inside the workspace
// group, only while the bot is on: with no TELEGRAM_BOT_TOKEN the page and its
// actions are 404.
func financeTelegram(g *router.Group, bot *telegram.Bot, al *alerts.Service) {
	if bot == nil {
		return
	}

	h := account.NewTelegram(bot, al)

	g.Get("/settings/telegram", h.Show).Name("telegram.show")
	// The answer of Connect is rendered at its own URL (the code is never redirected through the session), so a
	// reload of that page lands here and goes back to the page.
	g.Get("/settings/telegram/connect", h.Show).Name("telegram.connect.show")
	g.Post("/settings/telegram/connect", h.Connect).Name("telegram.connect")
	g.Delete("/settings/telegram/links/{link}", h.Revoke).Name("telegram.revoke")
	g.Put("/settings/telegram/alerts", h.SetAlerts).Name("telegram.alerts")
}
