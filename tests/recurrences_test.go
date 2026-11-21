package tests

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func monthlyRule(account int64, start string) finance.RecurrenceInput {
	return finance.RecurrenceInput{
		AccountID: account, Kind: models.KindExpense, Amount: "10.00", Description: "Rent", Frequency: finance.FreqMonthly,
		Interval: 1, StartDate: start, Backfill: true,
	}
}

func occurrenceDates(t *testing.T, app *vukemtest.App, w who, ruleUID string) []string {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	dates, err := workspaces.Scoped[models.Transaction](wsCtx(t, w), db).WithTrashed().Where("recurrence_uid", ruleUID).
		OrderBy("occurrence_on", orm.Asc).Pluck[string](t.Context(), "occurrence_on")
	if err != nil {
		t.Fatal(err)
	}

	return dates
}

func TestGenerateDue(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc := firstAccount(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	rule, err := fin.CreateRecurrence(ctx, monthlyRule(acc, "2027-01-31"))
	if err != nil {
		t.Fatal(err)
	}

	got := occurrenceDates(t, app, ana, rule.UID)
	want := []string{"2027-01-31", "2027-02-28", "2027-03-31"}

	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("dates = %v, want %v", got, want)
	}

	if rule.GeneratedThrough.V != "2027-03-31" || rule.LastError.Valid {
		t.Errorf("rule = %+v, want watermark 2027-03-31 and no error", rule)
	}
}

func TestGenerateWithoutBackfillStartsToday(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)

	in := monthlyRule(firstAccount(t, app, ana), "2027-01-31")
	in.Backfill = false

	rule, err := fin.CreateRecurrence(wsCtx(t, ana), in)
	if err != nil {
		t.Fatal(err)
	}

	if got := occurrenceDates(t, app, ana, rule.UID); len(got) != 1 || got[0] != "2027-03-31" {
		t.Errorf("dates = %v, want only today", got)
	}
}

func TestBackfillOverThirtySixIsRefused(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-31")
	ana := signUp(t, app, "Ana", "ana@example.com")

	in := monthlyRule(firstAccount(t, app, ana), "2023-01-31")
	if _, err := softService(t, app).CreateRecurrence(wsCtx(t, ana), in); err == nil {
		t.Fatal("a 51-month backfill was accepted")
	}

	vukemtest.AssertDatabaseCount(t, app, "recurrences", 0)
}

