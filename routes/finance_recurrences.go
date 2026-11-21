package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeRecurrences(g *router.Group, h *ledger.Handlers) {
	g.Get("/recurrences", h.Recurrences).Name("recurrences.index")
	g.Post("/recurrences", h.StoreRecurrence).Name("recurrences.store")
	g.Patch("/recurrences/{recurrence}", h.UpdateRecurrence).Name("recurrences.update")
	g.Delete("/recurrences/{recurrence}", h.DestroyRecurrence).Name("recurrences.destroy")
	g.Post("/installments", h.StoreInstallments).Name("installments.store")
	g.Delete("/transactions/{transaction}/installments", h.DestroyInstallments).Name("installments.destroy")
}
