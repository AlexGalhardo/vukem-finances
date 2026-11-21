package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeCategories(g *router.Group, h *ledger.Handlers) {
	g.Get("/categories", h.Categories).Name("categories.index")
	g.Post("/categories", h.StoreCategory).Name("categories.store")
	g.Patch("/categories/{category}", h.UpdateCategory).Name("categories.update")
	g.Post("/categories/{category}/archive", h.ArchiveCategory).Name("categories.archive")
	g.Post("/categories/{category}/unarchive", h.UnarchiveCategory).Name("categories.unarchive")
	g.Delete("/categories/{category}", h.DestroyCategory).Name("categories.destroy")
	g.Post("/tags", h.StoreTag).Name("tags.store")
	g.Patch("/tags/{tag}", h.UpdateTag).Name("tags.update")
	g.Delete("/tags/{tag}", h.DestroyTag).Name("tags.destroy")
}
