package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

// financeAI registers the AI settings routes inside the workspace group. The
// page is readable by every member; each write checks ai.manage (owner only)
// in its handler.
func financeAI(g *router.Group, app *vukem.App, h *ledger.Handlers) error {
	svc, err := ai.New(app)
	if err != nil {
		return err
	}

	ah := ledger.NewAI(h, svc)

	g.Get("/ai", ah.Show).Name("ai.show")
	g.Put("/ai/settings", ah.Update).Name("ai.update")
	g.Post("/ai/key", ah.SaveKey).Name("ai.key.store")
	g.Delete("/ai/key/{credential}", ah.DeleteKey).Name("ai.key.destroy")
	g.Post("/ai/check", ah.Check).Name("ai.check")

	return nil
}
