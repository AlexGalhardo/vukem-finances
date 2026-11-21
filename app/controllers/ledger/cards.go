package ledger

import (
	"net/url"
	"strconv"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func cardInput(in requests.Card) finance.CardInput {
	return finance.CardInput{
		Name: in.Name, Limit: in.Limit, ClosingDay: int(id(in.ClosingDay)), DueDay: int(id(in.DueDay)),
		PaymentAccountID: id(in.PaymentAccountID), Brand: in.Brand, Last4: in.Last4,
	}
}

// payers are the accounts a statement can be paid from: not a card, not archived.
func (h *Handlers) payers(c *web.Ctx) ([]models.Account, error) {
	accounts, err := h.Fin.Accounts(c.Context())
	if err != nil {
		return nil, err
	}

	out := make([]models.Account, 0, len(accounts))

	for _, a := range accounts {
		if a.Kind != models.AccountCreditCard && !a.ArchivedAt.Valid {
			out = append(out, a)
		}
	}

	return out, nil
}

// Cards lists the credit cards with limit, statement and due date.
func (h *Handlers) Cards(c *web.Ctx) error {
	cards, err := h.Fin.CardOverviews(c.Context())
	if err != nil {
		return err
	}

	accounts, err := h.payers(c)
	if err != nil {
		return err
	}

	lim, err := h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	return pages.Cards.Render(c, pages.CardsProps{Cards: cards, Accounts: accounts, Brands: finance.CardBrands, Limits: lim})
}

// Card shows one card with its statements; ?ref= picks the statement shown
// (the one open today by default).
func (h *Handlers) Card(c *web.Ctx) error {
	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	card, err := h.Fin.CardOverview(c.Context(), cid)
	if err != nil {
		return err
	}

	list, err := h.Fin.Statements(c.Context(), cid)
	if err != nil {
		return err
	}

	ref := c.Query("ref")
	if ref == "" {
		ref = card.Current.Ref
	}

	detail, err := h.Fin.Statement(c.Context(), cid, ref)
	if err != nil {
		return err
	}

	accounts, err := h.payers(c)
	if err != nil {
		return err
	}

	return pages.CardStatement.Render(c, pages.CardStatementProps{
		Card: card, Statements: list, Statement: detail, Accounts: accounts, Brands: finance.CardBrands,
		Today: h.Fin.Today(c.Context()),
	})
}

// StoreCard adds a card with its account.
func (h *Handlers) StoreCard(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.CardSchema)
	if err != nil {
		return err
	}

	if _, err := h.Fin.CreateCard(c.Context(), cardInput(in)); err != nil {
		return back(c, "/cards", err)
	}

	return done(c, "/cards", "Card added.")
}

// UpdateCard changes a card.
func (h *Handlers) UpdateCard(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.CardSchema)
	if err != nil {
		return err
	}

	path := "/cards/" + strconv.FormatInt(cid, 10)

	if _, err := h.Fin.UpdateCard(c.Context(), cid, cardInput(in)); err != nil {
		return back(c, path, err)
	}

	return done(c, path, "Card updated.")
}

// ArchiveCard hides a card from the pickers.
func (h *Handlers) ArchiveCard(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	if _, err := h.Fin.ArchiveCard(c.Context(), cid); err != nil {
		return back(c, "/cards", err)
	}

	return done(c, "/cards", "Card archived.")
}

// UnarchiveCard brings an archived card back.
func (h *Handlers) UnarchiveCard(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	if _, err := h.Fin.UnarchiveCard(c.Context(), cid); err != nil {
		return back(c, "/cards", err)
	}

	return done(c, "/cards", "Card restored.")
}

// DestroyCard deletes a card, its account and its transactions.
func (h *Handlers) DestroyCard(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteCard(c.Context(), cid); err != nil {
		return err
	}

	return done(c, "/cards", "Card deleted.")
}

// PayStatement pays a statement of a card from an account.
func (h *Handlers) PayStatement(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	cid, err := c.ParamInt("card")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.StatementPaymentSchema)
	if err != nil {
		return err
	}

	ref := c.Param("ref")
	path := "/cards/" + strconv.FormatInt(cid, 10) + "?ref=" + url.QueryEscape(ref)

	t, err := h.Fin.PayStatement(c.Context(), cid, ref, finance.PaymentInput{
		FromAccountID: id(in.FromAccountID), Amount: in.Amount, Date: in.Date,
	})
	if err != nil {
		return back(c, path, err)
	}

	if c.WantsJSON() {
		return c.Created(path, t)
	}

	return done(c, path, "Payment recorded.")
}
