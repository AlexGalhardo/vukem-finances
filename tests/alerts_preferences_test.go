package tests

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func mailCount(app *vukemtest.App, to, subjectPart string) int {
	n := 0

	for _, m := range app.Mails().SentTo(to) {
		if strings.Contains(m.Subject, subjectPart) {
			n++
		}
	}

	return n
}

// allRoles is a workspace with one person of every role.
func allRoles(t *testing.T, app *vukemtest.App) (owner, admin, member, viewer who) {
	t.Helper()

	owner = signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, owner)

	for role, email := range map[string]string{
		models.RoleAdmin: "adm@example.com", models.RoleMember: "mem@example.com", models.RoleViewer: "vie@example.com",
	} {
		u := newVerifiedUser(t, app, role, email)
		join(t, app, owner, u, role)

		w := who{User: u, Workspace: owner.Workspace}
		switch role {
		case models.RoleAdmin:
			admin = w
		case models.RoleMember:
			member = w
		default:
			viewer = w
		}
	}

	return owner, admin, member, viewer
}

func TestPreferences(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	owner, admin, member, viewer := allRoles(t, app)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, owner)

	rows, err := svc.Preferences(ctx, owner.User.ID, models.RoleOwner)
	if err != nil || len(rows) != len(alerts.Kinds()) {
		t.Fatalf("preferences = %+v (%v)", rows, err)
	}

	for _, r := range rows {
		if !r.InApp || !r.Email {
			t.Fatalf("an owner's default for %s = %+v, want in-app and email", r.Kind, r)
		}
	}

	// A member defaults to in-app only; a viewer can never turn mail on.
	if rows, _ := svc.Preferences(ctx, member.User.ID, models.RoleMember); rows[0].Email || !rows[0].InApp {
		t.Fatalf("a member's default = %+v", rows[0])
	}

	if err := svc.SetPreferences(ctx, viewer.User.ID, models.RoleViewer, []alerts.PreferenceInput{
		{Kind: alerts.KindBudget80, InApp: true, Email: true},
	}); err != nil {
		t.Fatal(err)
	}

	if rows, _ := svc.Preferences(ctx, viewer.User.ID, models.RoleViewer); rows[0].Email || rows[0].EmailAllowed {
		t.Fatalf("a viewer's mail choice = %+v", rows[0])
	}

	if err := svc.SetPreferences(ctx, owner.User.ID, models.RoleOwner, []alerts.PreferenceInput{{Kind: "nope"}}); err == nil {
		t.Error("an unknown kind was saved")
	}

	// The admin opts out of mail for this kind; the owner opts out of everything.
	if err := svc.SetPreferences(ctx, admin.User.ID, models.RoleAdmin, []alerts.PreferenceInput{
		{Kind: alerts.KindBudget80, InApp: true, Email: false},
	}); err != nil {
		t.Fatal(err)
	}

	for range 2 { // saving twice edits the row
		if err := svc.SetPreferences(ctx, owner.User.ID, models.RoleOwner, []alerts.PreferenceInput{
			{Kind: alerts.KindBudget80, InApp: false, Email: false},
		}); err != nil {
			t.Fatal(err)
		}
	}

	acc := firstAccount(t, app, owner)
	food := categoryID(t, app, owner, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, owner, acc, food, "85.00", "2027-03-02")

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	for who, want := range map[string]int{"ana@example.com": 0, "adm@example.com": 0, "mem@example.com": 0, "vie@example.com": 0} {
		if n := mailCount(app, who, "budget at 80%"); n != want {
			t.Errorf("mails to %s = %d, want %d", who, n, want)
		}
	}

	// The owner opted out of both channels: no row at all. The others have theirs.
	count := func(w who) int {
		n, err := svc.Unread(ctx, w.User.ID)
		if err != nil {
			t.Fatal(err)
		}

		return int(n)
	}

	if count(owner) != 0 || count(admin) != 1 || count(member) != 1 || count(viewer) != 1 {
		t.Fatalf("unread: owner %d admin %d member %d viewer %d, want 0, 1, 1, 1",
			count(owner), count(admin), count(member), count(viewer))
	}

	if len(notificationsOf(t, app, owner, alerts.KindBudget80)) != 3 {
		t.Fatal("the opted-out owner got a notification row")
	}
}

