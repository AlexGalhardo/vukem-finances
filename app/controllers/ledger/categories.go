package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Categories lists the categories and the tags.
func (h *Handlers) Categories(c *web.Ctx) error {
	cats, err := h.Fin.Categories(c.Context())
	if err != nil {
		return err
	}

	tags, err := h.Fin.Tags(c.Context())
	if err != nil {
		return err
	}

	return pages.Categories.Render(c, pages.CategoriesProps{
		Categories: cats, Tags: tags, Colors: models.CategoryColors, Icons: models.CategoryIcons,
		MaxTags: models.MaxTagsPerWorkspace,
	})
}

func categoryInput(in requests.Category) finance.CategoryInput {
	return finance.CategoryInput{Name: in.Name, Kind: in.Kind, ParentID: id(in.ParentID), Color: in.Color, Icon: in.Icon}
}

// StoreCategory adds a category.
func (h *Handlers) StoreCategory(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.CategorySchema)
	if err != nil {
		return err
	}

	if _, err := h.Fin.AddCategory(c.Context(), categoryInput(in)); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category added.")
}

// UpdateCategory renames a category and changes its parent, color and icon.
func (h *Handlers) UpdateCategory(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("category")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.CategorySchema)
	if err != nil {
		return err
	}

	if _, err := h.Fin.UpdateCategory(c.Context(), cid, categoryInput(in)); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category updated.")
}

// ArchiveCategory hides a category from the pickers.
func (h *Handlers) ArchiveCategory(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("category")
	if err != nil {
		return err
	}

	if _, err := h.Fin.ArchiveCategory(c.Context(), cid); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category archived.")
}

// UnarchiveCategory brings an archived category back.
func (h *Handlers) UnarchiveCategory(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("category")
	if err != nil {
		return err
	}

	if _, err := h.Fin.UnarchiveCategory(c.Context(), cid); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category restored.")
}

// DestroyCategory deletes a category, moving what used it to reassignTo.
func (h *Handlers) DestroyCategory(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageData); err != nil {
		return err
	}

	cid, err := c.ParamInt("category")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.CategoryDeleteSchema)
	if err != nil {
		return err
	}

	if err := h.Fin.DeleteCategoryReassigning(c.Context(), cid, id(in.ReassignTo)); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category deleted.")
}
