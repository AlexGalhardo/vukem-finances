package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// The 21st message of a minute from one chat is dropped (with one polite
// notice, then silence); another chat is not affected.
func TestSecurity_TelegramRateLimit(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	other := signUp(t, app, "Bob", "bob@example.com")

	// The link takes 1 message (the /start).
	linkChat(t, bot, w, 8001)
	linkChat(t, bot, other, 8002)

	f.Reset()

	for range 19 {
		say(t, bot, 8001, "/help")
	}

	if got := len(f.Sent(8001)); got != 19 {
		t.Fatalf("%d replies to 19 messages", got)
	}

	// Message 21 of the minute (the /start was the first).
	say(t, bot, 8001, "/help")
	say(t, bot, 8001, "/help")

	got := f.Sent(8001)
	if len(got) != 20 {
		t.Fatalf("%d replies after 21 messages, want 19 + 1 notice = 20: the 21st must be dropped", len(got))
	}

	mustContain(t, got[19], "Too many messages")

	// Further messages are dropped without any reply: the notice comes once a minute.
	for range 10 {
		say(t, bot, 8001, "-10 coffee")
	}

	if got := len(f.Sent(8001)); got != 20 {
		t.Fatalf("%d replies: a dropped message got an answer", got)
	}

	if n := len(transactionsOf(t, app, w)); n != 0 {
		t.Fatalf("%d transactions", n)
	}

	// Another chat has its own budget.
	say(t, bot, 8002, "/help")
	mustContain(t, lastText(t, f, 8002), "Record")
}

// Link attempts are limited apart: the sixth /start with a code in a minute is
// told to wait, and even a valid code is not tried.
func TestSecurity_TelegramStartRateLimit(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	for range 5 {
		say(t, bot, 8101, "/start AAAAAAAAAAAAAAAAAAAAAA")
	}

	code, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)

	say(t, bot, 8101, "/start "+code)
	mustContain(t, lastText(t, f, 8101), "Too many attempts")

	if n := len(linksOf(t, app)); n != 0 {
		t.Fatalf("a rate limited attempt linked: %d links", n)
	}

}

// Button taps are limited apart: 60 a minute per chat, the rest are dropped.
func TestSecurity_TelegramCallbackRateLimit(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Cy", "cy@example.com")
	linkChat(t, bot, w, 8102)

	f.Reset()

	for range 70 {
		tap(t, bot, 8102, "ex:AAAAAAAAAAAAAAAAAAAAAAAAAA")
	}

	if got := len(f.Calls("answerCallbackQuery")); got != 60 {
		t.Fatalf("%d taps answered, want 60", got)
	}
}

// The counters live in the database: a new process (a second app on the same
// database, with its own limiters) keeps counting where the first stopped.
func TestRateLimitSurvivesRestart(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 8201)

	f.Reset()

	for range 19 {
		say(t, bot, 8201, "/help")
	}

	if got := len(f.Sent(8201)); got != 19 {
		t.Fatalf("%d replies before the restart", got)
	}

	// The restart: a new app over the same database.
	env := tgEnv(f, map[string]string{
		"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas", "DATABASE_URL": app.Source.Get("DATABASE_URL"),
	})

	restarted := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: env})

	fin, err := finance.New(restarted.App, service(t, restarted))
	if err != nil {
		t.Fatal(err)
	}

	again, err := telegram.New(restarted.App, service(t, restarted), fin)
	if err != nil || again == nil {
		t.Fatalf("telegram.New: %v %v", again, err)
	}

	if again == bot || again.Limits == bot.Limits {
		t.Fatal("the restart shares the limiters of the first process")
	}

	// 20 of the minute are spent (1 /start + 19): one more passes, the next is dropped.
	say(t, again, 8201, "/help")
	say(t, again, 8201, "/help")

	got := f.Sent(8201)
	if len(got) != 20 {
		t.Fatalf("%d replies: the restarted process gave the chat a fresh budget", len(got))
	}

	mustContain(t, got[19], "Too many messages")
}
