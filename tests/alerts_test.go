package tests

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/queue"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func alertsService(t *testing.T, app *vukemtest.App) *alerts.Service {
	t.Helper()

	return alerts.New(app.App, service(t, app), softService(t, app))
}

func kindsOf(cands []alerts.Candidate) map[string]string {
	out := map[string]string{}
	for _, c := range cands {
		out[c.Kind] = c.Subject
	}

	return out
}

func evaluate(t *testing.T, svc *alerts.Service, w who) []alerts.Candidate {
	t.Helper()

	cands, err := svc.Evaluate(wsCtx(t, w))
	if err != nil {
		t.Fatal(err)
	}

	return cands
}

func notificationsOf(t *testing.T, app *vukemtest.App, w who, kind string) []models.Notification {
	t.Helper()

	rows, err := workspaces.Scoped[models.Notification](wsCtx(t, w), mustDB(t, app)).Where("kind", kind).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func TestEvaluate(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-03-18")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	if err := fin.SaveBudget(ctx, finance.BudgetInput{Limit: "200.00"}); err != nil {
		t.Fatal(err)
	}

	if got := kindsOf(evaluate(t, svc, ana)); len(got) != 0 {
		t.Fatalf("nothing spent: %v", got)
	}

	spend(t, fin, ana, acc, food, "79.00", "2027-03-02")

	if got := kindsOf(evaluate(t, svc, ana)); len(got) != 0 {
		t.Fatalf("79%%: %v", got)
	}

	spend(t, fin, ana, acc, food, "1.00", "2027-03-03")

	if got := kindsOf(evaluate(t, svc, ana)); len(got) != 1 || got[alerts.KindBudget80] == "" {
		t.Fatalf("80%%: %v", got)
	}

	spend(t, fin, ana, acc, food, "20.00", "2027-03-04")

	got := kindsOf(evaluate(t, svc, ana))
	if len(got) != 1 || got[alerts.KindBudget100] == "" {
		t.Fatalf("100%%: %v, want only budget_100", got)
	}

	// The subject carries the month, so the next month is a new subject.
	first := got[alerts.KindBudget100]

	clock.Set(day("2027-04-02"))
	spend(t, fin, ana, acc, food, "100.00", "2027-04-01")

	if next := kindsOf(evaluate(t, svc, ana))[alerts.KindBudget100]; next == "" || next == first {
		t.Fatalf("subject in April = %q, first %q", next, first)
	}

	// 200 overall: 120 of March are not April's; April has 100 only.
	spend(t, fin, ana, acc, 0, "100.00", "2027-04-01")

	if got := kindsOf(evaluate(t, svc, ana)); got[alerts.KindOverallBudget100] == "" {
		t.Fatalf("overall budget at 100%%: %v", got)
	}

	// A goal reached.
	g, err := fin.CreateGoal(ctx, finance.GoalInput{Name: "Trip", Target: "50.00"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "50.00"}); err != nil {
		t.Fatal(err)
	}

	if got := kindsOf(evaluate(t, svc, ana)); got[alerts.KindGoalReached] != "goal:"+g.UID {
		t.Fatalf("goal: %v", got)
	}

	// A failed rule.
	rule, err := fin.CreateRecurrence(ctx, monthlyRule(acc, "2027-04-02"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := mustDB(t, app).SQL().ExecContext(t.Context(), "UPDATE recurrences SET last_error = 'quota'"); err != nil {
		t.Fatal(err)
	}

	if got := kindsOf(evaluate(t, svc, ana)); got[alerts.KindRecurrenceFailed] == "" || got[alerts.KindRecurrenceFailed][:11] != "recurrence:" {
		t.Fatalf("rule %s failed: %v", rule.UID, got)
	}
}

func TestEvaluateCards(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-03-16")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "100.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	food := categoryID(t, app, ana, "Groceries")
	spend(t, fin, ana, card.AccountID, food, "79.00", "2027-03-06")

	if got := kindsOf(evaluate(t, svc, ana)); got[alerts.KindCardLimit80] != "" {
		t.Fatalf("79%% of the limit alerted: %v", got)
	}

	spend(t, fin, ana, card.AccountID, food, "1.00", "2027-03-07")

	got := kindsOf(evaluate(t, svc, ana))
	if got[alerts.KindCardLimit80] != "card:"+card.UID+":2027-03" {
		t.Fatalf("card limit: %v", got)
	}

	// The statement of 2027-03-10 falls due on the 20th: four days out is quiet.
	if got[alerts.KindStatementDue] != "" {
		t.Fatalf("statement due four days ahead: %v", got)
	}

	clock.Set(day("2027-03-17"))

	if got := kindsOf(evaluate(t, svc, ana)); got[alerts.KindStatementDue] != "card:"+card.UID+":2027-03-10" {
		t.Fatalf("statement due in three days: %v", got)
	}
}

// TestAlertsAreNotRepeated runs the job many times, and at once, and checks
// that the person is notified and mailed once.
func TestAlertsAreNotRepeated(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	tx := spend(t, fin, ana, acc, food, "85.00", "2027-03-02")

	var wg sync.WaitGroup

	for range 4 {
		wg.Go(func() {
			if err := svc.Run(ctx); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	for range 3 {
		if err := svc.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}

	if n := len(notificationsOf(t, app, ana, alerts.KindBudget80)); n != 1 {
		t.Fatalf("budget_80 notifications = %d, want 1", n)
	}

	mails := func() int {
		n := 0

		for _, m := range app.Mails().SentTo("ana@example.com") {
			if m.Subject == "Groceries budget at 80%" {
				n++
			}
		}

		return n
	}

	if mails() != 1 {
		t.Fatalf("mails = %d, want 1", mails())
	}

	// Falling below 80 percent and rising again in the same month is the same alert.
	if err := fin.DeleteTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, ana, acc, food, "85.00", "2027-03-03")

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if n := len(notificationsOf(t, app, ana, alerts.KindBudget80)); n != 1 || mails() != 1 {
		t.Fatalf("after the dip: %d notifications, %d mails, want 1 and 1", n, mails())
	}
}

// switchQueue refuses every job while down is set, so no mail can be queued.
type switchQueue struct {
	*queue.Fake

	down atomic.Bool
}

func (q *switchQueue) Push(ctx context.Context, m *queue.Message) error {
	if q.down.Load() {
		return errors.New("queue is down")
	}

	return q.Fake.Push(ctx, m)
}

func TestTransactionWriteDoesNotFailOnMailError(t *testing.T) {
	t.Parallel()

	q := &switchQueue{Fake: queue.NewFake()}
	clock := vukemtest.NewFakeClock(day("2027-03-18"))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:        map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		AppOptions: []vukem.Option{vukem.WithQueueDriver(q)},
		Clock:      clock.Now,
	})

	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	q.down.Store(true)

	// The write succeeds although neither the job nor its mail can be queued.
	spend(t, fin, ana, acc, food, "90.00", "2027-03-02")

	// The run itself reports the failed mail, keeps the notification and
	// leaves the mail unclaimed for the retry.
	if err := svc.Run(ctx); err == nil {
		t.Fatal("a mail that could not be queued was reported as sent")
	}

	rows := notificationsOf(t, app, ana, alerts.KindBudget80)
	if len(rows) != 1 || rows[0].EmailedAt.Valid {
		t.Fatalf("notifications = %+v, want one with its mail still owed", rows)
	}

	q.down.Store(false)

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if rows := notificationsOf(t, app, ana, alerts.KindBudget80); len(rows) != 1 || !rows[0].EmailedAt.Valid {
		t.Fatalf("after the retry: %+v", rows)
	}
}
