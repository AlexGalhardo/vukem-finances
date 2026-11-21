package tests

import (
	"net/url"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// summaryFixture has Ana (UTC) and Bia (Sao Paulo), both with March spending.
func summaryFixture(t *testing.T) (*vukemtest.App, *finance.Service) {
	t.Helper()

	app := newApp(t)

	for name, tz := range map[string]string{"ana": "UTC", "bia": "America/Sao_Paulo"} {
		w := signUp(t, app, name, name+"@example.com")
		addTransaction(t, w, firstAccount(t, app, w), categoryID(t, app, w, "Housing"), "expense", "50.00", "2026-03-05").
			AssertRedirect("/transactions")
		putForm(w.Client, "/workspace", url.Values{"name": {"Home"}, "currency": {"USD"}, "timezone": {tz}}).
			AssertRedirect("/workspace")
	}

	fin, err := finance.New(app.App, service(t, app))
	if err != nil {
		t.Fatal(err)
	}

	return app, fin
}

func TestMonthlySummaryRunsInWorkspaceLocalTime(t *testing.T) {
	t.Parallel()

	app, fin := summaryFixture(t)
	sent := func(who string) int { return len(app.Mails().SentTo(who + "@example.com")) }

	// 2026-04-01 07:00 UTC: too early for UTC, still 04:00 in Sao Paulo.
	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	if sent("ana") != 0 || sent("bia") != 0 {
		t.Fatalf("summaries before 08:00 local: ana %d, bia %d", sent("ana"), sent("bia"))
	}

	// 08:00 UTC: Ana's morning; Bia's is at 11:00 UTC.
	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	if sent("ana") != 1 || sent("bia") != 0 {
		t.Fatalf("at 08:00 UTC: ana %d, bia %d; want 1 and 0", sent("ana"), sent("bia"))
	}

	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	if sent("ana") != 1 || sent("bia") != 1 {
		t.Fatalf("at 11:00 UTC: ana %d, bia %d; want 1 and 1", sent("ana"), sent("bia"))
	}

	// 2026-12-01 01:30 UTC is still 30 November in Sao Paulo: nothing is due there.
	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 12, 1, 1, 30, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	// Other days of the month send nothing.
	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	if sent("ana") != 1 || sent("bia") != 1 {
		t.Errorf("unexpected summaries: ana %d, bia %d", sent("ana"), sent("bia"))
	}
}

func TestMonthlySummarySentOncePerMonth(t *testing.T) {
	t.Parallel()

	app, fin := summaryFixture(t)
	at := time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC)

	// The task runs hourly and may run twice: one mail per workspace and month.
	for range 3 {
		if err := fin.SendDueMonthlySummaries(t.Context(), at); err != nil {
			t.Fatal(err)
		}

		at = at.Add(time.Hour)
	}

	for _, who := range []string{"ana", "bia"} {
		if n := len(app.Mails().SentTo(who + "@example.com")); n != 1 {
			t.Errorf("%s got %d summaries, want 1", who, n)
		}
	}

	vukemtest.AssertDatabaseCount(t, app, "monthly_summary_sent", 2)

	// A new month is a new summary.
	if err := fin.SendDueMonthlySummaries(t.Context(), time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	vukemtest.AssertDatabaseCount(t, app, "monthly_summary_sent", 4)
}
