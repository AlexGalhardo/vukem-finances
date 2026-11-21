package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeTransactions(g *router.Group, h *ledger.Handlers) {
	g.Get("/transactions", h.Transactions).Name("transactions.index")
	g.Post("/transactions", h.StoreTransaction).Name("transactions.store")
	g.Patch("/transactions/{transaction}", h.UpdateTransaction).Name("transactions.update")
	g.Delete("/transactions/{transaction}", h.DestroyTransaction).Name("transactions.destroy")
}
