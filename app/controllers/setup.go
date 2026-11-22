package controllers

import (
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/desktopgate"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errBadSetupCode = errs.New(http.StatusForbidden, "The setup code is missing or wrong.")

// Setup creates the first owner of a personal installation. The page exists
// only while there is no user, and needs the setup secret.
type Setup struct {
	acc    *accounts.Accounts
	ws     *workspaces.Service
	secret *accounts.SetupSecret
}

// NewSetup returns the setup handlers.
func NewSetup(acc *accounts.Accounts, ws *workspaces.Service, secret *accounts.SetupSecret) *Setup {
	return &Setup{acc: acc, ws: ws, secret: secret}
}

func (h *Setup) open(c *web.Ctx) error {
	open, err := h.ws.SetupOpen(c.Context())
	if err != nil {
		return err
	}

	if !open {
		return errs.ErrNotFound
	}

	return nil
}

// Show renders the setup form.
func (h *Setup) Show(c *web.Ctx) error {
	if err := h.open(c); err != nil {
		return err
	}

	return pages.Setup.Render(c, pages.SetupProps{
		Currency:     workspaces.DefaultCurrency,
		CodeRequired: !desktopgate.Verified(c.Context()),
	})
}

// Store checks the setup secret, creates the owner and the workspace, and
// sends the owner to the login.
func (h *Setup) Store(c *web.Ctx) error {
	if err := h.open(c); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.SetupSchema)
	if err != nil {
		return err
	}

	// In the desktop app the gate already proved the request comes from the
	// app window, and nobody there can read the boot log for the code.
	if !desktopgate.Verified(c.Context()) && !h.secret.Check(in.Code) {
		return errBadSetupCode
	}

	hash, err := h.acc.Hasher.Make(in.Password)
	if err != nil {
		return err
	}

	owner := models.User{Name: in.Name, Email: strings.ToLower(strings.TrimSpace(in.Email)), Password: hash}
	if _, _, err := h.ws.CreateOwner(c.Context(), owner, in.Name+"'s workspace", in.Currency); err != nil {
		return err
	}

	h.secret.Spend()
	inertia.Flash(c, "success", h.acc.T(c.Context(), "auth.account_created"))

	return c.Redirect("/login")
}
