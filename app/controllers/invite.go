package controllers

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Invite serves the invite link of the personal edition, where there is no
// public sign-up: the link is how a family member gets an account.
type Invite struct {
	acc *accounts.Accounts
	ws  *workspaces.Service
}

// NewInvite returns the invite handlers.
func NewInvite(acc *accounts.Accounts, ws *workspaces.Service) *Invite {
	return &Invite{acc: acc, ws: ws}
}

// Entry is the invite link itself. A signed-in user confirms it; a person
// with no account goes to register; one with an account signs in first.
func (h *Invite) Entry(c *web.Ctx) error {
	token := c.Param("token")

	_, hasAccount, err := h.ws.InvitedAddress(c.Context(), token)
	if err != nil {
		return err
	}

	if _, err := h.acc.Web.User(c); err == nil {
		return pages.InviteAccept.Render(c, pages.InviteAcceptProps{Token: token})
	}

	if hasAccount {
		c.Session().Put(auth.IntendedKey, "/invites/"+token)

		return c.Redirect("/login")
	}

	return c.Redirect("/invites/" + token + "/register")
}

// ShowRegister renders the registration form of an invited person.
func (h *Invite) ShowRegister(c *web.Ctx) error {
	token := c.Param("token")

	address, hasAccount, err := h.ws.InvitedAddress(c.Context(), token)
	if err != nil {
		return err
	}

	if hasAccount {
		return c.Redirect("/invites/" + token)
	}

	return pages.InviteRegister.Render(c, pages.InviteRegisterProps{Token: token, Email: address})
}

// Register creates the account and joins the workspace of the invite.
func (h *Invite) Register(c *web.Ctx) error {
	in, err := web.Parse(c, requests.InviteRegisterSchema)
	if err != nil {
		return err
	}

	hash, err := h.acc.Hasher.Make(in.Password)
	if err != nil {
		return err
	}

	if _, _, err := h.ws.RegisterByInvite(c.Context(), c.Param("token"), in.Name, hash); err != nil {
		return err
	}

	inertia.Flash(c, "success", h.acc.T(c.Context(), "auth.account_created"))

	return c.Redirect("/login")
}
