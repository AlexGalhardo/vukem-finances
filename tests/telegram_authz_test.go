package tests

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

// asMember is the person of u working in the workspace of w.
func asMember(w, u who) who { return who{User: u.User, Workspace: w.Workspace, Client: u.Client} }

// A person removed from the workspace loses the bot at once: the next message,
// and a tap on a draft made before the removal, are refused.
func TestSecurity_RemovedMemberLosesBot(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	join(t, app, ana, bob.User, models.RoleMember)

	bobIn := asMember(ana, bob)
	linkChat(t, bot, bobIn, 7001)

	say(t, bot, 7001, "/balance")
	mustContain(t, lastText(t, f, 7001), "Main account")

	// A draft made while Bob is a member.
	say(t, bot, 7001, "-5 coffee")
	confirm := buttonData(t, f, 7001, "No category")

	removeMember(t, app, ana, bob.User.ID)

	for _, text := range []string{"/balance", "/month", "/last", "/budgets", "/account", "-5 coffee", "+100 gift"} {
		f.Reset()
		say(t, bot, 7001, text)
		mustContain(t, lastText(t, f, 7001), "no longer have access")
	}

	f.Reset()
	tap(t, bot, 7001, confirm)

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "no longer have access") {
		t.Fatalf("the tap on an old draft was not refused: %v", got)
	}

	if n := len(transactionsOf(t, app, ana)); n != 0 {
		t.Fatalf("a removed member recorded %d transactions", n)
	}

	// /workspace still works, and only offers what Bob belongs to now.
	f.Reset()
	say(t, bot, 7001, "/workspace")

	call, _ := f.LastSent(7001)
	if got := fakes.Buttons(call); len(got) != 1 || got[0] != "ws:"+strconv.FormatInt(bob.Workspace.ID, 10) {
		t.Fatalf("workspace choices = %v, want only Bob's own", got)
	}

	// The old workspace cannot be chosen by forging the button either.
	f.Reset()
	tap(t, bot, 7001, "ws:"+strconv.FormatInt(ana.Workspace.ID, 10))

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "not a member") {
		t.Fatalf("a forged workspace tap was accepted: %v", got)
	}

	tap(t, bot, 7001, "ws:"+strconv.FormatInt(bob.Workspace.ID, 10))
	say(t, bot, 7001, "/balance")
	mustContain(t, lastText(t, f, 7001), "Main account")

	link, _, _ := bot.Store.LinkByChat(t.Context(), 7001)
	if link.CurrentWorkspaceID.V != bob.Workspace.ID {
		t.Fatalf("current workspace = %d", link.CurrentWorkspaceID.V)
	}
}

func containsText(c fakes.Call, part string) bool {
	text, _ := c.Params["text"].(string)

	return strings.Contains(text, part)
}

func stringContains(s, sub string) bool { return strings.Contains(s, sub) }

// A viewer reads but never records, and a member demoted to viewer loses
// writing at once, including the taps on drafts made before.
func TestViewerCannotRecord(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	vic := signUp(t, app, "Vic", "vic@example.com")
	join(t, app, ana, vic.User, models.RoleViewer)

	viewer := asMember(ana, vic)
	linkChat(t, bot, viewer, 7101)

	for _, text := range []string{"-5 coffee", "+100 gift", "12,50 lunch yesterday"} {
		say(t, bot, 7101, text)
		mustContain(t, lastText(t, f, 7101), "read only")
	}

	db, _ := app.DB()

	if n, _ := orm.QueryOf[models.TelegramPending](db).Count(t.Context()); n != 0 {
		t.Fatalf("a viewer made %d drafts", n)
	}

	// Reading works.
	for _, text := range []string{"/balance", "/month", "/last", "/budgets"} {
		f.Reset()
		say(t, bot, 7101, text)

		if got := lastText(t, f, 7101); stringContains(got, "read only") || stringContains(got, "no longer") {
			t.Errorf("%s refused to a viewer: %q", text, got)
		}
	}

	// A member who made a draft and is demoted before the tap.
	mia := signUp(t, app, "Mia", "mia@example.com")
	join(t, app, ana, mia.User, models.RoleMember)
	linkChat(t, bot, asMember(ana, mia), 7102)

	say(t, bot, 7102, "-7 snack")
	confirm := buttonData(t, f, 7102, "No category")

	if _, err := orm.QueryOf[models.Membership](db).Where("workspace_id", ana.Workspace.ID).Where("user_id", mia.User.ID).
		Update(t.Context(), orm.Set{"role": models.RoleViewer}); err != nil {
		t.Fatal(err)
	}

	f.Reset()
	tap(t, bot, 7102, confirm)

	if got := f.Calls("answerCallbackQuery"); len(got) != 1 || !containsText(got[0], "read only") {
		t.Fatalf("a demoted member's tap was not refused: %v", got)
	}

	if n := len(transactionsOf(t, app, ana)); n != 0 {
		t.Fatalf("%d transactions were recorded by viewers", n)
	}
}