func TestChannels(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	owner, admin, member, viewer := allRoles(t, app)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, owner)
	acc := firstAccount(t, app, owner)
	food := categoryID(t, app, owner, "Groceries")

	if err := fin.SetBudget(ctx, food, "100.00"); err != nil {
		t.Fatal(err)
	}

	spend(t, fin, owner, acc, food, "120.00", "2027-03-02")

	g, err := fin.CreateGoal(ctx, finance.GoalInput{Name: "Trip", Target: "50.00"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "50.00"}); err != nil {
		t.Fatal(err)
	}

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "100.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	spend(t, fin, owner, card.AccountID, food, "90.00", "2027-03-06")

	if _, err := fin.CreateRecurrence(ctx, monthlyRule(acc, "2027-03-02")); err != nil {
		t.Fatal(err)
	}

	if _, err := mustDB(t, app).SQL().ExecContext(t.Context(), "UPDATE recurrences SET last_error = 'quota'"); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	for _, subject := range []string{
		"Groceries budget at 100%", "Goal reached: Trip", "Visa is near its limit", "A recurring entry could not be posted",
	} {
		for _, who := range []string{"ana@example.com", "adm@example.com"} {
			if n := mailCount(app, who, subject); n != 1 {
				t.Errorf("mails %q to %s = %d, want 1", subject, who, n)
			}
		}

		for _, who := range []string{"mem@example.com", "vie@example.com"} {
			if n := mailCount(app, who, subject); n != 0 {
				t.Errorf("mails %q to %s = %d, want none", subject, who, n)
			}
		}
	}

	// Everybody has the in-app list, and the viewer's rows are theirs alone.
	for _, w := range []who{owner, admin, member, viewer} {
		list, err := svc.List(ctx, w.User.ID, 50)
		if err != nil || len(list) < 4 {
			t.Fatalf("list of %s = %d rows (%v)", w.User.Email, len(list), err)
		}

		for _, n := range list {
			if n.UserID != w.User.ID {
				t.Fatalf("%s sees a row of user %d", w.User.Email, n.UserID)
			}
		}
	}
}

// A mail still owed (its send failed, or the run stopped) follows the choice
// the person holds when it is sent, not the one held when the row was made.
func TestPendingMailFollowsCurrentPreference(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	owner, admin, _, _ := allRoles(t, app)

	svc := alertsService(t, app)
	ctx := wsCtx(t, owner)
	db := mustDB(t, app)

	for _, w := range []who{owner, admin} {
		n := models.Notification{
			UserID: w.User.ID, Kind: alerts.KindBudget100, SubjectKey: "pending", TitleKey: alerts.TitleKey(alerts.KindBudget100),
			Params: `{"name":"Groceries","threshold":100,"spent_cents":12000,"limit_cents":10000}`, InApp: true, Email: true,
		}
		if err := orm.Create(ctx, db, &n); err != nil {
			t.Fatal(err)
		}
	}

	if err := svc.SetPreferences(ctx, owner.User.ID, models.RoleOwner, []alerts.PreferenceInput{
		{Kind: alerts.KindBudget100, InApp: true, Email: false},
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if n := mailCount(app, "ana@example.com", "budget at 100%"); n != 0 {
		t.Errorf("mails after opting out = %d, want none", n)
	}

	if n := mailCount(app, "adm@example.com", "budget at 100%"); n != 1 {
		t.Errorf("mails to the person who kept mail on = %d, want 1", n)
	}

	// Turning mail back on does not send the alert that was dropped.
	if err := svc.SetPreferences(ctx, owner.User.ID, models.RoleOwner, []alerts.PreferenceInput{
		{Kind: alerts.KindBudget100, InApp: true, Email: true},
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if n := mailCount(app, "ana@example.com", "budget at 100%"); n != 0 {
		t.Errorf("mails after opting back in = %d, want none", n)
	}
}

// A name is typed by any member and read in the mail of the others.
func TestAlertMailEscapesNames(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-18")
	owner, _, _, _ := allRoles(t, app)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, owner)

	const name = `<img src=x onerror=alert(1)>`

	g, err := fin.CreateGoal(ctx, finance.GoalInput{Name: name, Target: "50.00"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, g.ID, finance.ContributionInput{Amount: "50.00"}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	sent := 0

	for _, m := range app.Mails().SentTo("ana@example.com") {
		if !strings.Contains(m.Subject, "Goal reached") {
			continue
		}

		sent++

		if strings.Contains(m.HTML, "<img src=x") {
			t.Errorf("the goal name reached the mail unescaped")
		}
	}

	if sent != 1 {
		t.Fatalf("goal mails = %d, want 1", sent)
	}
}

func TestStatementDueMail(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-03-17")
	owner := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, owner)

	svc := alertsService(t, app)
	fin := softService(t, app)
	ctx := wsCtx(t, owner)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "1000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	spend(t, fin, owner, card.AccountID, categoryID(t, app, owner, "Groceries"), "42.00", "2027-03-06")

	if err := svc.Run(ctx); err != nil {
		t.Fatal(err)
	}

	sent := app.Mails().SentTo("ana@example.com")

	for _, m := range sent {
		if strings.Contains(m.Subject, "statement is due 2027-03-20") {
			if !strings.Contains(m.HTML, "$42.00") {
				t.Errorf("the mail does not show the amount:\n%s", m.HTML)
			}

			return
		}
	}

	t.Fatalf("no statement mail among %d mails", len(sent))
}
