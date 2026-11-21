package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeWorkspace(g *router.Group, h *ledger.Handlers) {
	g.Get("/workspace", h.Workspace).Name("workspace.show")
	g.Put("/workspace", h.UpdateWorkspace).Name("workspace.update")
}
