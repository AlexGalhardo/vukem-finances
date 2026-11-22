package routes

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
)

// financePrivacy registers the personal data export. The password checks share
// one small bucket per person, so the password cannot be guessed through them.
func financePrivacy(g *router.Group, ph *ledger.PrivacyHandlers, confirm func(http.Handler) http.Handler) {
	g.Post("/account/export", ph.ConfirmExport).Name("account.export.confirm").Use(confirm)
	g.Get("/account/export.json", ph.Export).Name("account.export")
}
