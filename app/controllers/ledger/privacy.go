package ledger

import (
	"encoding/json"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// reauthKey is the session value that says the password was just confirmed for
// the personal export; reauthWindow is how long it lasts, and it is spent by use.
const (
	reauthKey    = "privacy.export_until"
	reauthWindow = 5 * time.Minute
)

// PrivacyHandlers serve the privacy page: the personal data export, the
// deletion of a workspace and the deletion of the account. Each of them asks
// for the current password first.
type PrivacyHandlers struct {
	h   *Handlers
	a   *accounts.Accounts
	svc *privacy.Service
	now func() time.Time
}

// NewPrivacy returns the privacy handlers.
func NewPrivacy(h *Handlers, a *accounts.Accounts, svc *privacy.Service, now func() time.Time) *PrivacyHandlers {
	return &PrivacyHandlers{h: h, a: a, svc: svc, now: now}
}

// confirmPassword re-authenticates the signed-in person: the session alone is
// not enough for an action that cannot be undone or that hands out all the data.
func (p *PrivacyHandlers) confirmPassword(c *web.Ctx, u models.User, password string) error {
	if ok, err := p.a.Hasher.Verify(u.Password, password); err != nil || !ok {
		web.SecurityEvent(c.Request(), "privacy.reauth_failed", "user_id", u.AuthID())
		return errs.NewValidation().Add("password", p.a.T(c.Context(), "auth.two_factor.wrong_password"))
	}

	return nil
}

// ConfirmExport checks the password and sends the browser to the download. The
// download is a GET so the browser saves it as a file, and it is allowed only
// once after this confirmation, within five minutes.
func (p *PrivacyHandlers) ConfirmExport(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.ConfirmPasswordSchema)
	if err != nil {
		return err
	}

	if err := p.confirmPassword(c, u, in.Password); err != nil {
		return err
	}

	c.Session().Put(reauthKey, p.now().Add(reauthWindow).Unix())

	return c.Redirect("/account/export.json")
}

// Export streams the personal export. Without a fresh password confirmation it
// goes back to the privacy page.
func (p *PrivacyHandlers) Export(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	// The confirmation is spent by the first use, valid or not.
	until := sessionInt64(c.Session().Pull(reauthKey))

	if until < p.now().Unix() {
		return c.Redirect("/account/privacy")
	}

	ctx := c.Context()

	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.SetHeader("Content-Disposition", `attachment; filename="vukem-finances-personal-export-`+p.h.Fin.Today(ctx)+`.json"`)
	c.SetHeader("X-Content-Type-Options", "nosniff")
	c.SetHeader("Cache-Control", "no-store")

	return finishDownload(c.Writer(), p.svc.Export(ctx, c.Writer(), u))
}

// sessionInt64 reads a number back from the session: a store that encodes
// values as JSON returns json.Number or float64, a memory store the original.
func sessionInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	}

	return 0
}

// DeleteWorkspace schedules the deletion of the current workspace: the owner,
// the password and the exact name.
func (p *PrivacyHandlers) DeleteWorkspace(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	// The role is checked first, so a member learns nothing from a wrong password.
	if err := p.h.can(c, workspaces.AbilityTransfer); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.DeleteWorkspaceSchema)
	if err != nil {
		return err
	}

	if err := p.confirmPassword(c, u, in.Password); err != nil {
		return err
	}

	if err := p.svc.RequestWorkspaceDeletion(c.Context(), u, in.Name); err != nil {
		return back(c, "/account/privacy", err)
	}

	return done(c, "/account/privacy", "The workspace is scheduled for deletion.")
}

// CancelWorkspaceDeletion brings the current workspace back. The workspace is
// read-only while it waits, so this is the one write the web stack lets through.
func (p *PrivacyHandlers) CancelWorkspaceDeletion(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := p.svc.CancelWorkspaceDeletion(c.Context(), u); err != nil {
		return back(c, "/account/privacy", err)
	}

	return done(c, "/account/privacy", "The deletion was cancelled.")
}
