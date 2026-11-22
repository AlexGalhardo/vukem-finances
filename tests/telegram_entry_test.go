package tests

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func TestEntryFlow(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9001)

	t.Run("an expense with a category", func(t *testing.T) {
		say(t, bot, 9001, "-12,50 almoço ontem")

		draft, _ := f.LastSent(9001)
		text, _ := draft.Params["text"].(string)
		mustContain(t, text, "Expense $12.50", "almoço", "Main account")

		if draft.Params["parse_mode"] != "HTML" {
			t.Error("the confirmation is not sent as HTML")
		}

		buttons := fakes.Buttons(draft)
		// The five expense categories, "No category" and "Cancel".
		if len(buttons) != 7 {
			t.Fatalf("buttons = %v, want 5 categories + no category + cancel", buttons)
		}

		if got := len(transactionsOf(t, app, w)); got != 0 {
			t.Fatalf("%d transactions before the tap: nothing is saved until a tap", got)
		}

		tap(t, bot, 9001, buttonData(t, f, 9001, "Groceries"))

		txs := transactionsOf(t, app, w)
		if len(txs) != 1 {
			t.Fatalf("%d transactions after the tap", len(txs))
		}

		got := txs[0]
		if got.Kind != models.KindExpense || got.AmountCents != 1250 || got.Description != "almoço" ||
			got.OccurredOn != time.Now().UTC().Add(-24*time.Hour).Format("2006-01-02") ||
			got.CategoryID.V != categoryID(t, app, w, "Groceries") {
			t.Fatalf("saved %+v", got)
		}

		saved, _ := f.LastSent(9001)
		mustContain(t, saved.Params["text"].(string), "Saved", "$12.50")

		if undo := fakes.Buttons(saved); len(undo) != 1 || !strings.HasPrefix(undo[0], "eu:") {
			t.Fatalf("undo buttons = %v", undo)
		}
	})

	t.Run("an income with a thousand separator", func(t *testing.T) {
		say(t, bot, 9001, "+3.000,00 salary")
		mustContain(t, lastText(t, f, 9001), "Income $3,000.00")

		call, _ := f.LastSent(9001)
		if got := fakes.Buttons(call); len(got) != 4 { // two income categories + no category + cancel
			t.Fatalf("income buttons = %v", got)
		}

		tap(t, bot, 9001, buttonData(t, f, 9001, "Salary"))

		txs := transactionsOf(t, app, w)
		last := txs[len(txs)-1]

		if last.Kind != models.KindIncome || last.AmountCents != 300000 || last.CategoryID.V != categoryID(t, app, w, "Salary") {
			t.Fatalf("saved %+v", last)
		}
	})

	t.Run("1.234,56 is 123456 cents", func(t *testing.T) {
		say(t, bot, 9001, "1.234,56 rent")
		tap(t, bot, 9001, buttonData(t, f, 9001, "No category"))

		txs := transactionsOf(t, app, w)
		last := txs[len(txs)-1]

		if last.AmountCents != 123456 || last.CategoryID.Valid || last.Description != "rent" {
			t.Fatalf("saved %+v", last)
		}
	})

	t.Run("the categories used last come first", func(t *testing.T) {
		say(t, bot, 9001, "-3 bus")
		call, _ := f.LastSent(9001)

		rows := call.Params["reply_markup"].(map[string]any)["inline_keyboard"].([]any)
		first := rows[0].([]any)[0].(map[string]any)["text"]

		if first != "Groceries" {
			t.Fatalf("first category = %v, want the one used last (Groceries)", first)
		}
	})

	t.Run("cancel saves nothing", func(t *testing.T) {
		before := len(transactionsOf(t, app, w))

		say(t, bot, 9001, "-99 mistake")
		tap(t, bot, 9001, buttonData(t, f, 9001, "Cancel"))

		if got := len(transactionsOf(t, app, w)); got != before {
			t.Fatalf("a cancelled draft saved: %d -> %d", before, got)
		}
	})

	t.Run("a message that is not an entry gets help, never an error trace", func(t *testing.T) {
		before := len(f.Sent(9001))

		for _, text := range []string{"hello there", "lunch 12", "0", "-5 x 31/02", strings.Repeat("a", 600), "/nonsense"} {
			say(t, bot, 9001, text)
		}

		got := f.Sent(9001)[before:]
		if len(got) != 6 {
			t.Fatalf("%d replies for 6 messages", len(got))
		}

		for _, text := range got {
			if strings.Contains(text, "panic") || strings.Contains(text, "goroutine") || strings.Contains(text, "Error") {
				t.Errorf("a reply looks like an error trace: %q", text)
			}
		}

		mustContain(t, got[0], "could not read")
		mustContain(t, got[2], "amount")
		mustContain(t, got[3], "date")
		mustContain(t, got[4], "too long")
		mustContain(t, got[5], "Commands")
	})
}

