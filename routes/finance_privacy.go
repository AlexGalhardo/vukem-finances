package routes

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/controllers/ledger"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// financePrivacy registers the personal data export. The password checks share
// one small bucket per person, so the password cannot be guessed through them.
func financePrivacy(g *router.Group, ph *ledger.PrivacyHandlers, confirm func(http.Handler) http.Handler) {
	g.Post("/account/export", ph.ConfirmExport).Name("account.export.confirm").Use(confirm)
	g.Get("/account/export.json", ph.Export).Name("account.export")

	g.Post("/account/deletion", ph.DeleteAccount).Name("account.deletion.store").Use(confirm)
	g.Post("/workspace/deletion", ph.DeleteWorkspace).Name("workspace.deletion.store").Use(confirm)
	g.Post("/workspace/deletion/cancel", ph.CancelWorkspaceDeletion).Name("workspace.deletion.cancel")
}

// pendingAccountGate sends a person whose account waits to be deleted to the
// page that offers to cancel it: nothing else of the app opens until they do.
func pendingAccountGate() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if u, err := auth.UserFrom[models.User](r.Context(), "web"); err == nil && u.PendingDeletion() {
				http.Redirect(w, r, "/account/deletion", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// accountDeletionRoutes are the two routes a pending account may use: outside
// the workspace group, so they need no workspace.
func accountDeletionRoutes(r *router.Router, ph *ledger.PrivacyHandlers, signedIn router.Middleware, confirm func(http.Handler) http.Handler) {
	r.Get("/account/deletion", ph.ShowAccountDeletion).Name("account.deletion.show").Use(signedIn)
	r.Post("/account/deletion/cancel", ph.CancelAccountDeletion).Name("account.deletion.cancel").Use(signedIn, confirm)
}
