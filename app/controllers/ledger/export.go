package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/export"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errDelimiter = errs.New(http.StatusUnprocessableEntity, "The delimiter must be comma or semicolon.")

// ExportTransactionsCSV streams the transactions the list filters match as a
// CSV download. Any member of the workspace may take it, a viewer included:
// it holds what they can already read. The rows are counted first, so a
// result over the cap is refused with a 413 before the first byte instead of
// being cut.
func (h *Handlers) ExportTransactionsCSV(c *web.Ctx) error {
	var opt export.CSVOptions

	switch c.Query("delimiter") {
	case "", "comma":
	case "semicolon":
		opt.Semicolon = true
	default:
		return errDelimiter
	}

	opt.BOM = c.Query("bom") == "1"

	_, filter := txFilters(c)
	ctx := c.Context()

	n, err := h.Fin.CountTransactions(ctx, filter)
	if err != nil {
		return err
	}

	if n > export.MaxCSVRows {
		return export.ErrTooManyRows
	}

	rows, err := export.Transactions(ctx, h.Fin, filter)
	if err != nil {
		return err
	}

	c.SetHeader("Content-Type", "text/csv; charset=utf-8")
	c.SetHeader("Content-Disposition", `attachment; filename="vukem-finances-transactions-`+h.Fin.Today(ctx)+`.csv"`)
	c.SetHeader("X-Content-Type-Options", "nosniff")
	c.SetHeader("Cache-Control", "no-store")

	_, err = export.WriteCSV(c.Writer(), rows, opt)

	return err
}

// ExportWorkspaceJSON streams the whole workspace as one JSON document
// (docs/export-format.md). Only the owner and the admins take it: it holds
// the budgets, goals, rules and member names that a viewer's CSV does not.
func (h *Handlers) ExportWorkspaceJSON(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	ctx := c.Context()

	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.SetHeader("Content-Disposition", `attachment; filename="vukem-finances-export-`+h.Fin.Today(ctx)+`.json"`)
	c.SetHeader("X-Content-Type-Options", "nosniff")
	c.SetHeader("Cache-Control", "no-store")

	return export.WriteJSON(ctx, c.Writer(), h.Fin, h.Fin.Now())
}
