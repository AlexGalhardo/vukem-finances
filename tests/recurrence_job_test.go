package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/schedule"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/routes"
)

func TestRecurrenceJobRuns(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	fin := softService(t, app)

	rules := map[who]string{}

	for _, w := range []who{ana, bob} {
		r, err := fin.CreateRecurrence(wsCtx(t, w), monthlyRule(firstAccount(t, app, w), "2027-01-10"))
		if err != nil {
			t.Fatal(err)
		}

		rules[w] = r.UID
	}

	clock.Set(day("2027-03-10"))

	run := func() {
		t.Helper()

		if err := jobs.HandleRecurrencesGenerate(app.App)(t.Context(), jobs.RecurrencesGenerate{}); err != nil {
			t.Fatal(err)
		}
	}

	run()
	run()

	// Each workspace got its own Feb and Mar, and nothing crossed over.
	for w, uid := range rules {
		if got := occurrenceDates(t, app, w, uid); len(got) != 3 {
			t.Errorf("workspace %d: dates = %v, want 3", w.Workspace.ID, got)
		}
	}
}

func TestRecurrenceJobIsScheduledHourly(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	s := schedule.New(schedule.Options{})
	routes.Schedule(s, app.App)

	for _, task := range s.Tasks() {
		if task.Name() == "job:recurrences_generate" {
			if task.Expression() != "0 * * * *" {
				t.Fatalf("recurrences_generate runs at %q, want hourly", task.Expression())
			}

			return
		}
	}

	t.Fatal("recurrences_generate is not scheduled")
}
