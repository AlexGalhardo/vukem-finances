package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// BillingReconcile re-reads the PIX charges that can still change and applies
// refunds, disputes and expiries. routes/schedule.go runs it every 6 hours.
type BillingReconcile struct{}

// Kind names the job's handler; keep it stable across deploys.
func (BillingReconcile) Kind() string { return "billing_reconcile" }

// HandleBillingReconcile runs the reconciliation; it does nothing outside the
// saas edition or while PIX is off.
func HandleBillingReconcile(app *vukem.App) func(ctx context.Context, j BillingReconcile) error {
	return func(ctx context.Context, _ BillingReconcile) error {
		edition, err := config.LoadEdition(app.Source)
		if err != nil {
			return err
		}

		if !edition.SaaS() || !config.LoadBilling(app.Source).Pix.Enabled() {
			return nil
		}

		ws, err := workspaces.New(app)
		if err != nil {
			return err
		}

		b, err := billing.New(app, ws)
		if err != nil {
			return err
		}

		return b.ReconcilePix(ctx)
	}
}
