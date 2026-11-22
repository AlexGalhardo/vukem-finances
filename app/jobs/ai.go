package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/assistant"
	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
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

// AICategorize is the payload of the categorization job; it is defined with
// the finance service, which enqueues it after a write.
type AICategorize = finance.AICategorize

// HandleAICategorize proposes categories for the uncategorized transactions of
// the workspace of the job: rules first, then the model in batches of 30, at
// most categorize.RunCap transactions. An error makes the queue retry, and a
// retry proposes nothing twice (one suggestion per transaction).
func HandleAICategorize(app *vukem.App) func(ctx context.Context, j AICategorize) error {
	return func(ctx context.Context, j AICategorize) error {
		svc, err := Categorizer(app)
		if err != nil {
			return err
		}

		_, err = svc.Run(workspaces.WithTenant(ctx, j.WorkspaceID), categorize.Options{
			ImportID: j.ImportID, TransactionUID: j.TransactionUID,
		})

		return err
	}
}

// Categorizer builds the suggestions service on the shared AI client.
func Categorizer(app *vukem.App) (*categorize.Service, error) {
	shared, err := ai.Shared(app)
	if err != nil {
		return nil, err
	}

	fin, err := service(app)
	if err != nil {
		return nil, err
	}

	return categorize.New(fin.DB, shared.Client, fin, app.Log, app.Now), nil
}

// AssistantPrune deletes the assistant conversations nobody touched for 90
// days, in every workspace. routes/schedule.go runs it daily.
type AssistantPrune struct{}

// Kind names the job's handler; keep it stable across deploys.
func (AssistantPrune) Kind() string { return "assistant_prune" }

// HandleAssistantPrune removes the expired conversations and their messages.
func HandleAssistantPrune(app *vukem.App) func(ctx context.Context, j AssistantPrune) error {
	return func(ctx context.Context, _ AssistantPrune) error {
		db, err := app.DB()
		if err != nil {
			return err
		}

		_, err = assistant.Prune(ctx, db, app.Now())

		return err
	}
}
