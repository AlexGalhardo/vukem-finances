package api

import (
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// TagCreateIn is the body of tags.create.
type TagCreateIn struct {
	UID   string `json:"uid"   validate:"max=26"          doc:"Optional uid chosen by the client (a ULID); repeating the call returns the row already made"`
	Name  string `json:"name"  validate:"required,max=120" example:"work"`
	Color string `json:"color" validate:"max=20"          example:"blue"`
}

// TagUpdateIn is the body of tags.update; a field left out stays as it is.
type TagUpdateIn struct {
	UID   string  `json:"uid"   validate:"required,max=26"`
	Name  *string `json:"name"  validate:"max=120"`
	Color *string `json:"color" validate:"max=20"`
}

func (a *API) tagsList(c *web.Ctx, _ MeIn) (List[TagOut], error) {
	all, err := a.Fin.Tags(c.Context())
	if err != nil {
		return List[TagOut]{}, err
	}

	data := make([]TagOut, 0, len(all))
	for _, x := range all {
		data = append(data, tagOut(x))
	}

	return List[TagOut]{Data: data}, nil
}

func (a *API) tagsCreate(c *web.Ctx, in TagCreateIn) (TagOut, error) {
	ctx := c.Context()

	if err := a.checkUID(in.UID); err != nil {
		return TagOut{}, err
	}

	if old, found, err := existing[models.Tag](ctx, a, in.UID); err != nil || found {
		if err != nil {
			return TagOut{}, err
		}

		if old.DeletedAt.Valid {
			return TagOut{}, errUIDDeleted
		}

		if !strings.EqualFold(old.Name, strings.TrimSpace(in.Name)) {
			return TagOut{}, errUIDDiffers
		}

		return TagOut{}, c.JSON(http.StatusOK, tagOut(old))
	}

	x, err := a.Fin.CreateTagWithUID(ctx, in.UID, in.Name, in.Color)
	if err != nil {
		return TagOut{}, err
	}

	return tagOut(x), nil
}

func (a *API) tagsUpdate(c *web.Ctx, in TagUpdateIn) (TagOut, error) {
	ctx := c.Context()

	x, err := a.tagByUID(ctx, in.UID)
	if err != nil {
		return TagOut{}, err
	}

	if err := checkVersion(c, x.Version, tagOut(x)); err != nil {
		return TagOut{}, err
	}

	name, color := x.Name, x.Color
	if in.Name != nil {
		name = *in.Name
	}

	if in.Color != nil {
		color = *in.Color
	}

	saved, err := a.Fin.UpdateTag(ctx, x.ID, name, color)
	if err != nil {
		return TagOut{}, err
	}

	return tagOut(saved), nil
}

// Empty is the output of an operation that answers 204.
type Empty struct{}

func (a *API) tagsDelete(c *web.Ctx, in UIDIn) (Empty, error) {
	x, err := a.tagByUID(c.Context(), in.UID)
	if err != nil {
		return Empty{}, err
	}

	if err := checkVersion(c, x.Version, tagOut(x)); err != nil {
		return Empty{}, err
	}

	return Empty{}, a.Fin.DeleteTag(c.Context(), x.ID)
}
