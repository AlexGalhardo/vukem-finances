// Package ledger holds the finance pages: dashboard, transactions, accounts,
// categories, budgets, CSV import, workspace settings and billing. Every
// handler runs inside the workspace the tenant middleware resolved, so the
// services only ever see that workspace's rows.
package ledger

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Handlers serves the finance pages.
type Handlers struct {
	WS  *workspaces.Service
	Fin *finance.Service
}

// New returns the handlers.
func New(ws *workspaces.Service, fin *finance.Service) *Handlers { return &Handlers{WS: ws, Fin: fin} }

func id(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// back answers a failed form: a plan limit sends the user to /billing, the
// other rule violations (422, 409, 413) show their message on the page the
// form is on. Everything else goes to the error handler.
func back(c *web.Ctx, path string, err error) error {
	if err == nil {
		return nil
	}

	e, ok := errors.AsType[*errs.HTTPError](err)
	if !ok || c.WantsJSON() {
		return err
	}

	switch e.Status {
	case http.StatusPaymentRequired:
		inertia.Flash(c, "error", e.Message)
		return c.Redirect("/billing")
	case http.StatusUnprocessableEntity, http.StatusConflict, http.StatusRequestEntityTooLarge:
		inertia.Flash(c, "error", e.Message)
		return c.Redirect(path)
	}

	return err
}

// transactionError keeps a browser modal open. A 422 is flashed as field
// errors and sent back to /transactions, which Inertia reports as onError
// so the typed values stay. JSON clients still receive the original status.
// Returning ValidationError would use the session hook and land on "/".
func transactionError(c *web.Ctx, err error) error {
	if c.WantsJSON() {
		return err
	}

	if verr, ok := errors.AsType[*errs.ValidationError](err); ok {
		return redirectTransactionFields(c, verr.Fields)
	}

	e, ok := errors.AsType[*errs.HTTPError](err)
	if !ok || e.Status != http.StatusUnprocessableEntity {
		return back(c, "/transactions", err)
	}

	return redirectTransactionFields(c, map[string][]string{transactionField(e.Message): {e.Message}})
}

func transactionField(message string) string {
	msg := strings.ToLower(message)

	switch {
	case strings.Contains(msg, "date"):
		return "date"
	case strings.Contains(msg, "kind"):
		return "kind"
	case strings.Contains(msg, "account"):
		return "accountId"
	case strings.Contains(msg, "category"):
		return "categoryId"
	case strings.Contains(msg, "description"):
		return "description"
	default:
		return "amount"
	}
}

func redirectTransactionFields(c *web.Ctx, fields map[string][]string) error {
	if s := c.Session(); s != nil {
		s.Flash(web.ErrorsKey, fields)

		if r := c.Request(); r != nil && r.PostForm != nil {
			s.Flash(web.OldInputKey, web.FormToMap(r.PostForm))
		}
	}

	for _, key := range []string{"date", "kind", "accountId", "categoryId", "description", "amount"} {
		if len(fields[key]) > 0 {
			inertia.Flash(c, "error", fields[key][0])

			break
		}
	}

	return c.Redirect("/transactions")
}

func done(c *web.Ctx, path, message string) error {
	inertia.Flash(c, "success", message)
	return c.Redirect(path)
}

// can fails with a 403 unless the role of the user holds ability.
func (h *Handlers) can(c *web.Ctx, ability string) error { return h.WS.Can(c.Context(), ability) }
