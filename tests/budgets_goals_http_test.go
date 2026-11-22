package tests

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestBudgetsHTTP(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana, _, member, viewer := allRoles(t, app)

	food := strconv.FormatInt(categoryID(t, app, ana, "Groceries"), 10)

	// No category is the overall budget, and it may roll over.
	ana.Client.PostForm("/budgets", url.Values{"limit": {"500.00"}, "rollover": {"true"}}).AssertRedirect("/budgets")

	// A bad category must never turn into the overall budget.
	ana.Client.PostForm("/budgets", url.Values{"categoryId": {"abc"}, "limit": {"1.00"}}).AssertRedirect("/budgets")
	ana.Client.PostForm("/budgets", url.Values{"categoryId": {"-4"}, "limit": {"1.00"}}).AssertRedirect("/budgets")
	ana.Client.PostForm("/budgets", url.Values{"categoryId": {food}, "limit": {"100.00"}}).AssertRedirect("/budgets")

	if n := rowCountWhere(t, app, "budgets", "category_id IS NULL"); n != 1 {
		t.Fatalf("overall budgets = %d, want 1", n)
	}

	if rowCountWhere(t, app, "budgets", "category_id IS NULL AND limit_cents = 50000 AND rollover = 1") != 1 {
		b := firstOf[models.Budget](t, app, ana, "")
		t.Fatalf("the overall budget did not keep its limit and rollover: %+v", b)
	}

	page := ana.Client.Inertia("").Get("/budgets").AssertInertia("Budgets")
	page.PropEquals("budgets.0.overall", true)
	page.PropEquals("budgets.1.categoryId", float64(categoryID(t, app, ana, "Groceries")))

	// A month that is not a month is ignored; a real one is honoured.
	ana.Client.Inertia("").Get("/budgets?month=0000-01").AssertInertia("Budgets").PropEquals("month", "2027-03")
	ana.Client.Inertia("").Get("/budgets?month=2027-1").AssertInertia("Budgets").PropEquals("month", "2027-03")
	ana.Client.Inertia("").Get("/budgets?month=2027-01").AssertInertia("Budgets").PropEquals("month", "2027-01")

	for _, w := range []who{member, viewer} {
		browser(t, app, w.User, ana.Workspace.ID).
			PostForm("/budgets", url.Values{"limit": {"1.00"}}).AssertStatus(http.StatusForbidden)
	}
}

func TestGoalsHTTP(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana, _, member, viewer := allRoles(t, app)
	asMember := browser(t, app, member.User, ana.Workspace.ID)
	asViewer := browser(t, app, viewer.User, ana.Workspace.ID)

	ana.Client.PostForm("/goals", url.Values{"name": {"Trip"}, "target": {"1200.00"}, "targetDate": {"2027-06-30"}}).AssertRedirect("/goals")
	ana.Client.PostForm("/goals", url.Values{"name": {"Bad"}, "target": {"10.00"}, "targetDate": {"2027-02-30"}}).AssertRedirect("/goals")

	if n := rowCount(t, app, "goals"); n != 1 {
		t.Fatalf("goals = %d, want 1 (a bad date is refused)", n)
	}

	gid := idString(firstOf[models.Goal](t, app, ana, "").ID)

	// A member records contributions but does not manage goals; a viewer does neither.
	asMember.PostForm("/goals/"+gid+"/contributions", url.Values{"amount": {"200.00"}}).AssertRedirect("/goals")
	asMember.PostForm("/goals", url.Values{"name": {"X"}, "target": {"1.00"}}).AssertStatus(http.StatusForbidden)
	asMember.Delete("/goals/" + gid).AssertStatus(http.StatusForbidden)
	asViewer.PostForm("/goals/"+gid+"/contributions", url.Values{"amount": {"1.00"}}).AssertStatus(http.StatusForbidden)

	page := ana.Client.Inertia("").Get("/goals").AssertInertia("Goals")
	page.PropEquals("goals.0.savedCents", float64(20000))
	page.PropEquals("goals.0.percent", float64(16))
	page.PropEquals("goals.0.monthsLeft", float64(3)) // 2027-03-18 to 2027-06-30: Jun 18 fits, Jul 18 does not
	page.PropEquals("contributions.0.amount_cents", float64(20000))

	ana.Client.Delete("/goals/" + gid).AssertRedirect("/goals")

	if n := rowCount(t, app, "goals"); n != 1 || rowCountWhere(t, app, "goals", "deleted_at IS NOT NULL") != 1 {
		t.Fatal("deleting a goal must soft delete it")
	}
}

