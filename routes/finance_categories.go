package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeCategories(g *router.Group, h *ledger.Handlers) {
	g.Get("/categories", h.Categories).Name("categories.index")
	g.Post("/categories", h.StoreCategory).Name("categories.store")
}
