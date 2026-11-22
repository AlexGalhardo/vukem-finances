package tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// TestCardUpdateHTTP edits a card through its form: a viewer changes nothing.
func TestCardUpdateHTTP(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	vera := viewerOf(t, app, ana)
	bank := idString(firstAccount(t, app, ana))

	form := func(limit string) url.Values {
		return url.Values{
			"name": {"Visa"}, "limit": {limit}, "closingDay": {"10"}, "dueDay": {"25"}, "brand": {"visa"}, "last4": {"4242"},
			"paymentAccountId": {bank},
		}
	}

	ana.Client.PostForm("/cards", form("2000.00")).AssertRedirect("/cards")

	card := firstOf[models.CreditCard](t, app, ana, "")
	base := "/cards/" + idString(card.ID)

	patchForm(vera.Client, base, form("1.00")).AssertForbidden()
	patchForm(ana.Client, base, form("3500.00")).AssertRedirect(base)

	if n := rowCountWhere(t, app, "credit_cards", "limit_cents = 350000"); n != 1 {
		t.Fatalf("cards with the new limit = %d, want 1", n)
	}
}

// TestGoalLifecycleHTTP edits, archives, restores and trims the contributions
// of a goal through its forms, as each role.
func TestGoalLifecycleHTTP(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	ana, _, member, viewer := allRoles(t, app)
	asMember := browser(t, app, member.User, ana.Workspace.ID)
	asViewer := browser(t, app, viewer.User, ana.Workspace.ID)

	ana.Client.PostForm("/goals", url.Values{"name": {"Trip"}, "target": {"1200.00"}}).AssertRedirect("/goals")

	gid := idString(firstOf[models.Goal](t, app, ana, "").ID)
	base := "/goals/" + gid

	ana.Client.PostForm(base+"/contributions", url.Values{"amount": {"50.00"}}).AssertRedirect("/goals")

	contribution := idString(firstOf[models.GoalContribution](t, app, ana, "").ID)

	// A member records and trims contributions (the recording ability) but does not manage goals.
	for _, w := range []*vukemtest.Client{asMember, asViewer} {
		patchForm(w, base, url.Values{"name": {"Hijack"}, "target": {"1.00"}}).AssertStatus(http.StatusForbidden)
		w.PostForm(base+"/archive", nil).AssertStatus(http.StatusForbidden)
		w.PostForm(base+"/unarchive", nil).AssertStatus(http.StatusForbidden)
	}

	asViewer.Delete(base + "/contributions/" + contribution).AssertStatus(http.StatusForbidden)

	patchForm(ana.Client, base, url.Values{"name": {"Big trip"}, "target": {"2400.00"}}).AssertRedirect("/goals")

	if n := rowCountWhere(t, app, "goals", "name = 'Big trip' AND target_cents = 240000"); n != 1 {
		t.Fatalf("the edited goal is not saved (%d rows)", n)
	}

	ana.Client.PostForm(base+"/archive", nil).AssertRedirect("/goals")

	if n := rowCountWhere(t, app, "goals", "archived_at IS NOT NULL"); n != 1 {
		t.Fatalf("archived goals = %d, want 1", n)
	}

	ana.Client.PostForm(base+"/unarchive", nil).AssertRedirect("/goals")

	if n := rowCountWhere(t, app, "goals", "archived_at IS NOT NULL"); n != 0 {
		t.Fatalf("archived goals after unarchive = %d, want 0", n)
	}

	ana.Client.Delete(base + "/contributions/" + contribution).AssertRedirect("/goals")

	if n := rowCountWhere(t, app, "goal_contributions", "deleted_at IS NULL"); n != 0 {
		t.Fatalf("live contributions = %d, want 0", n)
	}
}
