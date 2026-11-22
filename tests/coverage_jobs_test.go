package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// scheduledTasks is the whole schedule, by name: a task added or moved has to be
// written here on purpose.
var scheduledTasks = map[string]string{ //nolint:gochecknoglobals // a constant table
	"session:prune":            "0 * * * *",
	"job:monthly_summary":      "0 * * * *",
	"job:recurrences_generate": "0 * * * *",
	"job:alerts_evaluate":      "0 * * * *",
	"job:changes_compact":      "30 3 * * *",
	"job:sync_ops_prune":       "45 3 * * *",
	"job:ai_reseal":            "0 0 * * *",
	"job:assistant_prune":      "20 4 * * *",
	"job:billing_reconcile":    "15 */6 * * *",
	"job:telegram_prune":       "10 4 * * *",
	"job:privacy_purge":        "40 4 * * *",
}

func TestScheduleHoldsExactlyTheDeclaredTasks(t *testing.T) {
	t.Parallel()

	// The test harness keeps sessions in memory; the framework schedules the
	// pruning of the database driver the real server runs with.
	app := newApp(t, map[string]string{"SESSION_DRIVER": "database"})

	s, err := app.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}

	for _, task := range s.Tasks() {
		want, ok := scheduledTasks[task.Name()]
		if !ok {
			t.Errorf("task %s is scheduled but not declared in scheduledTasks", task.Name())

			continue
		}

		seen[task.Name()] = true

		if task.Expression() != want {
			t.Errorf("task %s runs at %q, want %q", task.Name(), task.Expression(), want)
		}
	}

	for name := range scheduledTasks {
		if !seen[name] {
			t.Errorf("task %s is declared but not scheduled", name)
		}
	}
}

func TestJobsThatSweepRunOnAnEmptyInstallation(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	for range 2 { // every one of them is idempotent
		if err := jobs.HandleAIReseal(app.App)(t.Context(), jobs.AIReseal{}); err != nil {
			t.Errorf("ai_reseal: %v", err)
		}

		if err := jobs.HandleAssistantPrune(app.App)(t.Context(), jobs.AssistantPrune{}); err != nil {
			t.Errorf("assistant_prune: %v", err)
		}

		if err := jobs.HandleBillingReconcile(app.App)(t.Context(), jobs.BillingReconcile{}); err != nil {
			t.Errorf("billing_reconcile: %v", err)
		}
	}
}

func TestImportJobRunsTheQueuedImportOnce(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	id := queuedImport(t, app, ana, acc)

	for range 2 {
		if err := jobs.HandleImportCSV(app.App)(t.Context(), jobs.ImportCSV{WorkspaceID: ana.Workspace.ID, ImportID: id}); err != nil {
			t.Fatal(err)
		}
	}

	if got := latestImport(t, app, ana); got.Status != models.ImportDone || got.Imported != 3 {
		t.Fatalf("import = %+v, want done with 3 rows", got)
	}

	if n := rowCountWhere(t, app, "transactions", "description IN ('Coffee', 'Salary', 'Books')"); n != 3 {
		t.Fatalf("transactions = %d, want 3 (a second run writes nothing)", n)
	}
}

func TestMonthlySummaryJobSendsOnTheFirstOnce(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-03-20")
	ana := signUp(t, app, "Ana", "ana@example.com")

	addTransaction(t, ana, firstAccount(t, app, ana), categoryID(t, app, ana, "Housing"), "expense", "50.00", "2027-03-05").
		AssertRedirect("/transactions")

	run := func(at string) int {
		clock.Set(day(at).Add(9 * time.Hour))

		if err := jobs.HandleMonthlySummary(app.App)(t.Context(), jobs.MonthlySummary{}); err != nil {
			t.Fatal(err)
		}

		return len(app.Mails().SentTo("ana@example.com"))
	}

	if got := run("2027-03-25"); got != 0 {
		t.Fatalf("a summary went out on the 25th (%d)", got)
	}

	if got := run("2027-04-01"); got != 1 {
		t.Fatalf("mails on the 1st = %d, want 1", got)
	}

	if got := run("2027-04-01"); got != 1 {
		t.Fatalf("a second run sent again (%d)", got)
	}
}
