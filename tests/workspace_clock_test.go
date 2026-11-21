package tests

import (
	"net/url"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func TestTodayUsesWorkspaceTimezone(t *testing.T) {
	t.Parallel()

	// 01:30 UTC on 1 December is still 30 November in Sao Paulo (UTC-3).
	clock := vukemtest.NewFakeClock(time.Date(2026, 12, 1, 1, 30, 0, 0, time.UTC))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})

	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	putForm(ana.Client, "/workspace", url.Values{
		"name": {"Home"}, "currency": {"USD"}, "timezone": {"America/Sao_Paulo"},
	}).AssertRedirect("/workspace")

	for _, tc := range []struct {
		w            who
		today, month string
	}{
		{ana, "2026-11-30", "2026-11"},
		{bob, "2026-12-01", "2026-12"},
	} {
		tc.w.Client.Inertia("").Get("/transactions").AssertInertia("Transactions").PropEquals("today", tc.today)
		tc.w.Client.Inertia("").Get("/dashboard").AssertInertia("Dashboard").PropEquals("dashboard.month", tc.month)
		tc.w.Client.Inertia("").Get("/budgets").AssertInertia("Budgets").PropEquals("month", tc.month)
	}
}
