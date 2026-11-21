package workspaces

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

var (
	errOwnerRole   = errs.New(http.StatusConflict, "The owner's role changes only by transferring the ownership.")
	errGrantOwner  = errs.New(http.StatusUnprocessableEntity, "The owner role is given only by transferring the ownership.")
	errNotOwner    = errs.New(http.StatusForbidden, "Only the owner can do that.")
	errNotMember   = errs.New(http.StatusUnprocessableEntity, "Choose a member of this workspace.")
	errOwnerLeaves = errs.New(http.StatusConflict, "The owner cannot leave: transfer the ownership first.")
)

// lockWorkspace takes the row lock that serializes every change of the
// members of the workspace of ctx (SQLite already serializes writers), so
// two requests cannot both pass a check that the other invalidates.
func lockWorkspace(ctx context.Context, tx *database.Tx) error {
	id, ok := TenantID(ctx)
	if !ok {
		return errs.ErrNotFound
	}

	_, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, id)

	return err
}

// ChangeRole sets the role of userID in the workspace of ctx to admin,
// member or viewer. The owner is never changed here and nobody becomes
// owner here: TransferOwnership is the only way the owner moves, so the
// workspace has exactly one owner at all times. The caller needs the
// members.manage ability.
func (s *Service) ChangeRole(ctx context.Context, userID int64, role string) error {
	if role == models.RoleOwner {
		return errGrantOwner
	}

	if !slices.Contains([]string{models.RoleAdmin, models.RoleMember, models.RoleViewer}, role) {
		return errBadRole
	}

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx); err != nil {
			return err
		}

		m, err := Scoped[models.Membership](ctx, tx).Where("user_id", userID).First(ctx)
		if err != nil {
			return err
		}

		if m.Role == models.RoleOwner {
			return errOwnerRole
		}

		_, err = Scoped[models.Membership](ctx, tx).Where("id", m.ID).Update(ctx, orm.Set{"role": role})

		return err
	})
}

// TransferOwnership makes toUserID, a member of the workspace of ctx, its
// owner and demotes from (the current owner, read again under the lock) to
// admin, in one transaction. Two transfers racing leave exactly one owner:
// the second finds that its caller is no longer the owner.
func (s *Service) TransferOwnership(ctx context.Context, from models.User, toUserID int64) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx); err != nil {
			return err
		}

		current, err := Scoped[models.Membership](ctx, tx).Where("user_id", from.ID).First(ctx)
		if err != nil {
			return err
		}

		if current.Role != models.RoleOwner {
			return errNotOwner
		}

		if toUserID == from.ID {
			return errNotMember
		}

		target, err := Scoped[models.Membership](ctx, tx).Where("user_id", toUserID).First(ctx)
		if errors.Is(err, orm.ErrNotFound) {
			return errNotMember
		}

		if err != nil {
			return err
		}

		if _, err := Scoped[models.Membership](ctx, tx).Where("id", current.ID).
			Update(ctx, orm.Set{"role": models.RoleAdmin}); err != nil {
			return err
		}

		_, err = Scoped[models.Membership](ctx, tx).Where("id", target.ID).
			Update(ctx, orm.Set{"role": models.RoleOwner})

		return err
	})
}

// Leave removes u from the workspace of ctx. The owner cannot leave without
// a transfer: a workspace is never left without its owner.
func (s *Service) Leave(ctx context.Context, u models.User) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx); err != nil {
			return err
		}

		m, err := Scoped[models.Membership](ctx, tx).Where("user_id", u.ID).First(ctx)
		if err != nil {
			return err
		}

		if m.Role == models.RoleOwner {
			return errOwnerLeaves
		}

		_, err = Scoped[models.Membership](ctx, tx).Where("id", m.ID).Delete(ctx)

		return err
	})
}
