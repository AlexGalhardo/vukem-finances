package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeImports(g *router.Group, h *ledger.Handlers, imports router.Middleware) {
	g.Get("/import", h.Import).Name("import.index")
	g.Post("/import", h.UploadImport).Name("import.upload").Use(imports)
	g.Post("/import/{import}", h.ConfirmImport).Name("import.confirm").Use(imports)
}
