package routes

import (
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

func financeGoals(g *router.Group, h *ledger.Handlers) {
	g.Get("/goals", h.Goals).Name("goals.index")
	g.Post("/goals", h.StoreGoal).Name("goals.store")
	g.Patch("/goals/{goal}", h.UpdateGoal).Name("goals.update")
	g.Delete("/goals/{goal}", h.DestroyGoal).Name("goals.destroy")
	g.Post("/goals/{goal}/archive", h.ArchiveGoal).Name("goals.archive")
	g.Post("/goals/{goal}/unarchive", h.UnarchiveGoal).Name("goals.unarchive")
	g.Post("/goals/{goal}/contributions", h.StoreContribution).Name("goals.contributions.store")
	g.Delete("/goals/{goal}/contributions/{contribution}", h.DestroyContribution).Name("goals.contributions.destroy")
}
