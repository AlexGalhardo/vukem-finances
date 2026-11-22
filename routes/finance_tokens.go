package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
)

// financeTokens registers the Tokens page inside the workspace group: any
// member may make a token, which is limited by their role at every request.
func financeTokens(g *router.Group, app *vukem.App, acc *accounts.Accounts, h *ledger.Handlers) {
	th := ledger.NewTokens(h, &api.TokenService{DB: h.WS.DB, Prototype: acc.Tokens}, app.Now)

	g.Get("/tokens", th.Show).Name("tokens.index")
	g.Post("/tokens", th.Store).Name("tokens.store")
	g.Delete("/tokens/{token}", th.Destroy).Name("tokens.destroy")

	// The devices of the sync protocol (docs/specs/sync-protocol.md): a person
	// sees and revokes only their own.
	dh := &ledger.DeviceHandlers{Sync: &syncproto.Service{DB: h.WS.DB, Now: app.Now}}

	g.Get("/devices", dh.Show).Name("devices.index")
	g.Delete("/devices/{device}", dh.Destroy).Name("devices.destroy")
}
