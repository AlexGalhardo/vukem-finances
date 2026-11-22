package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeReports(g *router.Group, h *ledger.Handlers) {
	g.Get("/reports", h.Reports).Name("reports.index")
}
