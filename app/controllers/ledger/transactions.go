package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Transactions lists the transactions with the filters of the query string.
func (h *Handlers) Transactions(c *web.Ctx) error {
	f := pages.TxFilters{
		From: c.Query("from"), To: c.Query("to"), CategoryID: first(c.Query("category"), c.Query("categoryId")),
		AccountID: first(c.Query("account"), c.Query("accountId")), Kind: c.Query("kind"),
		Q: c.Query("q"), TagID: c.Query("tag"), Min: c.Query("min"), Max: c.Query("max"),
	}

	var tagIDs []int64
	if tid := id(f.TagID); tid > 0 {
		tagIDs = []int64{tid}
	}

	page, err := h.Fin.TransactionPage(c.Context(), finance.TxFilter{
		From: f.From, To: f.To, CategoryID: id(f.CategoryID), AccountID: id(f.AccountID), Kind: f.Kind,
		Q: f.Q, TagIDs: tagIDs, Min: f.Min, Max: f.Max, Page: int(id(c.Query("page"))),
	})
	if err != nil {
		return back(c, "/transactions", err)
	}

	tags, err := h.Fin.Tags(c.Context())
	if err != nil {
		return err
	}

	accounts, err := h.Fin.Accounts(c.Context())
	if err != nil {
		return err
	}

	cats, err := h.Fin.Categories(c.Context())
	if err != nil {
		return err
	}

	return pages.Transactions.Render(c, pages.TransactionsProps{
		Transactions: page.Rows, Accounts: accounts, Categories: cats, Tags: tags, Filters: f,
		Today: h.Fin.Today(c.Context()), Total: page.Total, Page: page.Page, PageSize: page.PageSize,
	})
}

// StoreTransaction records a transaction.
func (h *Handlers) StoreTransaction(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TransactionSchema)
	if err != nil {
		return transactionError(c, err)
	}

	t, err := h.Fin.CreateTransaction(c.Context(), finance.TxInput{
		AccountID: id(in.AccountID), CategoryID: id(in.CategoryID), Kind: in.Kind, Amount: in.Amount,
		Description: in.Description, Date: in.Date, Tags: in.Tags,
	})
	if err != nil {
		return transactionError(c, err)
	}

	if c.WantsJSON() {
		return c.Created("/transactions", t)
	}

	return done(c, "/transactions", "Transaction added.")
}

// UpdateTransaction changes a transaction of the current workspace.
func (h *Handlers) UpdateTransaction(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}
	tid, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}
	in, err := web.Parse(c, requests.TransactionSchema)
	if err != nil {
		return transactionError(c, err)
	}
	row, err := h.Fin.UpdateTransaction(c.Context(), tid, finance.TxInput{
		AccountID:   id(in.AccountID),
		CategoryID:  id(in.CategoryID),
		Kind:        in.Kind,
		Amount:      in.Amount,
		Description: in.Description,
		Date:        in.Date,
		Tags:        in.Tags,
	})
	if err != nil {
		return transactionError(c, err)
	}
	if c.WantsJSON() {
		return c.JSON(http.StatusOK, row)
	}
	return done(c, "/transactions", "Transaction updated.")
}

// DestroyTransaction deletes a transaction.
func (h *Handlers) DestroyTransaction(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	tid, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteTransaction(c.Context(), tid); err != nil {
		return err
	}

	return done(c, "/transactions", "Transaction deleted.")
}

// TransactionOptions answers the pickers of the new transaction and new
// transfer dialogs, which any page can open.
func (h *Handlers) TransactionOptions(c *web.Ctx) error {
	opts, err := h.Fin.QuickAddOptions(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, opts)
}
