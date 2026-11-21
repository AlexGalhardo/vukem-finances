package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeCards(g *router.Group, h *ledger.Handlers) {
	g.Get("/cards", h.Cards).Name("cards.index")
	g.Post("/cards", h.StoreCard).Name("cards.store")
	g.Get("/cards/{card}", h.Card).Name("cards.show")
	g.Patch("/cards/{card}", h.UpdateCard).Name("cards.update")
	g.Post("/cards/{card}/archive", h.ArchiveCard).Name("cards.archive")
	g.Post("/cards/{card}/unarchive", h.UnarchiveCard).Name("cards.unarchive")
	g.Delete("/cards/{card}", h.DestroyCard).Name("cards.destroy")
	g.Post("/cards/{card}/statements/{ref}/pay", h.PayStatement).Name("cards.pay")
}
