package privacy

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errNotOwner       = errs.New(http.StatusForbidden, "Only the owner can delete the workspace.")
	errWrongName      = errs.New(http.StatusUnprocessableEntity, "Type the name of the workspace exactly to confirm.")
	errAlreadyPending = errs.New(http.StatusConflict, "This workspace is already scheduled for deletion.")
	errNotPending     = errs.New(http.StatusConflict, "This workspace is not scheduled for deletion.")
	errGraceEnded     = errs.New(http.StatusConflict, "The grace period has ended: this deletion can no longer be cancelled.")
	errPersonalErase  = errs.New(http.StatusConflict,
		"The personal edition has one workspace: use \"Erase all local data\" to remove it.")
)

// dateOf is the day a purge happens, the way the emails and pages print it.
func dateOf(t time.Time) string { return t.UTC().Format("2006-01-02") }

// RequestWorkspaceDeletion schedules the deletion of the workspace of ctx. Only
// its owner can ask, and only by typing its name exactly; the caller has already
// re-authenticated the person. The workspace turns read-only at once (every
// surface refuses writes) and the purge job erases it after the grace period.
// The subscription is cancelled now; if that fails the request stands, the
// failure is raised as an alert and blocks the purge until it succeeds.
func (s *Service) RequestWorkspaceDeletion(ctx context.Context, actor models.User, typedName string) error {
	if s.WS.Edition.Personal() {
		return errPersonalErase
	}

	id, ok := workspaces.TenantID(ctx)
	if !ok {
		return errs.ErrNotFound
	}

	now := s.now().UTC()
	purgeAt := now.Add(GracePeriod)

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		ws, err := s.ownedWorkspace(ctx, tx, id, actor.ID)
		if err != nil {
			return err
		}

		if ws.PendingDeletion() {
			return errAlreadyPending
		}

		if typedName != ws.Name {
			return errWrongName
		}

		if _, err := orm.QueryOf[models.Workspace](tx).Where("id", id).Update(ctx, orm.Set{
			"state": models.StatePendingDeletion, "purge_at": purgeAt, "deletion_requested_at": now,
			"deletion_requested_by": actor.ID, "purge_error": "",
		}); err != nil {
			return err
		}

		return s.record(ctx, tx, EventWorkspaceRequested, s.Ref("workspace", id))
	})
	if err != nil {
		return err
	}

	s.cancelBilling(ctx, id)
	s.notifyMembers(ctx, id, func(to string, ws models.Workspace) mail.Mailable {
		return appmail.WorkspaceDeletionScheduled{To: to, Workspace: ws.Name, PurgeDate: dateOf(purgeAt), URL: s.WS.URL + "/dashboard"}
	})

	return nil
}

// CancelWorkspaceDeletion brings a pending workspace back. Only the owner can.
// The subscription cancelled by the request is not restored: the owner can
// subscribe again.
func (s *Service) CancelWorkspaceDeletion(ctx context.Context, actor models.User) error {
	id, ok := workspaces.TenantID(ctx)
	if !ok {
		return errs.ErrNotFound
	}

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		ws, err := s.ownedWorkspace(ctx, tx, id, actor.ID)
		if err != nil {
			return err
		}

		if !ws.PendingDeletion() {
			return errNotPending
		}

		// Once due, the purge may already have removed rows: bringing the workspace
		// back now would hand the person half of their data.
		if ws.PurgeAt.Valid && !ws.PurgeAt.V.After(s.now()) {
			return errGraceEnded
		}

		if _, err := orm.QueryOf[models.Workspace](tx).Where("id", id).Update(ctx, orm.Set{
			"state": models.StateActive, "purge_at": nil, "deletion_requested_at": nil,
			"deletion_requested_by": 0, "purge_error": "",
		}); err != nil {
			return err
		}

		return s.record(ctx, tx, EventWorkspaceCancelled, s.Ref("workspace", id))
	})
	if err != nil {
		return err
	}

	s.notifyMembers(ctx, id, func(to string, ws models.Workspace) mail.Mailable {
		return appmail.WorkspaceDeletionCancelled{To: to, Workspace: ws.Name, URL: s.WS.URL + "/dashboard"}
	})

	return nil
}

// ownedWorkspace loads the workspace and checks that actor is its owner, from
// the membership row now, never from a role the request carried. A person who
// is not a member gets a 404, so ids cannot be probed.
func (s *Service) ownedWorkspace(ctx context.Context, tx *database.Tx, id, actorID int64) (models.Workspace, error) {
	ws, err := orm.QueryOf[models.Workspace](tx).Find(ctx, id)
	if err != nil {
		return ws, err
	}

	m, err := orm.QueryOf[models.Membership](tx).Where("workspace_id", id).Where("user_id", actorID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return ws, errs.ErrNotFound
	}

	if err != nil {
		return ws, err
	}

	if m.Role != models.RoleOwner {
		return ws, errNotOwner
	}

	return ws, nil
}

// cancelBilling cancels what the workspace pays for. A failure never fails the
// request: it is stored as the reason the purge waits, and logged for the alert.
func (s *Service) cancelBilling(ctx context.Context, workspaceID int64) {
	if s.Billing == nil {
		return
	}

	reason := ""
	if err := s.Billing.CancelWorkspaceSubscriptions(ctx, workspaceID); err != nil {
		reason = PurgeBlockedBilling

		s.Log.ErrorContext(ctx, "alert: the subscription of a workspace waiting for deletion could not be cancelled; "+
			"the purge is blocked until it is", "workspace_ref", s.Ref("workspace", workspaceID))
	}

	if _, err := orm.QueryOf[models.Workspace](s.DB).Where("id", workspaceID).
		Update(ctx, orm.Set{"purge_error": reason}); err != nil {
		s.Log.ErrorContext(ctx, "privacy: could not store the purge state", "err", err)
	}
}

// PurgeBlockedBilling is the purge_error of a workspace whose subscription could
// not be cancelled.
const PurgeBlockedBilling = "billing_cancel_failed"

// notifyMembers mails every member of the workspace. A mail failure is logged
// and ignored: it must never block, or undo, a deletion (the personal edition
// may have no network at all).
func (s *Service) notifyMembers(ctx context.Context, workspaceID int64, build func(to string, ws models.Workspace) mail.Mailable) {
	ws, err := orm.QueryOf[models.Workspace](s.DB).Find(ctx, workspaceID)
	if err != nil {
		s.Log.WarnContext(ctx, "privacy: members were not told", "err", err)
		return
	}

	members, err := orm.QueryOf[models.Membership](s.DB).Where("workspace_id", workspaceID).Get(ctx)
	if err != nil {
		s.Log.WarnContext(ctx, "privacy: members were not told", "err", err)
		return
	}

	for _, m := range members {
		u, err := orm.QueryOf[models.User](s.DB).Find(ctx, m.UserID)
		if err != nil {
			continue
		}

		if err := s.WS.Notify(ctx, build(u.Email, ws)); err != nil {
			s.Log.WarnContext(ctx, "privacy: a member was not told", "err", err)
		}
	}
}
