package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeAccounts(g *router.Group, h *ledger.Handlers) {
	g.Get("/accounts", h.Accounts).Name("accounts.index")
	g.Post("/accounts", h.StoreAccount).Name("accounts.store")
	g.Delete("/accounts/{account}", h.DestroyAccount).Name("accounts.destroy")
}
