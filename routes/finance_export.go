package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

// financeExports registers the downloads. They share one limit per workspace.
// The CSV is open to every role that can read. The JSON needs manageData, checked
// before the limit so a viewer cannot spend the workspace's downloads.
func financeExports(g *router.Group, h *ledger.Handlers, exports, manageData router.Middleware) {
	g.Get("/transactions/export.csv", h.ExportTransactionsCSV).Name("transactions.export").Use(exports)
	g.Get("/workspace/export.json", h.ExportWorkspaceJSON).Name("workspace.export").Use(manageData, exports)
}
