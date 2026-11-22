package ledger

import (
	"errors"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	appoauth "github.com/AlexGalhardo/vukem-finances/app/oauth"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// ConnectedAppsHandlers serves the connected-apps page: the AI clients the
// signed-in person connected to the current workspace through the remote MCP
// endpoint, with revoke.
type ConnectedAppsHandlers struct {
	Grants *appoauth.GrantStore
}

// Show lists the grants of the person in this workspace.
func (h *ConnectedAppsHandlers) Show(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	list, err := h.Grants.Connected(ctx, u.ID, ws)
	if err != nil {
		return err
	}

	apps := make([]pages.ConnectedApp, 0, len(list))

	for _, g := range list {
		a := pages.ConnectedApp{
			ID: g.ID, Client: g.Client, Scopes: g.Scopes, CreatedAt: g.CreatedAt.UTC().Format(time.RFC3339), Status: "live",
		}

		if g.LastUsedAt.Valid {
			a.LastUsedAt = g.LastUsedAt.Time.UTC().Format(time.RFC3339)
		}

		if g.Revoked {
			a.Status = "revoked"
		}

		apps = append(apps, a)
	}

	return pages.ConnectedApps.Render(c, pages.ConnectedAppsProps{Apps: apps})
}

// Destroy revokes a grant of the person: the client's refresh token stops and
// its access token fails on the next call. A grant of another person or of
// another workspace is a 404.
func (h *ConnectedAppsHandlers) Destroy(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	if err := h.Grants.Revoke(ctx, u.ID, ws, c.Param("grant")); err != nil {
		if errors.Is(err, appoauth.ErrNoGrant) {
			return errs.New(http.StatusNotFound, "Not found.")
		}

		return err
	}

	return done(c, "/connected-apps", "App disconnected.")
}
