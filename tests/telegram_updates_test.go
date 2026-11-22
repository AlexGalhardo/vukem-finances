package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

// Redelivering an update id, one after the other or all at once, does nothing twice.
func TestUpdateIdempotent(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 3001)

	f.Reset()

	update := mustUpdate(t, fakes.TextUpdate(nextUpdate(), 3001, "/help"))

	for range 3 {
		if err := bot.HandleUpdate(t.Context(), update); err != nil {
			t.Fatal(err)
		}
	}

	if got := len(f.Sent(3001)); got != 1 {
		t.Fatalf("update delivered 3 times, %d replies; want 1", got)
	}

	// The same update from many goroutines (a poll overlapping a webhook retry).
	racy := mustUpdate(t, fakes.TextUpdate(nextUpdate(), 3001, "/help"))

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = bot.HandleUpdate(t.Context(), racy)
		}()
	}

	wg.Wait()

	if got := len(f.Sent(3001)); got != 2 {
		t.Fatalf("%d replies after the concurrent redelivery; want 2 in all", got)
	}

	// An entry redelivered creates one draft, one confirmation, and no transaction by itself.
	entry := mustUpdate(t, fakes.TextUpdate(nextUpdate(), 3001, "-12,50 lunch"))
	_ = bot.HandleUpdate(t.Context(), entry)
	_ = bot.HandleUpdate(t.Context(), entry)

	db, _ := app.DB()

	n, err := orm.QueryOf[models.TelegramPending](db).Count(t.Context())
	if err != nil || n != 1 {
		t.Fatalf("%d drafts for one update id (%v), want 1", n, err)
	}
}

// A failure that is the server's own (not the person's) releases the claim, so the redelivery is handled.
func TestUpdateClaimReleasedOnFailure(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	db, _ := app.DB()

	id := int64(nextUpdate())

	first, err := bot.Store.ClaimUpdate(t.Context(), id)
	if err != nil || !first {
		t.Fatalf("first claim = %v, %v", first, err)
	}

	if again, _ := bot.Store.ClaimUpdate(t.Context(), id); again {
		t.Fatal("a second claim won")
	}

	if err := bot.Store.ReleaseUpdate(t.Context(), id); err != nil {
		t.Fatal(err)
	}

	if again, _ := bot.Store.ClaimUpdate(t.Context(), id); !again {
		t.Fatal("a released claim was not claimable")
	}

	if n, _ := orm.QueryOf[models.TelegramUpdate](db).Where("update_id", id).Count(t.Context()); n != 1 {
		t.Fatalf("%d rows for the id", n)
	}
}

// pollOffsets lists the offsets of the getUpdates calls the fake received.
func pollOffsets(f *fakes.Telegram) []int {
	var out []int

	for _, c := range f.Calls("getUpdates") {
		o, _ := c.Params["offset"].(float64)
		out = append(out, int(o))
	}

	return out
}

// A restart in the middle of a batch loses nothing and repeats nothing: the
// next poll asks from the first update that was not handled.
func TestPollerResumesFromOffset(t *testing.T) {
	t.Parallel()

	app, f, first := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, first, w, 4001)

	f.Reset()

	base := 5_000_000 + nextUpdate()*10
	for i := range 5 {
		f.Enqueue(fakes.TextUpdate(base+i, 4001, "/help"))
	}

	// The first process is stopped after it has handled the second update.
	ctx, cancel := context.WithCancel(t.Context())
	first.SetAfterUpdate(func(id int64) {
		if id == int64(base+1) {
			cancel()
		}
	})

	handled, leased, err := first.PollOnce(ctx)
	if err != nil || !leased || handled != 2 {
		t.Fatalf("first poll: handled %d leased %v err %v; want 2 handled", handled, leased, err)
	}

	// A clean shutdown frees the lease (Run does it on the way out).
	if err := first.Store.ReleaseLease(t.Context(), first.Owner); err != nil {
		t.Fatal(err)
	}

	// The restarted process is a new instance with a new owner.
	second, err := telegram.New(app.App, service(t, app), softService(t, app))
	if err != nil {
		t.Fatal(err)
	}

	handled, leased, err = second.PollOnce(t.Context())
	if err != nil || !leased || handled != 3 {
		t.Fatalf("second poll: handled %d leased %v err %v; want the 3 left", handled, leased, err)
	}

	if got := len(f.Sent(4001)); got != 5 {
		t.Fatalf("%d replies for 5 updates (a loss or a repeat)", got)
	}

	offsets := pollOffsets(f)
	if len(offsets) != 2 || offsets[0] != 0 || offsets[1] != base+2 {
		t.Fatalf("poll offsets = %v, want [0 %d]", offsets, base+2)
	}

	// Nothing left: the stored offset moves past the last update.
	if _, _, err := second.PollOnce(t.Context()); err != nil {
		t.Fatal(err)
	}

	if offs := pollOffsets(f); offs[len(offs)-1] != base+5 {
		t.Fatalf("offsets = %v, want the last to be %d", offs, base+5)
	}
}

// Two instances never both poll: the one without the lease does nothing until
// the lease of the other expires.
func TestPollerLease(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app, f, one := tgAppAt(t, clock)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, one, w, 4101)

	two, err := telegram.New(app.App, service(t, app), softService(t, app))
	if err != nil {
		t.Fatal(err)
	}

	f.Reset()
	f.Enqueue(fakes.TextUpdate(7_000_000+nextUpdate(), 4101, "/help"))

	if _, leased, err := one.PollOnce(t.Context()); err != nil || !leased {
		t.Fatalf("first instance: leased %v err %v", leased, err)
	}

	before := len(f.Calls("getUpdates"))

	handled, leased, err := two.PollOnce(t.Context())
	if err != nil || leased || handled != 0 {
		t.Fatalf("second instance while the first holds the lease: handled %d leased %v err %v", handled, leased, err)
	}

	if got := len(f.Calls("getUpdates")); got != before {
		t.Fatalf("the instance without the lease polled Telegram (%d calls)", got-before)
	}

	// The first keeps renewing; the second stays out.
	clock.Advance(time.Minute)

	if _, leased, _ := one.PollOnce(t.Context()); !leased {
		t.Fatal("the holder lost its lease while renewing")
	}

	if _, leased, _ := two.PollOnce(t.Context()); leased {
		t.Fatal("the second instance took a live lease")
	}

	// The first stops renewing (it crashed); its lease runs out and the second takes over.
	clock.Advance(3 * time.Minute)

	if _, leased, err := two.PollOnce(t.Context()); err != nil || !leased {
		t.Fatalf("the second instance did not take the expired lease: %v %v", leased, err)
	}

	if _, leased, _ := one.PollOnce(t.Context()); leased {
		t.Fatal("the first instance polled with a lease the second now holds")
	}
}

// The worker stops when its context ends, and frees the lease.
func TestPollerRunStops(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 4201)

	f.Enqueue(fakes.TextUpdate(9_000_000+nextUpdate(), 4201, "/help"))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() { done <- bot.Run(ctx) }()

	deadline := time.After(5 * time.Second)

	for len(f.Sent(4201)) < 2 {
		select {
		case <-deadline:
			t.Fatal("the worker did not handle the update")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the worker did not stop")
	}

	if len(f.Calls("deleteWebhook")) == 0 {
		t.Error("polling did not remove the webhook first")
	}

	db, _ := app.DB()

	st, err := orm.QueryOf[models.TelegramState](db).Where("name", "poller").First(t.Context())
	if err != nil || st.LeaseOwner != "" {
		t.Fatalf("the lease was not freed: %+v %v", st, err)
	}
}
