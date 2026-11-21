package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"
)

// RecurrencesGenerate creates the transactions that fell due for the rules of
// every workspace, each workspace in its own timezone. routes/schedule.go runs
// it hourly; a doubled or retried run makes nothing twice.
type RecurrencesGenerate struct{}

// Kind names the job's handler; keep it stable across deploys.
func (RecurrencesGenerate) Kind() string { return "recurrences_generate" }

// HandleRecurrencesGenerate generates what is due.
func HandleRecurrencesGenerate(app *vukem.App) func(ctx context.Context, j RecurrencesGenerate) error {
	return func(ctx context.Context, _ RecurrencesGenerate) error {
		fin, err := service(app)
		if err != nil {
			return err
		}

		return fin.GenerateAllDue(ctx)
	}
}
