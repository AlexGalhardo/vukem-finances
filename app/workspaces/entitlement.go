package workspaces

import (
	"context"
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Entitled reports whether workspace id has Pro at now: a valid Stripe
// subscription that is not revoked, or a valid PIX entitlement that is not
// revoked. It is the only place that decides, and Limits uses it. Access is
// derived from rows, never stored as a flag.
func (s *Service) Entitled(ctx context.Context, id int64, now time.Time) (bool, error) {
	return s.entitledIn(ctx, s.DB, id, now)
}

// entitledIn is Entitled reading through ex, so a caller inside a transaction
// sees the plan as that transaction sees it.
func (s *Service) entitledIn(ctx context.Context, ex database.Executor, id int64, now time.Time) (bool, error) {
	ctx = WithTenant(ctx, id)

	subs, err := Scoped[models.WorkspaceSubscription](ctx, ex).Get(ctx)
	if err != nil {
		return false, err
	}

	for i := range subs {
		if subs[i].Valid() {
			return true, nil
		}
	}

	pix, err := Scoped[models.PixEntitlement](ctx, ex).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return pix.Granting(now), nil
}
