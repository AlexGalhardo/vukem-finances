package api

import (
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/web"

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

func (a *API) categoryUIDs(c *web.Ctx) ([]models.Category, map[int64]string, error) {
	all, err := a.Fin.Categories(c.Context())
	if err != nil {
		return nil, nil, err
	}

	uids := make(map[int64]string, len(all))
	for _, x := range all {
		uids[x.ID] = x.UID
	}

	return all, uids, nil
}

func (a *API) categoryView(c *web.Ctx, x models.Category) (CategoryOut, error) {
	_, uids, err := a.categoryUIDs(c)
	if err != nil {
		return CategoryOut{}, err
	}

	return categoryOut(x, uids), nil
}

// parentID turns the uid of a parent into its id; "" is no parent.
func (a *API) parentID(c *web.Ctx, uid string) (int64, error) {
	if uid == "" {
		return 0, nil
	}

	p, err := a.categoryByUID(c.Context(), uid)

	return p.ID, err
}

func (a *API) categoriesList(c *web.Ctx, _ MeIn) (List[CategoryOut], error) {
	all, uids, err := a.categoryUIDs(c)
	if err != nil {
		return List[CategoryOut]{}, err
	}

	data := make([]CategoryOut, 0, len(all))
	for _, x := range all {
		data = append(data, categoryOut(x, uids))
	}

	return List[CategoryOut]{Data: data}, nil
}

func (a *API) categoriesShow(c *web.Ctx, in UIDIn) (CategoryOut, error) {
	x, err := a.categoryByUID(c.Context(), in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	return a.categoryView(c, x)
}

func (a *API) categoriesCreate(c *web.Ctx, in CategoryCreateIn) (CategoryOut, error) {
	ctx := c.Context()

	if err := a.checkUID(in.UID); err != nil {
		return CategoryOut{}, err
	}

	if old, found, err := existing[models.Category](ctx, a, in.UID); err != nil || found {
		if err != nil {
			return CategoryOut{}, err
		}

		if old.DeletedAt.Valid {
			return CategoryOut{}, errUIDDeleted
		}

		if old.Name != strings.TrimSpace(in.Name) || (in.Kind != "" && old.Kind != in.Kind) {
			return CategoryOut{}, errUIDDiffers
		}

		v, err := a.categoryView(c, old)
		if err != nil {
			return CategoryOut{}, err
		}

		return CategoryOut{}, c.JSON(http.StatusOK, v)
	}

	parent, err := a.parentID(c, in.ParentUID)
	if err != nil {
		return CategoryOut{}, err
	}

	x, err := a.Fin.AddCategory(ctx, finance.CategoryInput{
		UID: in.UID, Name: in.Name, Kind: in.Kind, ParentID: parent, Color: in.Color, Icon: in.Icon,
	})
	if err != nil {
		return CategoryOut{}, err
	}

	return a.categoryView(c, x)
}

func (a *API) categoriesUpdate(c *web.Ctx, in CategoryUpdateIn) (CategoryOut, error) {
	ctx := c.Context()

	x, err := a.categoryByUID(ctx, in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	cur, err := a.categoryView(c, x)
	if err != nil {
		return CategoryOut{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return CategoryOut{}, err
	}

	next := finance.CategoryInput{Name: x.Name, Kind: x.Kind, ParentID: x.ParentID.V, Color: x.Color, Icon: x.Icon}
	if in.Name != nil {
		next.Name = *in.Name
	}

	if in.ParentUID != nil {
		if next.ParentID, err = a.parentID(c, *in.ParentUID); err != nil {
			return CategoryOut{}, err
		}
	}

	if in.Color != nil {
		next.Color = *in.Color
	}

	if in.Icon != nil {
		next.Icon = *in.Icon
	}

	saved, err := a.Fin.UpdateCategory(ctx, x.ID, next)
	if err != nil {
		return CategoryOut{}, err
	}

	return a.categoryView(c, saved)
}

func (a *API) categoriesArchive(c *web.Ctx, in UIDIn) (CategoryOut, error) {
	x, err := a.categoryByUID(c.Context(), in.UID)
	if err != nil {
		return CategoryOut{}, err
	}

	cur, err := a.categoryView(c, x)
	if err != nil {
		return CategoryOut{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return CategoryOut{}, err
	}

	saved, err := a.Fin.ArchiveCategory(c.Context(), x.ID)
	if err != nil {
		return CategoryOut{}, err
	}

	return a.categoryView(c, saved)
}
