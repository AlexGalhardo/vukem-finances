package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

// financeConnectedApps registers the connected-apps page inside the workspace
// group when the remote MCP endpoint exists (saas with an https URL).
func financeConnectedApps(g *router.Group, app *vukem.App) error {
	stack, err := mcpOf(app)
	if err != nil || stack == nil {
		return err
	}

	ch := &ledger.ConnectedAppsHandlers{Grants: stack.oauth.Store}

	g.Get("/connected-apps", ch.Show).Name("connected-apps.index")
	g.Delete("/connected-apps/{grant}", ch.Destroy).Name("connected-apps.destroy").SecretParams("grant")

	return nil
}
