package reports

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// wsID is the workspace of ctx; the scoped queries match nothing without one.
func wsID(ctx context.Context) int64 {
	id, _ := workspaces.TenantID(ctx)

	return id
}

// categories lists the categories of the workspace, trashed ones too.
func (s *Service) categories(ctx context.Context) ([]models.Category, error) {
	return workspaces.Scoped[models.Category](ctx, s.DB).WithTrashed().Get(ctx)
}

// accounts maps id to account for the accounts of the workspace, trashed ones too.
func (s *Service) accounts(ctx context.Context) (map[int64]models.Account, error) {
	rows, err := workspaces.Scoped[models.Account](ctx, s.DB).WithTrashed().Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]models.Account, len(rows))
	for _, a := range rows {
		out[a.ID] = a
	}

	return out, nil
}

// tags maps id to tag for the tags of the workspace, trashed ones too.
func (s *Service) tags(ctx context.Context) (map[int64]models.Tag, error) {
	rows, err := workspaces.Scoped[models.Tag](ctx, s.DB).WithTrashed().Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]models.Tag, len(rows))
	for _, t := range rows {
		out[t.ID] = t
	}

	return out, nil
}
