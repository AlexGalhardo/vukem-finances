package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// PrivacyPurge erases the workspaces and the accounts whose deletion grace period
// ended, and lets go of the payment records whose retention ended
// (D-022). It is idempotent: a pass that stopped half way is finished by the next
// one, and a workspace whose subscription cannot be cancelled waits and raises an
// alert in the log. routes/schedule.go runs it once a day.
type PrivacyPurge struct{}

// Kind names the job's handler; keep it stable across deploys.
func (PrivacyPurge) Kind() string { return "privacy_purge" }

// HandlePrivacyPurge runs one purge pass.
func HandlePrivacyPurge(app *vukem.App) func(ctx context.Context, j PrivacyPurge) error {
	return func(ctx context.Context, _ PrivacyPurge) error {
		ws, err := workspaces.New(app)
		if err != nil {
			return err
		}

		svc, err := privacy.New(app, ws)
		if err != nil {
			return err
		}

		// An account that signed in during its grace period holds sessions the purge ends.
		if svc.Access, err = accounts.New(app); err != nil {
			return err
		}

		// The personal edition has no billing: nothing to cancel there.
		if ws.Edition.SaaS() {
			if svc.Billing, err = billing.New(app, ws); err != nil {
				return err
			}
		}

		rep, err := svc.Purge(ctx)
		if err != nil {
			return err
		}

		app.Log.InfoContext(ctx, "privacy purge", "workspaces", rep.Workspaces, "accounts", rep.Accounts,
			"blocked", rep.Blocked, "records_removed", rep.Records)

		return nil
	}
}
