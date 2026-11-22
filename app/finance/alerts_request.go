package finance

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// AlertsEvaluate is the job that makes the notifications a workspace is due
// (app/alerts) and sends their mail. A write enqueues it for its own
// workspace so the delay is seconds; routes/schedule.go runs it hourly with no
// workspace, which means every workspace. It lives here because the write
// path enqueues it and app/alerts imports this package.
type AlertsEvaluate struct {
	WorkspaceID int64 `json:"workspaceId"`
}

// Kind names the job's handler; keep it stable across deploys.
func (AlertsEvaluate) Kind() string { return "alerts_evaluate" }

// requestAlerts asks for an alert evaluation after the rows were committed,
// when at least one of them is an expense. It never fails the write: the
// hourly run catches whatever a failed enqueue misses, and the evaluation is
// idempotent, so asking twice is harmless.
func (s *Service) requestAlerts(ctx context.Context, rows ...models.Transaction) {
	wanted := false

	for i := range rows {
		if rows[i].Kind == models.KindExpense {
			wanted = true

			break
		}
	}

	if !wanted {
		return
	}

	s.RequestAlerts(ctx)
}

// RequestAlerts enqueues an alert evaluation for the workspace of ctx.
func (s *Service) RequestAlerts(ctx context.Context) {
	id, ok := workspaces.TenantID(ctx)
	if !ok {
		return
	}

	if err := s.Queue.Dispatch(ctx, AlertsEvaluate{WorkspaceID: id}); err != nil && s.Log != nil {
		s.Log.Warn("alerts: could not enqueue an evaluation", "workspace", id, "error", err)
	}
}
