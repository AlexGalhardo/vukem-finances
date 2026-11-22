package tests

import (
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func linksOf(t *testing.T, app *vukemtest.App) []models.TelegramLink {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows, err := orm.QueryOf[models.TelegramLink](db).OrderBy("id", orm.Asc).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func TestLinkFlow(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	say(t, bot, 501, "/start")
	mustContain(t, lastText(t, f, 501), "Settings, Telegram", "not end-to-end encrypted")

	linkChat(t, bot, w, 501)
	mustContain(t, lastText(t, f, 501), "Connected", w.Workspace.Name, "not end-to-end encrypted")

	links := linksOf(t, app)
	if len(links) != 1 || links[0].UserID != w.User.ID || links[0].ChatID != 501 || links[0].CurrentWorkspaceID.V != w.Workspace.ID {
		t.Fatalf("links = %+v", links)
	}

	say(t, bot, 501, "/start")
	mustContain(t, lastText(t, f, 501), "You are connected to")
}

// A code works once. The second use, from the same chat or another one, is refused.
func TestSecurity_LinkCodeSingleUse(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	code, _, err := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	if err != nil {
		t.Fatal(err)
	}

	say(t, bot, 601, "/start "+code)
	mustContain(t, lastText(t, f, 601), "Connected")

	say(t, bot, 602, "/start "+code)
	mustContain(t, lastText(t, f, 602), "invalid, expired or already used")

	if n := len(linksOf(t, app)); n != 1 {
		t.Fatalf("%d links, want 1", n)
	}

	// A new code spends the earlier unused ones: a person holds one live code.
	first, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	second, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)

	say(t, bot, 603, "/start "+first)
	mustContain(t, lastText(t, f, 603), "invalid, expired or already used")

	say(t, bot, 603, "/start "+second)
	mustContain(t, lastText(t, f, 603), "Connected")
}

func TestSecurity_LinkCodeExpires(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app, f, bot := tgAppAt(t, clock)
	w := signUp(t, app, "Ana", "ana@example.com")

	code, _, err := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	if err != nil {
		t.Fatal(err)
	}

	clock.Advance(10*time.Minute + time.Second)

	say(t, bot, 701, "/start "+code)
	mustContain(t, lastText(t, f, 701), "invalid, expired or already used")

	if n := len(linksOf(t, app)); n != 0 {
		t.Fatalf("an expired code linked a chat: %d links", n)
	}
}

// 50 chats race for one code: exactly one links.
func TestSecurity_LinkConcurrent(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	code, _, err := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			chat := int64(10_000 + i)
			// Errors from the handler are not the point: the database decides who links.
			_ = bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.TextUpdate(nextUpdate(), chat, "/start "+code)))
		}()
	}

	wg.Wait()

	if n := len(linksOf(t, app)); n != 1 {
		t.Fatalf("%d links from one code, want exactly 1", n)
	}
}

// The same chat repeating the same code races too: still one link, one row.
func TestSecurity_LinkConcurrentSameChat(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	code, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.TextUpdate(nextUpdate(), 801, "/start "+code)))
		}()
	}

	wg.Wait()

	if n := len(linksOf(t, app)); n != 1 {
		t.Fatalf("%d links, want 1", n)
	}
}

// There is no way to link by naming an account: an id, an email or a password
// in /start is only a bad code, and no web route takes a chat id.
func TestSecurity_NoLinkByAccountID(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	for _, attempt := range []string{
		strconv.FormatInt(w.User.ID, 10), "user_" + strconv.FormatInt(w.User.ID, 10),
		"ana@example.com", "ana", "password123456789012", "AAAAAAAAAAAAAAAAAAAAAA", "a b c",
		"' OR 1=1 --", strconv.FormatInt(w.Workspace.ID, 10),
	} {
		say(t, bot, 901, "/start "+attempt)
	}

	if n := len(linksOf(t, app)); n != 0 {
		t.Fatalf("linked without a code: %d links", n)
	}

	for _, text := range f.Sent(901) {
		if text == "" {
			t.Fatal("an empty reply")
		}
	}

	// Plain messages of an unlinked chat get no data, only the invitation to connect.
	say(t, bot, 901, "/balance")
	mustContain(t, lastText(t, f, 901), "not connected")

	// The only write path of a link is the single-use code: the web page offers
	// no chat id field and no route accepts one.
	for _, path := range []string{"/settings/telegram/link", "/telegram/link", "/settings/telegram/chat"} {
		res := w.Client.PostForm(path, url.Values{"chatId": {"901"}, "chat_id": {"901"}, "userId": {strconv.FormatInt(w.User.ID, 10)}})
		if res.Status() == 200 || res.Status() == 302 || res.Status() == 303 {
			t.Errorf("POST %s answered %d", path, res.Status())
		}
	}

	if n := len(linksOf(t, app)); n != 0 {
		t.Fatalf("a form linked a chat: %d links", n)
	}
}

