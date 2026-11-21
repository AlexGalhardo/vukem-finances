package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
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

	balances, err := h.Fin.AccountBalances(c.Context(), "")
	if err != nil {
		return err
	}

	net, err := finance.NetWorthOf(balances)
	if err != nil {
		return err
	}

	return pages.Accounts.Render(c, pages.AccountsProps{
		Accounts: accounts, Limits: lim, Balances: balances, NetWorthCents: net, Kinds: models.AccountKinds(),
	})
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

// UpdateAccount renames an account and changes its kind and opening balance.
func (h *Handlers) UpdateAccount(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	aid, err := c.ParamInt("account")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AccountSchema)
	if err != nil {
		return err
	}

	if _, err := h.Fin.UpdateAccount(c.Context(), aid, in.Name, in.Kind, in.Opening); err != nil {
		return back(c, "/accounts", err)
	}

	return done(c, "/accounts", "Account updated.")
}

// ArchiveAccount hides an account from the pickers and frees its plan place.
func (h *Handlers) ArchiveAccount(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	aid, err := c.ParamInt("account")
	if err != nil {
		return err
	}

	if _, err := h.Fin.ArchiveAccount(c.Context(), aid); err != nil {
		return back(c, "/accounts", err)
	}

	return done(c, "/accounts", "Account archived.")
}

// UnarchiveAccount brings an archived account back.
func (h *Handlers) UnarchiveAccount(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	aid, err := c.ParamInt("account")
	if err != nil {
		return err
	}

	if _, err := h.Fin.UnarchiveAccount(c.Context(), aid); err != nil {
		return back(c, "/accounts", err)
	}

	return done(c, "/accounts", "Account restored.")
}
