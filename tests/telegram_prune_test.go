package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/routes"
	"github.com/AlexGalhardo/vukem/schedule"
)

func TestTelegramPruneJob(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	db, _ := app.DB()
	ctx := t.Context()

	for _, id := range []int64{900_000_001, 900_000_002} {
		if ok, err := bot.Store.ClaimUpdate(ctx, id); err != nil || !ok {
			t.Fatal(ok, err)
		}
	}

	old := time.Now().UTC().Add(-8 * 24 * time.Hour)

	if _, err := orm.QueryOf[models.TelegramUpdate](db).Where("update_id", 900_000_001).Update(ctx, orm.Set{"created_at": old}); err != nil {
		t.Fatal(err)
	}

	// An expired code (and a live one), an old draft (and a fresh one).
	code, _, _ := bot.Store.NewCode(ctx, w.User.ID, w.Workspace.ID)
	_, _ = orm.QueryOf[models.TelegramLinkCode](db).Update(ctx, orm.Set{"expires_at": time.Now().UTC().Add(-48 * time.Hour)})
	live, _, _ := bot.Store.NewCode(ctx, w.User.ID, w.Workspace.ID)

	_ = bot.Store.NewPending(ctx, 1, w.User.ID, w.Workspace.ID, "01J0000000000000000000000A", "{}")
	_, _ = orm.QueryOf[models.TelegramPending](db).Update(ctx, orm.Set{"expires_at": time.Now().UTC().Add(-48 * time.Hour)})
	_ = bot.Store.NewPending(ctx, 1, w.User.ID, w.Workspace.ID, "01J0000000000000000000000B", "{}")

	// A finished rate window.
	say(t, bot, 5555, "/help")

	if err := jobs.HandleTelegramPrune(app.App)(ctx, jobs.TelegramPrune{}); err != nil {
		t.Fatal(err)
	}

	count := func(n int64, err error) int64 {
		if err != nil {
			t.Fatal(err)
		}

		return n
	}

	if n := count(orm.QueryOf[models.TelegramUpdate](db).Where("update_id", 900_000_001).Count(ctx)); n != 0 {
		t.Errorf("the 8 day old update id was kept")
	}

	if n := count(orm.QueryOf[models.TelegramUpdate](db).Where("update_id", 900_000_002).Count(ctx)); n != 1 {
		t.Errorf("a fresh update id was pruned")
	}

	if n := count(orm.QueryOf[models.TelegramLinkCode](db).Count(ctx)); n != 1 {
		t.Errorf("%d codes left, want only the live one (%s %s)", n, code[:1], live[:1])
	}

	if n := count(orm.QueryOf[models.TelegramPending](db).Count(ctx)); n != 1 {
		t.Errorf("%d drafts left, want the fresh one", n)
	}

	if n := count(orm.QueryOf[models.TelegramCounter](db).Count(ctx)); n == 0 {
		t.Error("a live rate window was pruned")
	}
}

func TestTelegramPruneIsScheduledDaily(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	s := schedule.New(schedule.Options{})

	routes.Schedule(s, app.App)

	for _, task := range s.Tasks() {
		if task.Name() == "job:telegram_prune" {
			if task.Expression() != "10 4 * * *" {
				t.Fatalf("telegram_prune runs at %q, want daily at 04:10", task.Expression())
			}

			return
		}
	}

	t.Fatal("telegram_prune is not scheduled")
}