// Group, supergroup and channel chats get no reply and no data, even with a
// valid code or from a linked person.
func TestSecurity_GroupChatIgnored(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 1001)

	code, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)

	for _, kind := range []string{"group", "supergroup", "channel"} {
		for _, text := range []string{"/start " + code, "/balance", "-10 coffee", "/help"} {
			up := fakes.ChatUpdate(nextUpdate(), -2002, kind, text)
			if err := bot.HandleUpdate(t.Context(), mustUpdate(t, up)); err != nil {
				t.Fatal(err)
			}

			// The same message from the linked person's id, in the group.
			up = fakes.ChatUpdate(nextUpdate(), 1001, kind, text)
			up["message"].(map[string]any)["chat"].(map[string]any)["id"] = int64(-3003)

			if err := bot.HandleUpdate(t.Context(), mustUpdate(t, up)); err != nil {
				t.Fatal(err)
			}
		}
	}

	if got := f.Sent(-2002); len(got) != 0 {
		t.Fatalf("a group chat got replies: %v", got)
	}

	if got := f.Sent(-3003); len(got) != 0 {
		t.Fatalf("a group chat got replies: %v", got)
	}

	if n := len(linksOf(t, app)); n != 1 {
		t.Fatalf("a group linked: %d links", n)
	}

	if n := len(transactionsOf(t, app, w)); n != 0 {
		t.Fatalf("a group recorded %d transactions", n)
	}
}

func TestUnlink(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 1101)

	say(t, bot, 1101, "/unlink")
	mustContain(t, lastText(t, f, 1101), "Disconnected", "delete the chat history")

	if _, ok, _ := bot.Store.LinkByChat(t.Context(), 1101); ok {
		t.Fatal("the chat is still linked")
	}

	say(t, bot, 1101, "-10 coffee")
	mustContain(t, lastText(t, f, 1101), "not connected")

	if n := len(transactionsOf(t, app, w)); n != 0 {
		t.Fatalf("an unlinked chat recorded %d transactions", n)
	}

	// The chat can link again with a new code, and the row is reused.
	linkChat(t, bot, w, 1101)

	if n := len(linksOf(t, app)); n != 1 {
		t.Fatalf("%d link rows, want the revoked one reused", n)
	}

	// Another person cannot take a chat that is linked now.
	bob := signUp(t, app, "Bob", "bob@example.com")
	code, _, _ := bot.Store.NewCode(t.Context(), bob.User.ID, bob.Workspace.ID)

	say(t, bot, 1101, "/start "+code)
	mustContain(t, lastText(t, f, 1101), "already connected")

	link, _, _ := bot.Store.LinkByChat(t.Context(), 1101)
	if link.UserID != w.User.ID {
		t.Fatal("a second person took the chat")
	}
}

// Revoking from the web is limited to the person's own links.
func TestRevokeOnlyOwnLinks(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	linkChat(t, bot, ana, 1201)

	link, _, _ := bot.Store.LinkByChat(t.Context(), 1201)

	if ok, err := bot.Store.RevokeOf(t.Context(), bob.User.ID, link.ID); err != nil || ok {
		t.Fatalf("Bob revoked Ana's link: %v %v", ok, err)
	}

	if ok, err := bot.Store.RevokeOf(t.Context(), ana.User.ID, link.ID); err != nil || !ok {
		t.Fatalf("Ana could not revoke her own link: %v %v", ok, err)
	}
}
