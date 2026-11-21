package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeBudgets(g *router.Group, h *ledger.Handlers) {
	g.Get("/budgets", h.Budgets).Name("budgets.index")
	g.Post("/budgets", h.StoreBudget).Name("budgets.store")
	g.Delete("/budgets/{budget}", h.DestroyBudget).Name("budgets.destroy")
}