func TestNotificationsHTTP(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana, admin, member, viewer := allRoles(t, app)
	asAdmin := browser(t, app, admin.User, ana.Workspace.ID)
	asMember := browser(t, app, member.User, ana.Workspace.ID)
	asViewer := browser(t, app, viewer.User, ana.Workspace.ID)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	g, err := fin.CreateGoal(ctx, finance.GoalInput{Name: "Trip", Target: "10.00"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "10.00"}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	// The bell feeds from a shared prop.
	asMember.Inertia("").Get("/goals").AssertInertia("Goals").PropEquals("notifications.unread", float64(1))

	mine := notificationsOf(t, app, ana, alerts.KindGoalReached)
	if len(mine) != 4 {
		t.Fatalf("notifications = %d, want one per member", len(mine))
	}

	var ofAdmin models.Notification

	for _, n := range mine {
		if n.UserID == admin.User.ID {
			ofAdmin = n
		}
	}

	// Another member cannot read it, even in the same workspace: 404, not 403.
	asMember.PostForm("/notifications/"+idString(ofAdmin.ID)+"/read", nil).AssertNotFound()
	asViewer.PostForm("/notifications/"+idString(ofAdmin.ID)+"/read", nil).AssertNotFound()

	if rowCountWhere(t, app, "notifications", "id = "+idString(ofAdmin.ID)+" AND read_at IS NOT NULL") != 0 {
		t.Fatal("a notification was read by someone else")
	}

	asAdmin.PostForm("/notifications/"+idString(ofAdmin.ID)+"/read", nil).AssertRedirect("/notifications")
	asAdmin.PostForm("/notifications/"+idString(ofAdmin.ID)+"/read", nil).AssertRedirect("/notifications")
	asAdmin.Inertia("").Get("/notifications").AssertInertia("Notifications").
		PropEquals("notifications.0.read", true).PropEquals("preferences.0.emailAllowed", true)

	// Read all only touches the caller's rows.
	asMember.PostForm("/notifications/read", nil).AssertRedirect("/notifications")

	if rowCountWhere(t, app, "notifications", "read_at IS NOT NULL") != 2 {
		t.Fatal("mark all read touched rows of another person")
	}

	// Preferences are the caller's own, a bad kind saves nothing, and a viewer's mail stays off.
	putForm(asViewer, "/notifications/preferences", url.Values{"inApp": {"goal_reached"}, "email": {"goal_reached"}}).
		AssertRedirect("/notifications")
	putForm(asViewer, "/notifications/preferences", url.Values{"inApp": {"nope"}}).AssertRedirect("/notifications")

	if rowCountWhere(t, app, "notification_preferences", "user_id = "+idString(viewer.User.ID)+" AND kind = 'goal_reached' AND email = 1") != 0 {
		t.Fatal("a viewer turned mail on")
	}

	if rowCountWhere(t, app, "notification_preferences", "user_id != "+idString(viewer.User.ID)) != 0 {
		t.Fatal("a preference was saved for someone else")
	}

	// Another workspace sees nothing of it.
	bob := signUp(t, app, "Bob", "bob@example.com")
	bob.Client.Inertia("").Get("/notifications").AssertInertia("Notifications").PropEquals("preferences.0.kind", "budget_80")
	bob.Client.PostForm("/notifications/"+idString(ofAdmin.ID)+"/read", nil).AssertNotFound()
}
