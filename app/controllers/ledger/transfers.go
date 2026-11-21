package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func transferInput(in requests.AccountTransfer) finance.TransferInput {
	return finance.TransferInput{
		FromAccountID: id(in.FromAccountID), ToAccountID: id(in.ToAccountID),
		Amount: in.Amount, Description: in.Description, Date: in.Date,
	}
}

// StoreTransfer moves money between two accounts of the workspace.
func (h *Handlers) StoreTransfer(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AccountTransferSchema)
	if err != nil {
		return transactionError(c, err)
	}

	t, err := h.Fin.CreateTransfer(c.Context(), transferInput(in))
	if err != nil {
		return transactionError(c, err)
	}

	if c.WantsJSON() {
		return c.Created("/transactions", t)
	}

	return done(c, "/transactions", "Transfer added.")
}

// UpdateTransfer edits a transfer through either of its legs.
func (h *Handlers) UpdateTransfer(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	tid, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AccountTransferSchema)
	if err != nil {
		return transactionError(c, err)
	}

	t, err := h.Fin.UpdateTransfer(c.Context(), tid, transferInput(in))
	if err != nil {
		return transactionError(c, err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, t)
	}

	return done(c, "/transactions", "Transfer updated.")
}

// DestroyTransfer deletes both legs of a transfer.
func (h *Handlers) DestroyTransfer(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	tid, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteTransfer(c.Context(), tid); err != nil {
		return err
	}

	return done(c, "/transactions", "Transfer deleted.")
}
