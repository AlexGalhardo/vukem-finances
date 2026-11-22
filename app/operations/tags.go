package operations

import (
	"context"
	"strings"

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

type tagOps struct {
	List   *Op[NoInput, List[TagOut]]
	Create *Op[TagCreateIn, TagOut]
	Update *Op[TagUpdateIn, TagOut]
	Delete *Op[UIDIn, Empty]
}

func (c *Catalog) defineTags() {
	c.Tags = tagOps{
		List: Define(c.Registry, Operation[NoInput, List[TagOut]]{
			Name: "tags.list", Summary: "List the tags", Scope: ScopeTagsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[TagOut], error) { return c.tagsList(ctx) },
		}),
		Create: Define(c.Registry, Operation[TagCreateIn, TagOut]{
			Name: "tags.create", Summary: "Create a tag", Scope: ScopeTagsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in TagCreateIn) (TagOut, error) { return c.tagsCreate(ctx, in) },
		}),
		Update: Define(c.Registry, Operation[TagUpdateIn, TagOut]{
			Name: "tags.update", Summary: "Rename or recolor a tag", Scope: ScopeTagsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in TagUpdateIn) (TagOut, error) { return c.tagsUpdate(ctx, in) },
		}),
		Delete: Define(c.Registry, Operation[UIDIn, Empty]{
			Name: "tags.delete", Summary: "Delete a tag and take it off its transactions", Scope: ScopeTagsWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (Empty, error) { return c.tagsDelete(ctx, in) },
		}),
	}
}

func (c *Catalog) tagsList(ctx context.Context) (List[TagOut], error) {
	all, err := c.Fin.Tags(ctx)
	if err != nil {
		return List[TagOut]{}, err
	}

	data := make([]TagOut, 0, len(all))
	for _, x := range all {
		data = append(data, tagOut(x))
	}

	return List[TagOut]{Data: data}, nil
}

func (c *Catalog) tagsCreate(ctx context.Context, in TagCreateIn) (TagOut, error) {
	if err := c.checkUID(in.UID); err != nil {
		return TagOut{}, err
	}

	if old, found, err := existing[models.Tag](ctx, c, in.UID); err != nil || found {
		if err != nil {
			return TagOut{}, err
		}

		if old.DeletedAt.Valid {
			return TagOut{}, errUIDDeleted
		}

		if !strings.EqualFold(old.Name, strings.TrimSpace(in.Name)) {
			return TagOut{}, errUIDDiffers
		}

		MarkExisting(ctx)

		return tagOut(old), nil
	}

	x, err := c.Fin.CreateTagWithUID(ctx, in.UID, in.Name, in.Color)
	if err != nil {
		return TagOut{}, err
	}

	return tagOut(x), nil
}

func (c *Catalog) tagsUpdate(ctx context.Context, in TagUpdateIn) (TagOut, error) {
	x, err := c.tagByUID(ctx, in.UID)
	if err != nil {
		return TagOut{}, err
	}

	if err := checkVersion(ctx, x.Version, tagOut(x)); err != nil {
		return TagOut{}, err
	}

	name, color := x.Name, x.Color
	if in.Name != nil {
		name = *in.Name
	}

	if in.Color != nil {
		color = *in.Color
	}

	saved, err := c.Fin.UpdateTag(ctx, x.ID, name, color)
	if err != nil {
		return TagOut{}, err
	}

	return tagOut(saved), nil
}

func (c *Catalog) tagsDelete(ctx context.Context, in UIDIn) (Empty, error) {
	x, err := c.tagByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	if err := checkVersion(ctx, x.Version, tagOut(x)); err != nil {
		return Empty{}, err
	}

	return Empty{}, c.Fin.DeleteTag(ctx, x.ID)
}
