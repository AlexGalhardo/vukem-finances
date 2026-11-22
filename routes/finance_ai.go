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
	svc, err := ai.Shared(app)
	if err != nil {
		return err
	}

	ah := ledger.NewAI(h, svc)

	g.Get("/ai", ah.Show).Name("ai.show")
	g.Put("/ai/settings", ah.Update).Name("ai.update")
	g.Post("/ai/key", ah.SaveKey).Name("ai.key.store")
	g.Delete("/ai/key/{credential}", ah.DeleteKey).Name("ai.key.destroy")
	g.Post("/ai/check", ah.Check).Name("ai.check")

	// The features. Each handler checks its own ability: recording for what
	// changes transactions, reading for the assistant (it only reads).
	fh := ledger.NewAIFeatures(h, svc, app.Log, app.Now)

	g.Get("/suggestions", fh.Suggestions).Name("suggestions.index")
	g.Post("/suggestions/run", fh.RunSuggestions).Name("suggestions.run")
	g.Post("/suggestions/accept-all", fh.AcceptAllSuggestions).Name("suggestions.accept_all")
	g.Post("/suggestions/{suggestion}/accept", fh.AcceptSuggestion).Name("suggestions.accept")
	g.Post("/suggestions/{suggestion}/reject", fh.RejectSuggestion).Name("suggestions.reject")
	g.Post("/suggestions/{suggestion}/undo", fh.UndoSuggestion).Name("suggestions.undo")
	g.Post("/transactions/draft", fh.DraftEntry).Name("transactions.draft")
	g.Get("/assistant/conversations", fh.AssistantStatus).Name("assistant.index")
	g.Post("/assistant/ask", fh.AssistantAsk).Name("assistant.ask")
	g.Get("/assistant/conversations/{conversation}", fh.AssistantMessages).Name("assistant.show")
	g.Delete("/assistant/conversations/{conversation}", fh.AssistantDelete).Name("assistant.destroy")
	g.Delete("/assistant/conversations", fh.AssistantDeleteAll).Name("assistant.destroy_all")

	return nil
}
