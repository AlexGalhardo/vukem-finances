package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

// Notifications belong to the signed-in person: no path names the id of
// another person, so one person cannot reach the rows of another.
func financeNotifications(g *router.Group, h *ledger.Handlers) {
	g.Get("/notifications", h.Notifications).Name("notifications.index")
	g.Post("/notifications/read", h.ReadAllNotifications).Name("notifications.read_all")
	g.Put("/notifications/preferences", h.SavePreferences).Name("notifications.preferences")
	g.Post("/notifications/{notification}/read", h.ReadNotification).Name("notifications.read")
}
