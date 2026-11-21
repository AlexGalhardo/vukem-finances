package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeDashboard(g *router.Group, h *ledger.Handlers) {
	g.Get("/dashboard", h.Dashboard).Name("dashboard")
}
