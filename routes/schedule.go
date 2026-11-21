package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/schedule"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
)

// Schedule declares the scheduled tasks. "vukem serve" runs them in process;
// with SCHEDULE_IN_SERVE=false run "vukem schedule:work" or a cron entry
// calling "vukem schedule:run" every minute instead. List them with
// "vukem schedule:list". Tasks must honor ctx: on shutdown it is cancelled
// after SHUTDOWN_TIMEOUT.
func Schedule(s *schedule.Schedule, _ *vukem.App) {
	// The summary of the month that just ended, to the owners of each
	// workspace once it is the 1st, 08:00 or later, in the workspace's own
	// timezone: hourly, and the sent marker makes a doubled run harmless.
	s.Job(jobs.MonthlySummary{}).Hourly().WithoutOverlapping()

	// Create the transactions that fell due for recurring rules, hourly so a
	// rule posts on its day in the workspace's own timezone; the watermark and
	// the natural key make a doubled run harmless.
	s.Job(jobs.RecurrencesGenerate{}).Hourly().WithoutOverlapping()

	// Purge old tombstones and compact the change log, once a day at night.
	s.Job(jobs.ChangesCompact{}).DailyAt("03:30").WithoutOverlapping()

	// Keys sealed with a former APP_KEY are re-sealed with the current one.
	s.Job(jobs.AIReseal{}).Daily().WithoutOverlapping()

	// Re-read the PIX charges that can still change (refunds, disputes, timers)
	// every 6 hours; a no-op in the personal edition and while PIX is off.
	s.Job(jobs.BillingReconcile{}).Cron("15 */6 * * *").WithoutOverlapping()
}
