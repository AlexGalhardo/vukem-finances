package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

const failMarkCreated = `CREATE TRIGGER fail_mark_created BEFORE UPDATE OF transaction_uid ON telegram_pending
	WHEN NEW.transaction_uid IS NOT NULL BEGIN SELECT RAISE(ABORT, 'injected'); END`

func execSQL(t *testing.T, app *vukemtest.App, query string) {
	t.Helper()

	if _, err := mustDB(t, app).SQL().ExecContext(t.Context(), query); err != nil {
		t.Fatal(err)
	}
}

// The entry is saved but recording that fails: the redelivered tap must finish
// the job (one row, with its undo), not answer that the draft expired.
func TestSavedEntryRecoversOnRedelivery(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9301)

	say(t, bot, 9301, "-40 dinner")

	data := buttonData(t, f, 9301, "Leisure")

	execSQL(t, app, failMarkCreated)

	if err := bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.CallbackUpdate(nextUpdate(), 9301, data))); err == nil {
		t.Fatal("the failure to record the entry was not reported")
	}

	execSQL(t, app, "DROP TRIGGER fail_mark_created")
	tap(t, bot, 9301, data)

	if got := len(transactionsOf(t, app, w)); got != 1 {
		t.Fatalf("%d transactions, want 1", got)
	}

	var recorded int

	err := mustDB(t, app).SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM telegram_pending WHERE transaction_uid IS NOT NULL").Scan(&recorded)
	if err != nil || recorded != 1 {
		t.Fatalf("drafts with their transaction = %d (%v), want 1: the undo would be lost", recorded, err)
	}
}

// One update that always fails must not stop the bot for every other chat.
func TestPollerSkipsAnUpdateThatKeepsFailing(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9401)

	say(t, bot, 9401, "-40 dinner")

	data := buttonData(t, f, 9401, "Leisure")

	execSQL(t, app, failMarkCreated)
	f.Reset()

	base := 7_000_000 + nextUpdate()*10
	f.Enqueue(fakes.CallbackUpdate(base, 9401, data))
	f.Enqueue(fakes.TextUpdate(base+1, 9401, "/help"))

	// A short failure is retried, with nothing skipped and nothing said yet.
	for range 19 {
		_, _, _ = bot.PollOnce(t.Context())
	}

	if got := len(f.Sent(9401)); got != 0 {
		t.Fatalf("%d replies while the update was still being retried", got)
	}

	_, _, _ = bot.PollOnce(t.Context())

	if got := len(f.Sent(9401)); got != 2 {
		t.Fatalf("%d replies, want the failure notice and the /help behind it", got)
	}
}