// The bot has no delete of its own: the only one is the undo of the entry a
// chat just saved, named by the reference of that chat's draft.
func TestBotCannotDeleteOthers(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	eve := signUp(t, app, "Eve", "eve@example.com")
	linkChat(t, bot, ana, 7201)
	linkChat(t, bot, eve, 7202)

	// A transaction Ana made in the web app.
	acct := firstAccount(t, app, ana)
	addTransaction(t, ana, acct, 0, "expense", "20.00", today())

	web := transactionsOf(t, app, ana)
	if len(web) != 1 {
		t.Fatalf("%d web transactions", len(web))
	}

	// One Ana made with the bot.
	say(t, bot, 7201, "-9 bus")
	tap(t, bot, 7201, buttonData(t, f, 7201, "No category"))

	saved, _ := f.LastSent(7201)
	undo := fakes.Buttons(saved)
	if len(undo) != 1 {
		t.Fatalf("undo buttons = %v", undo)
	}

	if len(transactionsOf(t, app, ana)) != 2 {
		t.Fatal("the bot entry was not saved")
	}

	// Eve taps Ana's undo, forges one with a transaction uid, an id, and a mix.
	for _, data := range []string{
		undo[0], "eu:" + web[0].UID, "eu:" + strconv.FormatInt(web[0].ID, 10), "eu:" + web[0].UID + ":" + web[0].UID,
		"eu:", "eu", "del:" + web[0].UID, "ex:" + web[0].UID, "eu:../../" + web[0].UID,
	} {
		tap(t, bot, 7202, data)
	}

	// Ana taps forged undos too.
	for _, data := range []string{"eu:" + web[0].UID, "eu:" + strconv.FormatInt(web[0].ID, 10), "del:" + web[0].UID} {
		tap(t, bot, 7201, data)
	}

	if n := len(transactionsOf(t, app, ana)); n != 2 {
		t.Fatalf("%d live transactions, want both still there", n)
	}

	// A message cannot delete anything either: there is no such command.
	for _, text := range []string{"/delete " + web[0].UID, "/del 1", "/remove", "delete everything", "/undo"} {
		say(t, bot, 7201, text)
	}

	if n := len(transactionsOf(t, app, ana)); n != 2 {
		t.Fatalf("a message deleted a transaction: %d left", n)
	}

	// Ana's own undo does work, once.
	tap(t, bot, 7201, undo[0])

	left := transactionsOf(t, app, ana)
	if len(left) != 1 || left[0].UID != web[0].UID {
		t.Fatalf("after the undo: %+v, want only the web transaction", left)
	}
}

func TestBotScopesAreTheFixedSet(t *testing.T) {
	t.Parallel()

	want := []string{
		"accounts:read", "categories:read", "transactions:read", "transactions:write", "budgets:read", "summary:read",
	}

	got := telegram.Scopes()
	if !slices.Equal(got, want) {
		t.Fatalf("bot scopes = %v, want %v", got, want)
	}

	for _, s := range got {
		if !api.ValidScope(s) || api.AbilityOf(s) == "" {
			t.Errorf("%s is not a scope of the API ceiling", s)
		}
	}
}

// The web has no endpoint that sets a chat id or lists another person's links.
func TestNoRouteTakesAChatID(t *testing.T) {
	t.Parallel()

	app, _, _ := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	for _, path := range []string{"/telegram/link", "/telegram", "/settings/telegram/chat"} {
		res := w.Client.PostForm(path, url.Values{"chat_id": {"1"}, "chatId": {"1"}})
		if res.Status() < 400 {
			t.Errorf("POST %s answered %d", path, res.Status())
		}
	}
}