func TestGenerateIsIdempotent(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	if _, err := fin.CreateRecurrence(ctx, monthlyRule(firstAccount(t, app, ana), "2027-01-31")); err != nil {
		t.Fatal(err)
	}

	clock.Set(day("2027-06-30"))

	var wg sync.WaitGroup

	for range 4 {
		wg.Go(func() {
			if _, err := fin.GenerateDue(ctx); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	n, err := fin.GenerateDue(ctx)
	if err != nil || n != 0 {
		t.Errorf("a further run made %d, %v; want 0", n, err)
	}

	// Jan 31, Feb 28, Mar 31, Apr 30, May 31, Jun 30, each once.
	vukemtest.AssertDatabaseCount(t, app, "transactions", 6)
}

func TestCatchUpAfterDowntime(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2026-01-15")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	rule, err := fin.CreateRecurrence(ctx, monthlyRule(firstAccount(t, app, ana), "2026-01-15"))
	if err != nil {
		t.Fatal(err)
	}

	// The server was off for 14 months.
	clock.Set(day("2027-03-15"))

	if n, err := fin.GenerateDue(ctx); err != nil || n != 14 {
		t.Fatalf("run made %d, %v; want 14", n, err)
	}

	if n, err := fin.GenerateDue(ctx); err != nil || n != 0 {
		t.Fatalf("second run made %d, %v; want 0", n, err)
	}

	if got := occurrenceDates(t, app, ana, rule.UID); len(got) != 15 || got[14] != "2027-03-15" {
		t.Errorf("dates = %v", got)
	}
}

// One run makes at most 500 occurrences per rule; the next run goes on.
func TestCatchUpIsCapped(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2026-01-16")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	in := monthlyRule(firstAccount(t, app, ana), "2026-01-16")
	in.Frequency = finance.FreqDaily

	rule, err := fin.CreateRecurrence(ctx, in)
	if err != nil {
		t.Fatal(err)
	}

	// 2026-01-17 to 2028-03-15 is 789 more days.
	clock.Set(day("2028-03-15"))

	for i, want := range []int{500, 289, 0} {
		if n, err := fin.GenerateDue(ctx); err != nil || n != want {
			t.Fatalf("run %d made %d, %v; want %d", i+1, n, err, want)
		}
	}

	if got := occurrenceDates(t, app, ana, rule.UID); len(got) != 790 {
		t.Errorf("dates = %d, want 790", len(got))
	}
}

func TestDeletedOccurrenceStaysDeleted(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	rule, err := fin.CreateRecurrence(ctx, monthlyRule(firstAccount(t, app, ana), "2027-01-10"))
	if err != nil {
		t.Fatal(err)
	}

	clock.Set(day("2027-04-10"))

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	feb := firstOfWhere[models.Transaction](t, app, ana, "occurrence_on", "2027-02-10")
	if err := fin.DeleteTransaction(ctx, feb.ID); err != nil {
		t.Fatal(err)
	}

	clock.Set(day("2027-06-10"))

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	assertLiveDates := func(want int64) {
		t.Helper()

		vukemtest.AssertDatabaseCount(t, app, "transactions", want)
	}

	// Jan, Mar, Apr, May, Jun are live; Feb stays in the trash.
	assertLiveDates(6)

	// The tombstone is purged; the watermark still keeps the occurrence out.
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := changes.Compact(t.Context(), db, clock.Now().Add(48*time.Hour), 0); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	got := occurrenceDates(t, app, ana, rule.UID)
	for _, d := range got {
		if d == "2027-02-10" {
			t.Fatalf("the deleted occurrence came back: %v", got)
		}
	}

	if len(got) != 5 {
		t.Errorf("dates = %v, want 5", got)
	}
}

func TestGenerateRuleFailureDoesNotBlockOthers(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-01")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	other, err := fin.CreateAccount(ctx, "Savings", models.AccountSavings, "")
	if err != nil {
		t.Fatal(err)
	}

	bad, err := fin.CreateRecurrence(ctx, monthlyRule(other.ID, "2027-01-15"))
	if err != nil {
		t.Fatal(err)
	}

	good, err := fin.CreateRecurrence(ctx, monthlyRule(firstAccount(t, app, ana), "2027-01-20"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.ArchiveAccount(ctx, other.ID); err != nil {
		t.Fatal(err)
	}

	clock.Set(day("2027-02-25"))

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	if got := occurrenceDates(t, app, ana, good.UID); len(got) != 2 {
		t.Errorf("the healthy rule made %v, want 2", got)
	}

	failed := firstOfWhere[models.Recurrence](t, app, ana, "uid", bad.UID)
	if !failed.LastError.Valid || len(occurrenceDates(t, app, ana, bad.UID)) != 0 {
		t.Errorf("failing rule = %+v, want a last_error and nothing generated", failed)
	}

	// Unarchived, the next run retries and clears the error.
	if _, err := fin.UnarchiveAccount(ctx, other.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	again := firstOfWhere[models.Recurrence](t, app, ana, "uid", bad.UID)
	if again.LastError.Valid || len(occurrenceDates(t, app, ana, bad.UID)) != 2 {
		t.Errorf("retried rule = %+v, want no error and 2 occurrences", again)
	}
}

func TestRecurrenceRespectsQuotaAtomically(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-01")
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc := firstAccount(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	var rules []models.Recurrence

	for _, desc := range []string{"A", "B"} {
		in := monthlyRule(acc, "2027-01-15")
		in.Description = desc

		r, err := fin.CreateRecurrence(ctx, in)
		if err != nil {
			t.Fatal(err)
		}

		rules = append(rules, r)
	}

	// The Free plan holds 100 a month: leave exactly one place.
	fillMonth(t, app, ana, acc, 99, "2027-01")
	clock.Set(day("2027-01-15"))

	var wg sync.WaitGroup

	for range 2 {
		wg.Go(func() { _, _ = fin.GenerateDue(ctx) })
	}

	wg.Wait()

	vukemtest.AssertDatabaseCount(t, app, "transactions", 100)

	made := len(occurrenceDates(t, app, ana, rules[0].UID)) + len(occurrenceDates(t, app, ana, rules[1].UID))
	if made != 1 {
		t.Errorf("rules generated %d, want exactly 1", made)
	}
}

func TestDeleteRecurrenceKeepsHistory(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc := firstAccount(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	rule, err := fin.CreateRecurrence(ctx, monthlyRule(acc, "2027-01-10"))
	if err != nil {
		t.Fatal(err)
	}

	// A future-dated generated row, as an edited rule leaves behind.
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	future := models.Transaction{
		AccountID: acc, Kind: models.KindExpense, AmountCents: 1000, Description: "Rent", OccurredOn: "2027-02-10",
		RecurrenceUID: sql.Null[string]{V: rule.UID, Valid: true}, OccurrenceOn: sql.Null[string]{V: "2027-02-10", Valid: true},
	}
	if err := orm.Create(ctx, db, &future); err != nil {
		t.Fatal(err)
	}

	if err := fin.DeleteRecurrence(ctx, rule.ID, true); err != nil {
		t.Fatal(err)
	}

	if live, err := workspaces.Scoped[models.Transaction](ctx, db).Count(ctx); err != nil || live != 1 {
		t.Fatalf("live transactions = %d, %v; want only the past one", live, err)
	}

	clock.Set(day("2027-05-10"))

	if n, err := fin.GenerateDue(ctx); err != nil || n != 0 {
		t.Errorf("a deleted rule generated %d, %v", n, err)
	}
}

func TestRuleFollowsAccountAndCategoryLifecycle(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	bank, err := fin.CreateAccount(ctx, "Bank", models.AccountChecking, "")
	if err != nil {
		t.Fatal(err)
	}

	from, err := fin.AddCategory(ctx, finance.CategoryInput{Name: "Old", Kind: models.KindExpense})
	if err != nil {
		t.Fatal(err)
	}

	to, err := fin.AddCategory(ctx, finance.CategoryInput{Name: "New", Kind: models.KindExpense})
	if err != nil {
		t.Fatal(err)
	}

	in := monthlyRule(bank.ID, "2027-01-10")
	in.CategoryID = from.ID

	rule, err := fin.CreateRecurrence(ctx, in)
	if err != nil {
		t.Fatal(err)
	}

	// A rule in the trash must follow its category to the target.
	if err := fin.DeleteRecurrence(ctx, rule.ID, false); err != nil {
		t.Fatal(err)
	}

	if err := fin.DeleteCategoryReassigning(ctx, from.ID, to.ID); err != nil {
		t.Fatal(err)
	}

	back, err := fin.RestoreRecurrence(ctx, rule.ID)
	if err != nil || !back.CategoryID.Valid || back.CategoryID.V != to.ID {
		t.Fatalf("restored = %+v, %v; want category %d", back.CategoryID, err, to.ID)
	}

	// Deleting the account takes its live rule along; restoring brings it back.
	if err := fin.DeleteAccount(ctx, bank.ID); err != nil {
		t.Fatal(err)
	}

	if rules, err := fin.Recurrences(ctx); err != nil || len(rules) != 0 {
		t.Fatalf("rules after the account was deleted = %d, %v", len(rules), err)
	}

	if _, err := fin.RestoreAccount(ctx, bank.ID); err != nil {
		t.Fatal(err)
	}

	if rules, err := fin.Recurrences(ctx); err != nil || len(rules) != 1 {
		t.Fatalf("rules after the account came back = %d, %v", len(rules), err)
	}
}

func TestUpcoming(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2027-01-10")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)

	rule, err := fin.CreateRecurrence(ctx, monthlyRule(acc, "2027-01-10"))
	if err != nil {
		t.Fatal(err)
	}

	// Feb 10 is a real future row of the rule (as after an edit): no virtual twin.
	db := mustDB(t, app)
	held := models.Transaction{
		AccountID: acc, Kind: models.KindExpense, AmountCents: 1000, Description: "Rent", OccurredOn: "2027-02-10",
		RecurrenceUID: sql.Null[string]{V: rule.UID, Valid: true}, OccurrenceOn: sql.Null[string]{V: "2027-02-10", Valid: true},
	}
	if err := orm.Create(ctx, db, &held); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
		AccountID: acc, Total: "30.00", Parts: 3, FirstDate: "2027-01-25", Description: "Chair",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := fin.Upcoming(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, it := range items {
		got = append(got, it.Date+" "+it.Source)
	}

	want := []string{"2027-01-25 installment", "2027-02-10 recurrence", "2027-02-25 installment"}
	if len(got) != len(want) {
		t.Fatalf("upcoming = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("upcoming = %v, want %v", got, want)
		}
	}

	// Once the day comes the occurrence is generated, and nothing shows twice.
	clock.Set(day("2027-02-12"))

	if _, err := fin.GenerateDue(ctx); err != nil {
		t.Fatal(err)
	}

	items, err = fin.Upcoming(ctx, 30)
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]int{}

	for _, it := range items {
		seen[it.Date]++
		if seen[it.Date] > 1 && it.Source == finance.SourceRecurrence {
			t.Errorf("duplicate upcoming on %s: %+v", it.Date, items)
		}
	}

	if seen["2027-03-10"] != 1 {
		t.Errorf("the March occurrence should show once as a virtual one: %+v", items)
	}
}
