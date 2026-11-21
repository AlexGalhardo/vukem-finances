package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// upcomingDays is how far ahead the dashboard and the recurrences page look.
const upcomingDays = 30

func recurrenceInput(in requests.Recurrence) finance.RecurrenceInput {
	return finance.RecurrenceInput{
		AccountID: id(in.AccountID), CategoryID: id(in.CategoryID), Kind: in.Kind, Amount: in.Amount,
		Description: in.Description, Frequency: in.Frequency, Interval: int(id(in.Interval)), StartDate: in.StartDate,
		EndDate: in.EndDate, MaxCount: int(id(in.MaxCount)), Active: in.Active, Backfill: in.Backfill,
	}
}

// Recurrences lists the rules with what is due in the next 30 days.
func (h *Handlers) Recurrences(c *web.Ctx) error {
	rules, err := h.Fin.Recurrences(c.Context())
	if err != nil {
		return err
	}

	upcoming, err := h.Fin.Upcoming(c.Context(), upcomingDays)
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

	return pages.Recurrences.Render(c, pages.RecurrencesProps{
		Recurrences: rules, Upcoming: upcoming, Accounts: accounts, Categories: cats, Today: h.Fin.Today(c.Context()),
	})
}

// StoreRecurrence makes a rule and posts what is already due.
func (h *Handlers) StoreRecurrence(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.RecurrenceSchema)
	if err != nil {
		return err
	}

	rule, err := h.Fin.CreateRecurrence(c.Context(), recurrenceInput(in))
	if err != nil {
		return back(c, "/recurrences", err)
	}

	if c.WantsJSON() {
		return c.Created("/recurrences", rule)
	}

	return done(c, "/recurrences", "Recurrence saved.")
}

// UpdateRecurrence changes the occurrences that are not generated yet.
func (h *Handlers) UpdateRecurrence(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	rid, err := c.ParamInt("recurrence")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.RecurrenceSchema)
	if err != nil {
		return err
	}

	rule, err := h.Fin.UpdateRecurrence(c.Context(), rid, recurrenceInput(in))
	if err != nil {
		return back(c, "/recurrences", err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, rule)
	}

	return done(c, "/recurrences", "Recurrence updated.")
}

// DestroyRecurrence deletes a rule and keeps what it generated; ?future=1 also
// deletes the generated transactions dated after today.
func (h *Handlers) DestroyRecurrence(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	rid, err := c.ParamInt("recurrence")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteRecurrence(c.Context(), rid, c.Query("future") == "1"); err != nil {
		return err
	}

	return done(c, "/recurrences", "Recurrence deleted.")
}

// StoreInstallments records an installment purchase as its parts.
func (h *Handlers) StoreInstallments(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.InstallmentSchema)
	if err != nil {
		return transactionError(c, err)
	}

	parts, err := h.Fin.CreateInstallments(c.Context(), finance.InstallmentInput{
		AccountID: id(in.AccountID), CategoryID: id(in.CategoryID), Total: in.Total, Parts: int(id(in.Parts)),
		FirstDate: in.FirstDate, Description: in.Description, Tags: in.Tags,
	})
	if err != nil {
		return transactionError(c, err)
	}

	if c.WantsJSON() {
		return c.Created("/transactions", parts)
	}

	return done(c, "/transactions", "Installments added.")
}

// DestroyInstallments deletes the part in the path and the ones after it, or
// the whole group with ?scope=group. Only parts dated after today go unless
// ?past=1.
func (h *Handlers) DestroyInstallments(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	tid, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}

	if _, err := h.Fin.DeleteInstallments(c.Context(), tid, c.Query("scope") == "group", c.Query("past") == "1"); err != nil {
		return err
	}

	return done(c, "/transactions", "Installments deleted.")
}
