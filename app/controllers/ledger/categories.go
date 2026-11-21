package ledger

import (
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Categories lists the categories.
func (h *Handlers) Categories(c *web.Ctx) error {
	cats, err := h.Fin.Categories(c.Context())
	if err != nil {
		return err
	}

	return pages.Categories.Render(c, pages.CategoriesProps{Categories: cats})
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

	if _, err := h.Fin.CreateCategory(c.Context(), in.Name, in.Kind); err != nil {
		return back(c, "/categories", err)
	}

	return done(c, "/categories", "Category added.")
}
