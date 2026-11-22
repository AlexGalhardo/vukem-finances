package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeTransactions(g *router.Group, h *ledger.Handlers) {
	g.Get("/transactions", h.Transactions).Name("transactions.index")
	g.Get("/transactions/options", h.TransactionOptions).Name("transactions.options")
	g.Post("/transactions", h.StoreTransaction).Name("transactions.store")
	g.Patch("/transactions/{transaction}", h.UpdateTransaction).Name("transactions.update")
	g.Delete("/transactions/{transaction}", h.DestroyTransaction).Name("transactions.destroy")
	g.Post("/transactions/{transaction}/tags/{tag}", h.AttachTag).Name("transactions.tags.attach")
	g.Delete("/transactions/{transaction}/tags/{tag}", h.DetachTag).Name("transactions.tags.detach")
	g.Post("/transfers", h.StoreTransfer).Name("transfers.store")
	g.Patch("/transfers/{transaction}", h.UpdateTransfer).Name("transfers.update")
	g.Delete("/transfers/{transaction}", h.DestroyTransfer).Name("transfers.destroy")
}
