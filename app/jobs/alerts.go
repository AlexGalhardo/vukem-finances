package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// AlertsEvaluate is the payload of the alerts job; it is defined with the
// finance service, which enqueues it after a write. routes/schedule.go runs it
// hourly with no workspace, which means every workspace.
type AlertsEvaluate = finance.AlertsEvaluate

// HandleAlertsEvaluate makes the notifications that are due and sends their
// mail. An error makes the queue retry, and a retry sends nothing twice.
func HandleAlertsEvaluate(app *vukem.App) func(ctx context.Context, j AlertsEvaluate) error {
	return func(ctx context.Context, j AlertsEvaluate) error {
		ws, err := workspaces.New(app)
		if err != nil {
			return err
		}

		fin, err := finance.New(app, ws)
		if err != nil {
			return err
		}

		bot, err := telegram.New(app, ws, fin)
		if err != nil {
			return err
		}

		svc := telegram.Attach(bot, alerts.New(app, ws, fin))

		if j.WorkspaceID == 0 {
			return svc.RunAll(ctx)
		}

		return svc.Run(workspaces.WithTenant(ctx, j.WorkspaceID))
	}
}
