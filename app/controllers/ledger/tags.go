package ledger

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// StoreTag adds a tag.
func (h *Handlers) StoreTag(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TagSchema)
	if err != nil {
		return err
	}

	t, err := h.Fin.CreateTag(c.Context(), in.Name, in.Color)
	if err != nil {
		return back(c, "/categories", err)
	}

	if c.WantsJSON() {
		return c.Created("/categories", t)
	}

	return done(c, "/categories", "Tag added.")
}

// UpdateTag renames and recolors a tag.
func (h *Handlers) UpdateTag(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	tid, err := c.ParamInt("tag")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TagSchema)
	if err != nil {
		return err
	}

	t, err := h.Fin.UpdateTag(c.Context(), tid, in.Name, in.Color)
	if err != nil {
		return back(c, "/categories", err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, t)
	}

	return done(c, "/categories", "Tag updated.")
}

// DestroyTag deletes a tag and takes it off its transactions.
func (h *Handlers) DestroyTag(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	tid, err := c.ParamInt("tag")
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteTag(c.Context(), tid); err != nil {
		return err
	}

	return done(c, "/categories", "Tag deleted.")
}

// AttachTag puts a tag on a transaction.
func (h *Handlers) AttachTag(c *web.Ctx) error { return h.changeTag(c, true) }

// DetachTag takes a tag off a transaction.
func (h *Handlers) DetachTag(c *web.Ctx) error { return h.changeTag(c, false) }

func (h *Handlers) changeTag(c *web.Ctx, attach bool) error {
	if err := h.can(c, workspaces.AbilityRecord); err != nil {
		return err
	}

	txID, err := c.ParamInt("transaction")
	if err != nil {
		return err
	}

	tagID, err := c.ParamInt("tag")
	if err != nil {
		return err
	}

	if attach {
		err = h.Fin.AttachTag(c.Context(), txID, tagID)
	} else {
		err = h.Fin.DetachTag(c.Context(), txID, tagID)
	}

	if err != nil {
		return transactionError(c, err)
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}

	return done(c, "/transactions", "Tags updated.")
}
