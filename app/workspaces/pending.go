package workspaces

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// CodePendingDeletion is the stable error code of a write refused because the
// workspace is in its deletion grace period.
const CodePendingDeletion = "workspace_pending_deletion"

// ErrPendingDeletion is the answer to a write while the workspace waits to be
// purged. 423 Locked: the data is there and readable, but no longer changes.
var ErrPendingDeletion = errs.New(http.StatusLocked,
	"This workspace is scheduled for deletion and cannot be changed. Its owner can cancel the deletion.").
	WithExtension("code", CodePendingDeletion)

// PendingDeletion reports whether the workspace is in its deletion grace
// period. A workspace that does not exist is not pending: it has no data left to
// protect, and every credential of it is gone.
func (s *Service) PendingDeletion(ctx context.Context, id int64) (bool, error) {
	w, err := orm.QueryOf[models.Workspace](s.DB).Where("id", id).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return false, nil
	}

	return w.PendingDeletion(), err
}

// CheckWritable returns ErrPendingDeletion when the workspace of ctx is in its
// grace period. Every surface that changes data calls it (or a wrapper of it):
// the web middleware, Invoke, the Telegram bot and the sync server.
func (s *Service) CheckWritable(ctx context.Context) error {
	id, ok := TenantID(ctx)
	if !ok {
		return nil
	}

	pending, err := s.PendingDeletion(ctx, id)
	if err != nil {
		return err
	}

	if pending {
		return ErrPendingDeletion
	}

	return nil
}

// exemptDuringDeletion are the writes that stay open in a pending workspace:
// cancelling the deletion, the account pages (the export, the deletion of the
// account: about the person, not the workspace data), and moving between
// workspaces (session state, not a change of the pending workspace's data).
func exemptDuringDeletion(r *http.Request) bool {
	switch {
	case r.URL.Path == "/workspace/deletion/cancel", strings.HasPrefix(r.URL.Path, "/account/"):
		return true
	case r.URL.Path == "/workspaces":
		return true
	case strings.HasPrefix(r.URL.Path, "/workspaces/") && strings.HasSuffix(r.URL.Path, "/switch"):
		return true
	}

	return false
}

// RefuseWritesWhilePending is a route middleware, after the tenant middleware:
// any method that is not a read answers 423 while the workspace waits to be
// purged. The check is the route layer's; Invoke, the bot and sync have their
// own, so a surface that skips the web stack is still held.
func (s *Service) RefuseWritesWhilePending() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				if !exemptDuringDeletion(r) {
					if err := s.CheckWritable(r.Context()); err != nil {
						web.Fail(w, r, err)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
