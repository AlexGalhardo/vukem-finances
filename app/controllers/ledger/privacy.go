package ledger

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
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

// ConfirmExport checks the password and sends the browser back to the privacy
// page, which then starts the download. The download is a plain GET so the
// browser saves it as a file, and it is allowed only once after this
// confirmation, within five minutes.
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

	return c.Redirect("/account/privacy?export=ready")
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

// DeleteAccount schedules the deletion of the account of the signed-in person:
// the password and their email typed again. It ends every credential, this
// session included, so the answer goes to the login page.
func (p *PrivacyHandlers) DeleteAccount(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.DeleteAccountSchema)
	if err != nil {
		return err
	}

	if err := p.confirmPassword(c, u, in.Password); err != nil {
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(in.Confirmation), u.Email) {
		return errs.NewValidation().Add("confirmation", "Type your email address exactly to confirm.")
	}

	if err := p.svc.RequestAccountDeletion(c.Context(), u); err != nil {
		return back(c, "/account/privacy", err)
	}

	if err := p.a.Web.Logout(c); err != nil {
		return err
	}

	inertia.ClearHistory(c)

	return c.Redirect("/login")
}

// ShowAccountDeletion tells a person whose account waits to be deleted when it
// will be erased, and offers to cancel. It never cancels by itself: signing in
// is not consent to keep the account.
func (p *PrivacyHandlers) ShowAccountDeletion(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if !u.PendingDeletion() {
		return c.Redirect("/dashboard")
	}

	return pages.AccountDeletion.Render(c, pages.AccountDeletionProps{Email: u.Email, PurgeDate: u.PurgeAt.V.UTC().Format("2006-01-02")})
}

// CancelAccountDeletion brings the account back.
func (p *PrivacyHandlers) CancelAccountDeletion(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := p.svc.CancelAccountDeletion(c.Context(), u); err != nil {
		return back(c, "/account/deletion", err)
	}

	return done(c, "/dashboard", "The deletion of your account was cancelled.")
}

// Show renders the privacy page for the current workspace.
func (p *PrivacyHandlers) Show(c *web.Ctx) error {
	ctx := c.Context()

	ws, err := p.h.WS.Current(ctx)
	if err != nil {
		return err
	}

	props := pages.PrivacyProps{
		WorkspaceName: ws.Name, IsOwner: p.h.can(c, workspaces.AbilityTransfer) == nil, Personal: p.h.WS.Edition.Personal(),
		ExportReady: c.Query("export") == "ready", Erased: c.Query("erased") == "1", CanErase: p.h.WS.Edition.Personal() && p.h.can(c, workspaces.AbilityTransfer) == nil, Pending: ws.PendingDeletion(), GraceDays: int(privacy.GracePeriod.Hours() / 24),
	}

	if ws.PendingDeletion() && ws.PurgeAt.Valid {
		props.PurgeDate = ws.PurgeAt.V.UTC().Format("2006-01-02")
	}

	return pages.Privacy.Render(c, props)
}

// EraseLocal is the personal edition's "erase all local data": the owner, the
// password and the typed word. The database file and its WAL and SHM files are
// removed a moment later and the process ends, so the next start is a new empty
// installation. It works with no network, and sends no mail.
func (p *PrivacyHandlers) EraseLocal(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if !p.h.WS.Edition.Personal() {
		return errs.ErrNotFound
	}

	if err := p.h.can(c, workspaces.AbilityTransfer); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.EraseLocalSchema)
	if err != nil {
		return err
	}

	if err := p.confirmPassword(c, u, in.Password); err != nil {
		return err
	}

	if in.Confirmation != privacy.EraseWord {
		return errs.NewValidation().Add("confirmation", "Type "+privacy.EraseWord+" exactly to confirm.")
	}

	if err := p.svc.ScheduleErase(in.Confirmation); err != nil {
		return back(c, "/account/privacy", err)
	}

	return c.Redirect("/account/privacy?erased=1")
}
