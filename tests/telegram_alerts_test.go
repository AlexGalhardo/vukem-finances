package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// A notification goes to the linked chat of a person who turned Telegram on,
// to nobody else, and a Telegram failure never fails the alert job.
func TestTelegramAlertChannel(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, ana)
	join(t, app, ana, bob.User, models.RoleMember)

	linkChat(t, bot, ana, 9801)
	linkChat(t, bot, asMember(ana, bob), 9802)

	svc := telegram.Attach(bot, alerts.New(app.App, service(t, app), softService(t, app)))
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acct := firstAccount(t, app, ana)
	food := categoryID(t, app, ana, "Groceries")

	if on, _ := svc.TelegramEnabled(ctx, ana.User.ID); on {
		t.Fatal("Telegram alerts are on by default")
	}

	// Ana opts in; Bob never does.
	if err := svc.SetTelegram(ctx, ana.User.ID, models.RoleOwner, true); err != nil {
		t.Fatal(err)
	}

	if on, _ := svc.TelegramEnabled(ctx, ana.User.ID); !on {
		t.Fatal("the switch did not stick")
	}

	// Turning it on changed nothing else: the mail default of an owner is still on.
	prefs, err := svc.Preferences(ctx, ana.User.ID, models.RoleOwner)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range prefs {
		if !p.InApp || !p.Email {
			t.Fatalf("%s lost its defaults: %+v", p.Kind, p)
		}
	}

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	f.Reset()
	spend(t, fin, ana, acct, food, "85.00", today())

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	got := f.Sent(9801)
	if len(got) != 1 {
		t.Fatalf("Ana's chat got %d messages: %v", len(got), got)
	}

	mustContain(t, got[0], "Budget alert", "Groceries", "80%", "$85.00", "$100.00", "Ana household")

	if n := len(f.Sent(9802)); n != 0 {
		t.Fatalf("Bob did not opt in and his chat got %d messages", n)
	}

	// The same alert is not sent again, however often the job runs.
	for range 3 {
		if err := svc.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}

	if n := len(f.Sent(9801)); n != 1 {
		t.Fatalf("%d messages after repeated runs, want 1", n)
	}

	// The text is escaped: a category named with markup stays text.
	db, _ := app.DB()

	if _, err := orm.QueryOf[models.Category](db).Where("id", food).Update(ctx, orm.Set{"name": "<b>Food</b> & more"}); err != nil {
		t.Fatal(err)
	}

	// Telegram down: the job still succeeds, the in-app alert exists, the claim is released.
	f.FailSend(true)
	f.Reset()
	spend(t, fin, ana, acct, food, "15.00", today())

	if err := svc.Run(ctx); err != nil {
		t.Fatalf("a Telegram failure failed the alert job: %v", err)
	}

	// One in-app row per member: the Telegram failure did not stop them.
	if n := len(notificationsOf(t, app, ana, alerts.KindBudget100)); n != 2 {
		t.Fatalf("budget_100 in-app notifications = %d, want 2", n)
	}

	pending, err := workspaces.Scoped[models.Notification](ctx, db).Where("kind", alerts.KindBudget100).Where("user_id", ana.User.ID).First(t.Context())
	if err != nil || pending.TelegrammedAt.Valid {
		t.Fatalf("the claim of a failed send was kept: %+v %v", pending, err)
	}

	// Telegram back: the next run delivers it once, escaped.
	f.FailSend(false)
	f.Reset()

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	got = f.Sent(9801)
	if len(got) != 1 {
		t.Fatalf("%d messages after Telegram came back, want 1", len(got))
	}

	mustContain(t, got[0], "&lt;b&gt;Food&lt;/b&gt; &amp; more", "100%")

	// Opting out stops it: the next alert (a goal) is not sent.
	if err := svc.SetTelegram(ctx, ana.User.ID, models.RoleOwner, false); err != nil {
		t.Fatal(err)
	}

	if on, _ := svc.TelegramEnabled(ctx, ana.User.ID); on {
		t.Fatal("the switch did not turn off")
	}
}

// With the bot off there is no channel, and a person's saved switch sends nothing.
func TestAlertsWithoutTheBotSendNothing(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	svc := alertsService(t, app)
	ctx := wsCtx(t, ana)

	if svc.Telegram != nil {
		t.Fatal("a Telegram channel exists with the bot off")
	}

	if err := svc.SetTelegram(ctx, ana.User.ID, models.RoleOwner, true); err != nil {
		t.Fatal(err)
	}

	fin := softService(t, app)
	food := categoryID(t, app, ana, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, ana, firstAccount(t, app, ana), food, "90.00", today())

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	rows := notificationsOf(t, app, ana, alerts.KindBudget80)
	if len(rows) != 1 || rows[0].Telegram {
		t.Fatalf("notifications = %+v, want one with no Telegram delivery", rows)
	}
}
