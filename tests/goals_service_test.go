package tests

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

func TestGoalLifecycle(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-31")
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	g, err := fin.CreateGoal(ctx, finance.GoalInput{Name: " Trip ", Target: "1200.00", TargetDate: "2027-06-30"})
	if err != nil || g.Name != "Trip" || g.UID == "" {
		t.Fatalf("goal = %+v (%v)", g, err)
	}

	for _, bad := range []finance.GoalInput{
		{Name: "", Target: "10"},
		{Name: "x", Target: "0"},
		{Name: "x", Target: "10", TargetDate: "2027-02-30"},
		{Name: "x", Target: "10", TargetDate: "0001-01-01"},
		{Name: "x", Target: "10", TargetDate: "9999-01-01"},
		{Name: strings.Repeat("a", 121), Target: "10"},
	} {
		if _, err := fin.CreateGoal(ctx, bad); err == nil {
			t.Errorf("goal %+v was accepted", bad)
		}
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "200.00"}); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "0"}); err == nil {
		t.Error("a zero contribution was accepted")
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "5", Date: "2027-13-01"}); err == nil {
		t.Error("a bad date was accepted")
	}

	// Another workspace cannot touch the goal.
	if _, err := fin.AddContribution(wsCtx(t, bob), g.ID, finance.ContributionInput{Amount: "5"}); err == nil {
		t.Error("a foreign goal took a contribution")
	}

	list, err := fin.Goals(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("goals = %+v (%v)", list, err)
	}

	if list[0].SavedCents != 20000 || list[0].RequiredMonthlyCents != 20000 || list[0].MonthsLeft != 5 {
		t.Fatalf("status = %+v", list[0])
	}

	if err := fin.DeleteGoal(ctx, g.ID); err != nil {
		t.Fatal(err)
	}

	if list, _ := fin.Goals(ctx); len(list) != 0 {
		t.Fatalf("deleted goal listed: %+v", list)
	}

	if _, err := fin.RestoreGoal(ctx, g.ID); err != nil {
		t.Fatal(err)
	}

	list, _ = fin.Goals(ctx)
	if len(list) != 1 || list[0].SavedCents != 20000 {
		t.Fatalf("restored goal = %+v", list)
	}

	if _, err := fin.ArchiveGoal(ctx, g.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "5"}); err == nil {
		t.Error("an archived goal took a contribution")
	}
}
