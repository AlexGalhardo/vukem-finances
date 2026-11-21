package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Accounts lists the accounts.
func (h *Handlers) Accounts(c *web.Ctx) error {
	accounts, err := h.Fin.Accounts(c.Context())
	if err != nil {
		return err
	}

	lim, err := h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	return pages.Accounts.Render(c, pages.AccountsProps{Accounts: accounts, Limits: lim})
}

// StoreAccount adds an account.
func (h *Handlers) StoreAccount(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AccountSchema)
	if err != nil {
		return err
	}

	if _, err := h.Fin.CreateAccount(c.Context(), in.Name, in.Kind, in.Opening); err != nil {
		return back(c, "/accounts", err)
	}

	return done(c, "/accounts", "Account added.")
}

// DestroyAccount deletes an account and its transactions.
func (h *Handlers) DestroyAccount(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	aid, err := c.ParamInt("account")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteAccount(c.Context(), aid); err != nil {
		return err
	}

	return done(c, "/accounts", "Account deleted.")
}
