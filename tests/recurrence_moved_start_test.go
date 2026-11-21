package tests

import "testing"

// Backfill is asked for at creation. Editing a future rule so that it starts in
// the past posts nothing that was already due.
func TestMovingAFutureRuleIntoThePastPostsNothing(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	in := monthlyRule(firstAccount(t, app, ana), "2027-05-01")

	rule, err := fin.CreateRecurrence(ctx, in)
	if err != nil {
		t.Fatal(err)
	}

	in.StartDate, in.Active = "2025-01-01", true

	got, err := fin.UpdateRecurrence(ctx, rule.ID, in)
	if err != nil {
		t.Fatal(err)
	}

	if dates := occurrenceDates(t, app, ana, rule.UID); len(dates) != 0 {
		t.Errorf("moving the start back posted %v", dates)
	}

	if got.GeneratedThrough.V != "2027-03-30" {
		t.Errorf("watermark = %q, want 2027-03-30", got.GeneratedThrough.V)
	}
}
