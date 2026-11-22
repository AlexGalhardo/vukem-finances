package ledger

import (
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// DeviceHandlers serves the Devices page: the phones the signed-in person
// registered for the sync protocol in the current workspace.
type DeviceHandlers struct {
	Sync *syncproto.Service
}

// Show lists the devices.
func (d *DeviceHandlers) Show(c *web.Ctx) error {
	return d.render(c, nil)
}

// Pair makes a pairing code and answers with the page that shows it once: no
// redirect and no flash, so the code never goes through the session.
func (d *DeviceHandlers) Pair(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	code, err := d.Sync.CreatePairingCode(ctx, syncproto.Actor{WorkspaceID: ws, UserID: u.ID})
	if err != nil {
		return back(c, "/devices", err)
	}

	c.Writer().Header().Set("Cache-Control", "no-store")

	return d.render(c, &pages.CreatedPairing{
		Code: code.Code[:4] + "-" + code.Code[4:], Server: serverAddress(c), ExpiresAt: code.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// serverAddress is the address the phone types: the one this request came to.
func serverAddress(c *web.Ctx) string {
	r := c.Request()

	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}

	return scheme + "://" + r.Host
}

func (d *DeviceHandlers) render(c *web.Ctx, pairing *pages.CreatedPairing) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	list, err := d.Sync.Devices(ctx, syncproto.Actor{WorkspaceID: ws, UserID: u.ID})
	if err != nil {
		return err
	}

	views := make([]pages.DeviceView, 0, len(list))

	for _, dev := range list {
		v := pages.DeviceView{
			ID: dev.DeviceID, Name: dev.Name, Platform: dev.Platform, LastPullSeq: dev.LastPullSeq, Status: "live",
			CreatedAt: dev.CreatedAt.UTC().Format(time.RFC3339),
		}

		if dev.LastSeenAt.Valid {
			v.LastSeenAt = dev.LastSeenAt.V.UTC().Format(time.RFC3339)
		}

		if dev.RevokedAt.Valid {
			v.RevokedAt, v.Status = dev.RevokedAt.V.UTC().Format(time.RFC3339), "revoked"
		}

		views = append(views, v)
	}

	return pages.Devices.Render(c, pages.DevicesProps{Devices: views, Pairing: pairing})
}

// Destroy revokes a device of the person: it stops syncing at once and keeps
// its local data until the person clears it.
func (d *DeviceHandlers) Destroy(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	if err := d.Sync.RevokeDevice(ctx, ws, u.ID, c.Param("device")); err != nil {
		return err
	}

	return done(c, "/devices", "Device revoked.")
}