// Two taps on the same confirmation, one after the other or at once, save one transaction.
func TestDoubleTapCreatesOne(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9101)

	say(t, bot, 9101, "-40 dinner")

	data := buttonData(t, f, 9101, "Leisure")

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.CallbackUpdate(nextUpdate(), 9101, data)))
		}()
	}

	wg.Wait()
	tap(t, bot, 9101, data)

	if got := len(transactionsOf(t, app, w)); got != 1 {
		t.Fatalf("%d transactions from one confirmation tapped 21 times", got)
	}

	// A tap on another category of the same draft after it was used does nothing.
	tap(t, bot, 9101, data[:len(data)-1]+"1")

	if got := len(transactionsOf(t, app, w)); got != 1 {
		t.Fatalf("%d transactions", got)
	}
}

// A draft belongs to the chat that made it.
func TestDraftIsBoundToItsChat(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	eve := signUp(t, app, "Eve", "eve@example.com")
	linkChat(t, bot, ana, 9201)
	linkChat(t, bot, eve, 9202)

	say(t, bot, 9201, "-40 dinner")
	data := buttonData(t, f, 9201, "No category")

	tap(t, bot, 9202, data)

	if n := len(transactionsOf(t, app, ana)) + len(transactionsOf(t, app, eve)); n != 0 {
		t.Fatalf("another chat consumed the draft: %d transactions", n)
	}

	// Ana's draft is still hers.
	tap(t, bot, 9201, data)

	if n := len(transactionsOf(t, app, ana)); n != 1 {
		t.Fatalf("%d transactions for Ana", n)
	}
}

// The Undo button works for 60 seconds, once.
func TestUndoWindow(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app, f, bot := tgAppAt(t, clock)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9301)

	save := func(text string) string {
		say(t, bot, 9301, text)
		tap(t, bot, 9301, buttonData(t, f, 9301, "No category"))

		saved, _ := f.LastSent(9301)

		return fakes.Buttons(saved)[0]
	}

	// Inside the window.
	undo := save("-10 in time")
	clock.Advance(59 * time.Second)

	f.Reset()
	tap(t, bot, 9301, undo)

	if got := len(transactionsOf(t, app, w)); got != 0 {
		t.Fatalf("%d transactions after an undo inside the window", got)
	}

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "Undone") {
		t.Fatalf("answer = %v", got)
	}

	// Again: nothing to undo.
	f.Reset()
	tap(t, bot, 9301, undo)

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || containsText(got[0], "Undone") {
		t.Fatalf("a second undo answered %v", got)
	}

	// Outside the window.
	late := save("-20 too late")
	clock.Advance(61 * time.Second)

	f.Reset()
	tap(t, bot, 9301, late)

	if got := len(transactionsOf(t, app, w)); got != 1 {
		t.Fatalf("%d transactions: an undo after 60 seconds deleted", got)
	}

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "60 seconds") {
		t.Fatalf("answer = %v", got)
	}

	// The undone transaction is a soft delete: the row is still there, trashed.
	db, _ := app.DB()

	all, err := orm.QueryOf[models.Transaction](db).WithTrashed().Get(t.Context())
	if err != nil || len(all) != 2 {
		t.Fatalf("rows = %d (%v), want the undone one kept as a tombstone", len(all), err)
	}
}

// A draft is good for 15 minutes.
func TestPendingExpires(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app, f, bot := tgAppAt(t, clock)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9401)

	say(t, bot, 9401, "-5 old")
	data := buttonData(t, f, 9401, "No category")

	clock.Advance(14*time.Minute + 59*time.Second)
	say(t, bot, 9401, "-6 fresh")
	fresh := buttonData(t, f, 9401, "No category")

	clock.Advance(2 * time.Second)

	f.Reset()
	tap(t, bot, 9401, data)

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "expired") {
		t.Fatalf("answer = %v", got)
	}

	if got := len(transactionsOf(t, app, w)); got != 0 {
		t.Fatalf("an expired draft saved: %d", got)
	}

	// The draft made a moment before is still good.
	tap(t, bot, 9401, fresh)

	if got := transactionsOf(t, app, w); len(got) != 1 || got[0].AmountCents != 600 {
		t.Fatalf("the fresh draft: %+v", got)
	}
}

// The plan limit of a Free workspace answers in the chat, and the draft can be retried.
func TestEntryRespectsThePlanLimit(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 9501)

	acct := firstAccount(t, app, w)
	fin := softService(t, app)

	for range 100 {
		if _, err := fin.CreateTransaction(tgCtx(w), finance.TxInput{
			AccountID: acct, Kind: models.KindExpense, Amount: "1.00", Description: "x", Date: today(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	say(t, bot, 9501, "-1 one too many")
	tap(t, bot, 9501, buttonData(t, f, 9501, "No category"))

	mustContain(t, lastText(t, f, 9501), "Upgrade to Pro")

	if got := len(transactionsOf(t, app, w)); got != 100 {
		t.Fatalf("%d transactions, want the limit of 100 kept", got)
	}
}
