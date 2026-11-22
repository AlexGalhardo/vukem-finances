package operations

import (
	"context"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// CategoryCreateIn is the body of categories.create.
type CategoryCreateIn struct {
	UID       string `json:"uid"        validate:"max=26"          doc:"Optional uid chosen by the client (a ULID); repeating the call returns the row already made"`
	Name      string `json:"name"       validate:"required,max=60" example:"Groceries"`
	Kind      string `json:"kind"       validate:"max=20"          doc:"income or expense; a subcategory takes the kind of its parent" example:"expense"`
	ParentUID string `json:"parent_uid" validate:"max=26"          doc:"The parent category; a category nests one level"`
	Color     string `json:"color"      validate:"max=20"          doc:"A color name: slate, blue, teal, green, olive, amber, orange, red, pink or violet" example:"green"`
	Icon      string `json:"icon"       validate:"max=20"          doc:"An icon name such as cart, home or wallet" example:"cart"`
}

// CategoryUpdateIn is the body of categories.update; a field left out stays as
// it is, and parent_uid "" makes the category top-level. The kind never changes.
type CategoryUpdateIn struct {
	UID       string  `json:"uid"        validate:"required,max=26"`
	Name      *string `json:"name"       validate:"max=60"`
	ParentUID *string `json:"parent_uid" validate:"max=26"`
	Color     *string `json:"color"      validate:"max=20"`
	Icon      *string `json:"icon"       validate:"max=20"`
}

type categoryOps struct {
	List    *Op[NoInput, List[CategoryOut]]
	Show    *Op[UIDIn, CategoryOut]
	Create  *Op[CategoryCreateIn, CategoryOut]
	Update  *Op[CategoryUpdateIn, CategoryOut]
	Archive *Op[UIDIn, CategoryOut]
}

func (c *Catalog) defineCategories() {
	c.Categories = categoryOps{
		List: Define(c.Registry, Operation[NoInput, List[CategoryOut]]{
			Name: "categories.list", Summary: "List the categories", Scope: ScopeCategoriesRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[CategoryOut], error) { return c.categoriesList(ctx) },
		}),
		Show: Define(c.Registry, Operation[UIDIn, CategoryOut]{
			Name: "categories.show", Summary: "Read a category", Scope: ScopeCategoriesRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (CategoryOut, error) { return c.categoriesShow(ctx, in) },
		}),
		Create: Define(c.Registry, Operation[CategoryCreateIn, CategoryOut]{
			Name: "categories.create", Summary: "Create a category", Scope: ScopeCategoriesWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in CategoryCreateIn) (CategoryOut, error) {
				return c.categoriesCreate(ctx, in)
			},
		}),
		Update: Define(c.Registry, Operation[CategoryUpdateIn, CategoryOut]{
			Name: "categories.update", Summary: "Change a category", Scope: ScopeCategoriesWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in CategoryUpdateIn) (CategoryOut, error) {
				return c.categoriesUpdate(ctx, in)
			},
		}),
		Archive: Define(c.Registry, Operation[UIDIn, CategoryOut]{
			Name: "categories.archive", Summary: "Archive a category", Scope: ScopeCategoriesWrite, Kind: KindDestructive, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (CategoryOut, error) { return c.categoriesArchive(ctx, in) },
		}),
	}
}

func (c *Catalog) categoryUIDs(ctx context.Context) ([]models.Category, map[int64]string, error) {
	all, err := c.Fin.Categories(ctx)
	if err != nil {
		return nil, nil, err
	}

	uids := make(map[int64]string, len(all))
	for _, x := range all {
		uids[x.ID] = x.UID
	}

	return all, uids, nil
}

func (c *Catalog) categoryView(ctx context.Context, x models.Category) (CategoryOut, error) {
	_, uids, err := c.categoryUIDs(ctx)
	if err != nil {
		return CategoryOut{}, err
	}

	return categoryOut(x, uids), nil
}

// parentID turns the uid of a parent into its id; "" is no parent.
func (c *Catalog) parentID(ctx context.Context, uid string) (int64, error) {
	if uid == "" {
		return 0, nil
	}

	p, err := c.categoryByUID(ctx, uid)

	return p.ID, err
}

func (c *Catalog) categoriesList(ctx context.Context) (List[CategoryOut], error) {
	all, uids, err := c.categoryUIDs(ctx)
	if err != nil {
		return List[CategoryOut]{}, err
	}

	data := make([]CategoryOut, 0, len(all))
	for _, x := range all {
		data = append(data, categoryOut(x, uids))
	}

	return List[CategoryOut]{Data: data}, nil
}

func (c *Catalog) categoriesShow(ctx context.Context, in UIDIn) (CategoryOut, error) {
	x, err := c.categoryByUID(ctx, in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	return c.categoryView(ctx, x)
}

func (c *Catalog) categoriesCreate(ctx context.Context, in CategoryCreateIn) (CategoryOut, error) {
	if err := c.checkUID(in.UID); err != nil {
		return CategoryOut{}, err
	}

	if old, found, err := existing[models.Category](ctx, c, in.UID); err != nil || found {
		if err != nil {
			return CategoryOut{}, err
		}

		if old.DeletedAt.Valid {
			return CategoryOut{}, errUIDDeleted
		}

		if old.Name != strings.TrimSpace(in.Name) || (in.Kind != "" && old.Kind != in.Kind) {
			return CategoryOut{}, errUIDDiffers
		}

		MarkExisting(ctx)

		return c.categoryView(ctx, old)
	}

	parent, err := c.parentID(ctx, in.ParentUID)
	if err != nil {
		return CategoryOut{}, err
	}

	x, err := c.Fin.AddCategory(ctx, finance.CategoryInput{
		UID: in.UID, Name: in.Name, Kind: in.Kind, ParentID: parent, Color: in.Color, Icon: in.Icon,
	})
	if err != nil {
		return CategoryOut{}, err
	}

	return c.categoryView(ctx, x)
}

func (c *Catalog) categoriesUpdate(ctx context.Context, in CategoryUpdateIn) (CategoryOut, error) {
	x, err := c.categoryByUID(ctx, in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	cur, err := c.categoryView(ctx, x)
	if err != nil {
		return CategoryOut{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
		return CategoryOut{}, err
	}

	next := finance.CategoryInput{Name: x.Name, Kind: x.Kind, ParentID: x.ParentID.V, Color: x.Color, Icon: x.Icon}
	if in.Name != nil {
		next.Name = *in.Name
	}

	if in.ParentUID != nil {
		if next.ParentID, err = c.parentID(ctx, *in.ParentUID); err != nil {
			return CategoryOut{}, err
		}
	}

	if in.Color != nil {
		next.Color = *in.Color
	}

	if in.Icon != nil {
		next.Icon = *in.Icon
	}

	saved, err := c.Fin.UpdateCategory(ctx, x.ID, next)
	if err != nil {
		return CategoryOut{}, err
	}

	return c.categoryView(ctx, saved)
}

func (c *Catalog) categoriesArchive(ctx context.Context, in UIDIn) (CategoryOut, error) {
	x, err := c.categoryByUID(ctx, in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	cur, err := c.categoryView(ctx, x)
	if err != nil {
		return CategoryOut{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
		return CategoryOut{}, err
	}

	saved, err := c.Fin.ArchiveCategory(ctx, x.ID)
	if err != nil {
		return CategoryOut{}, err
	}

	return c.categoryView(ctx, saved)
}
