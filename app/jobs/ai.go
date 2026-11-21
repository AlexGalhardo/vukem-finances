package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// AIReseal re-encrypts the AI keys sealed with an older APP_KEY. routes/schedule.go
// runs it daily.
type AIReseal struct{}

// Kind names the job's handler; keep it stable across deploys.
func (AIReseal) Kind() string { return "ai_reseal" }

// HandleAIReseal re-seals the credentials that need it.
func HandleAIReseal(app *vukem.App) func(ctx context.Context, j AIReseal) error {
	return func(ctx context.Context, _ AIReseal) error {
		edition, err := config.LoadEdition(app.Source)
		if err != nil {
			return err
		}

		vault, err := ai.NewVault(app, config.LoadAI(app.Source, edition).AllowPrivateHosts)
		if err != nil {
			return err
		}

		n, err := vault.Reseal(ctx)
		if n > 0 && app.Log != nil {
			app.Log.InfoContext(ctx, "ai keys re-sealed", "count", n)
		}

		return err
	}
}
