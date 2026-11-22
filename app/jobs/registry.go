// Package jobs holds the queued jobs.
package jobs

import (
	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"
)

// All lists the job handlers, built from the app so they reach its
// database, clock and mailer.
func All(app *vukem.App) []queue.Registration {
	return []queue.Registration{
		queue.Handle(HandleImportCSV(app)),
		queue.Handle(HandleMonthlySummary(app)),
		queue.Handle(HandleChangesCompact(app)),
		queue.Handle(HandleSyncOpsPrune(app)),
		queue.Handle(HandleRecurrencesGenerate(app)),
		queue.Handle(HandleAlertsEvaluate(app)),
		queue.Handle(HandleAIReseal(app)),
		queue.Handle(HandleAICategorize(app)),
		queue.Handle(HandleAssistantPrune(app)),
		queue.Handle(HandleBillingReconcile(app)),
		queue.Handle(HandleTelegramPrune(app)),
	}
}
