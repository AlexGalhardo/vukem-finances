package finance

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func date(s string) time.Time {
	d, err := time.Parse(isoDate, s)
	if err != nil {
		panic(err)
	}

	return d
}

func TestGoalRequiredMonthly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		today, target string
		remaining     int64
		cents         int64
		months        int
		ok            bool
	}{
		{"month end clamps", "2027-01-31", "2027-06-30", 120000, 24000, 5, true},
		{"leap february", "2028-01-31", "2028-02-29", 10000, 10000, 1, true},
		{"rounds up", "2027-01-10", "2027-04-10", 10000, 3334, 3, true},
		{"same day is one month", "2027-01-10", "2027-01-10", 5000, 5000, 1, true},
		{"less than a month is one month", "2027-01-10", "2027-02-01", 5000, 5000, 1, true},
		{"passed", "2027-03-01", "2027-02-01", 5000, 5000, 0, false},
	}

	for _, c := range cases {
		cents, months, ok := requiredMonthly(c.remaining, date(c.today), date(c.target))
		if cents != c.cents || months != c.months || ok != c.ok {
			t.Errorf("%s: got %d over %d months (%v), want %d over %d (%v)", c.name, cents, months, ok, c.cents, c.months, c.ok)
		}
	}
}

func goal(target int64, targetDate string) models.Goal {
	g := models.Goal{ID: 1, Name: "Trip", TargetCents: target}
	if targetDate != "" {
		g.TargetDate.V, g.TargetDate.Valid = targetDate, true
	}

	return g
}

func part(cents int64, on string) models.GoalContribution {
	return models.GoalContribution{AmountCents: cents, OnDate: on}
}

func TestGoalProgress(t *testing.T) {
	t.Parallel()

	today := date("2027-01-31")

	st := goalStatus(goal(120000, "2027-06-30"), []models.GoalContribution{part(20000, "2027-01-05")}, today)
	if st.SavedCents != 20000 || st.RemainingCents != 100000 || st.Percent != 16 || st.Reached {
		t.Fatalf("progress = %+v", st)
	}

	if st.RequiredMonthlyCents != 20000 || st.MonthsLeft != 5 {
		t.Fatalf("required = %d over %d months, want 20000 over 5", st.RequiredMonthlyCents, st.MonthsLeft)
	}

	done := goalStatus(goal(1000, ""), []models.GoalContribution{part(1500, "2027-01-05")}, today)
	if !done.Reached || done.RemainingCents != 0 || done.Percent != 150 {
		t.Fatalf("reached = %+v", done)
	}

	// A withdrawal never shows a negative percent.
	neg := goalStatus(goal(1000, ""), []models.GoalContribution{part(-500, "2027-01-05")}, today)
	if neg.Percent != 0 || neg.RemainingCents != 1500 {
		t.Fatalf("negative = %+v", neg)
	}

	late := goalStatus(goal(1000, "2027-01-01"), nil, today)
	if !late.Overdue || late.RequiredMonthlyCents != 0 {
		t.Fatalf("overdue = %+v", late)
	}
}

func TestGoalProgressSaturates(t *testing.T) {
	t.Parallel()

	var parts []models.GoalContribution
	for range 2000 {
		parts = append(parts, part(models.MaxCents, "2027-01-05"))
	}

	st := goalStatus(goal(models.MaxCents, ""), parts, date("2027-01-31"))
	if !st.Reached || st.Percent < 100 {
		t.Fatalf("saturated = %+v", st)
	}

}

func TestGoalProjection(t *testing.T) {
	t.Parallel()

	today := date("2027-04-15")
	g := goal(100000, "")

	// 300.00 in the three full months before April: 100.00 a month.
	parts := []models.GoalContribution{part(10000, "2027-01-10"), part(10000, "2027-02-10"), part(10000, "2027-03-10")}

	st := goalStatus(g, parts, today)
	if st.ProjectedDate != "2027-11-15" { // 70000 left at 10000 a month: 7 months
		t.Fatalf("projected = %q, want 2027-11-15", st.ProjectedDate)
	}

	// One contribution is not a pace.
	if one := goalStatus(g, parts[:1], today); one.ProjectedDate != "" {
		t.Fatalf("one contribution projected %q", one.ProjectedDate)
	}

	// The current month and older months are outside the window.
	old := []models.GoalContribution{part(10000, "2026-12-10"), part(10000, "2027-04-02")}
	if st := goalStatus(g, old, today); st.ProjectedDate != "" {
		t.Fatalf("outside the window projected %q", st.ProjectedDate)
	}
}
