package workspaces

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// inviteMu serializes RegisterByInvite so one token creates one account.
var inviteMu sync.Mutex

// pendingInvite loads the unused, unexpired invite behind token, or answers
// 404 for any other case so tokens cannot be probed.
func (s *Service) pendingInvite(ctx context.Context, ex database.Executor, token string) (models.Invite, error) {
	inv, err := orm.QueryOf[models.Invite](ex).Where("token_hash", hashToken(token)).WhereNull("accepted_at").First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Invite{}, errs.ErrNotFound
	}

	if err != nil {
		return models.Invite{}, err
	}

	if !inv.ExpiresAt.Valid || s.now().After(inv.ExpiresAt.V) {
		return models.Invite{}, errs.ErrNotFound
	}

	return inv, nil
}

// InvitedAddress returns the address a usable invite was sent to, and
// whether it already has an account (then the person signs in instead).
func (s *Service) InvitedAddress(ctx context.Context, token string) (string, bool, error) {
	inv, err := s.pendingInvite(ctx, s.DB, token)
	if err != nil {
		return "", false, err
	}

	exists, err := orm.QueryOf[models.User](s.DB).Where("email", strings.ToLower(inv.Email)).Exists(ctx)

	return inv.Email, exists, err
}

// RegisterByInvite creates the account of an invited person in the personal
// edition: the email is the one of the invite (the form cannot change it) and
// is marked verified, because the link was delivered to that address. The
// account joins the workspace of the invite with its role, and the token is
// spent, all in one transaction. An unknown, used or expired token, and an
// address that already has an account, answer 404.
func (s *Service) RegisterByInvite(
	ctx context.Context, token, name, passwordHash string,
) (models.User, models.Workspace, error) {
	inviteMu.Lock()
	defer inviteMu.Unlock()

	var (
		u  models.User
		ws models.Workspace
	)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		inv, err := s.pendingInvite(ctx, tx, token)
		if err != nil {
			return err
		}

		email := strings.ToLower(strings.TrimSpace(inv.Email))

		taken, err := orm.QueryOf[models.User](tx).Where("email", email).Exists(ctx)
		if err != nil {
			return err
		}

		if taken {
			return errs.ErrNotFound
		}

		u = models.User{Name: strings.TrimSpace(name), Email: email, Password: passwordHash}
		u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = s.now().UTC(), true

		if err := orm.Create(ctx, tx, &u); err != nil {
			return err
		}

		if err := orm.Create(ctx, tx, &models.Membership{
			WorkspaceID: inv.WorkspaceID, UserID: u.ID, Role: inv.Role,
		}); err != nil {
			return err
		}

		_, err = orm.QueryOf[models.Invite](tx).Where("id", inv.ID).Update(ctx, orm.Set{"accepted_at": s.now().UTC()})
		if err != nil {
			return err
		}

		ws, err = orm.QueryOf[models.Workspace](tx).Find(ctx, inv.WorkspaceID)

		return err
	})

	return u, ws, err
}
