package tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func TestTelegramSettingsPage(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	web := func() interface {
	} {
		return nil
	}
	_ = web

	page := func() map[string]any {
		res := browser(t, app, ana.User, ana.Workspace.ID).Inertia("").Get("/settings/telegram").AssertOK()

		var out map[string]any
		res.JSON(&out)

		return out["props"].(map[string]any)
	}

	props := page()
	if props["username"] != "fake_finance_bot" || len(props["links"].([]any)) != 0 || props["created"] != nil || props["alertsOn"] != false {
		t.Fatalf("props = %v", props)
	}

	// Connect shows the deep link and the code once, and says not to cache it.
	res := browser(t, app, ana.User, ana.Workspace.ID).Inertia("").PostForm("/settings/telegram/connect", url.Values{}).AssertOK()
	res.AssertHeaderContains("Cache-Control", "no-store")

	link, ok := res.JSONPath("props.created.link")
	if !ok {
		t.Fatalf("no link in the answer: %s", res.Body())
	}

	code, _ := res.JSONPath("props.created.code")
	if !strings.HasPrefix(link.(string), "https://t.me/fake_finance_bot?start=") || !strings.HasSuffix(link.(string), code.(string)) {
		t.Fatalf("link = %v code = %v", link, code)
	}

	if len(code.(string)) != 22 {
		t.Fatalf("code %q is not 128 bits in base64url (22 characters)", code)
	}

	if secs, _ := res.JSONPath("props.created.seconds"); asFloat(secs) < 590 || asFloat(secs) > 600 {
		t.Fatalf("the code lives %v seconds, want about 600", secs)
	}

	// Reloading never shows it again.
	if again := page(); again["created"] != nil {
		t.Fatalf("the code is shown again: %v", again["created"])
	}

	// The code links through the bot, once.
	say(t, bot, 9701234, "/start "+code.(string))
	mustContain(t, lastText(t, f, 9701234), "Connected")

	props = page()

	links := props["links"].([]any)
	if len(links) != 1 {
		t.Fatalf("links = %v", links)
	}

	one := links[0].(map[string]any)
	if one["chatHint"] != "1234" {
		t.Fatalf("the page shows %v of the chat id, want only the last 4 digits", one["chatHint"])
	}

	// The full chat id, the token and the secrets are nowhere in the page.
	body := browser(t, app, ana.User, ana.Workspace.ID).Inertia("").Get("/settings/telegram").Body()
	for _, s := range []string{"9701234", fakes.TelegramToken, "fake-token-for-tests-only", tgWebhookSecret, bot.Cfg.WebhookPathSecret()} {
		if strings.Contains(body, s) {
			t.Errorf("the page leaks %q", s)
		}
	}

	// The alert switch.
	form := url.Values{"enabled": {"true"}}
	putForm(browser(t, app, ana.User, ana.Workspace.ID), "/settings/telegram/alerts", form).AssertStatus(http.StatusSeeOther)

	if props = page(); props["alertsOn"] != true {
		t.Fatalf("alertsOn = %v", props["alertsOn"])
	}

	putForm(browser(t, app, ana.User, ana.Workspace.ID), "/settings/telegram/alerts", url.Values{"enabled": {"maybe"}}).
		AssertStatus(http.StatusUnprocessableEntity)

	putForm(browser(t, app, ana.User, ana.Workspace.ID), "/settings/telegram/alerts", url.Values{"enabled": {"false"}}).AssertStatus(http.StatusSeeOther)

	if props = page(); props["alertsOn"] != false {
		t.Fatalf("alertsOn = %v", props["alertsOn"])
	}

	// Revoke from the web: the chat is cut off.
	id := int64(asFloat(one["id"]))
	browser(t, app, ana.User, ana.Workspace.ID).Delete("/settings/telegram/links/" + strconv.FormatInt(id, 10)).AssertStatus(http.StatusSeeOther)

	if _, ok, _ := bot.Store.LinkByChat(t.Context(), 9701234); ok {
		t.Fatal("the link survived the revoke")
	}

	say(t, bot, 9701234, "/balance")
	mustContain(t, lastText(t, f, 9701234), "not connected")
}

// A revoke names a link of the person or is a 404, and a person holds a few chats at most.
func TestTelegramSettingsAreOwnOnly(t *testing.T) {
	t.Parallel()

	app, _, bot := tgApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	linkChat(t, bot, ana, 9801234)

	link, _, _ := bot.Store.LinkByChat(t.Context(), 9801234)
	target := "/settings/telegram/links/" + strconv.FormatInt(link.ID, 10)

	browser(t, app, bob.User, bob.Workspace.ID).Delete(target).AssertNotFound()

	if _, ok, _ := bot.Store.LinkByChat(t.Context(), 9801234); !ok {
		t.Fatal("Bob revoked Ana's chat")
	}

	browser(t, app, ana.User, ana.Workspace.ID).Delete("/settings/telegram/links/999999").AssertNotFound()

	// The sixth chat is refused until one is disconnected.
	for i := range 4 {
		linkChat(t, bot, ana, int64(9802000+i))
	}

	res := browser(t, app, ana.User, ana.Workspace.ID).Inertia("").PostForm("/settings/telegram/connect", url.Values{})
	if res.Status() == http.StatusOK {
		if _, ok := res.JSONPath("props.created.link"); ok {
			t.Fatalf("a sixth chat was offered a code: %s", res.Body())
		}
	}

}

// Signed out, the page and its actions are not there; with the bot off they are 404 for everyone.
func TestTelegramSettingsNeedASessionAndTheBot(t *testing.T) {
	t.Parallel()

	app, _, _ := tgApp(t)

	for _, path := range []string{"/settings/telegram"} {
		res := app.Client(t).Get(path)
		if res.Status() == http.StatusOK {
			t.Errorf("GET %s answered 200 to a signed-out visitor", path)
		}
	}

	off := newApp(t)
	w := signUp(t, off, "Ana", "ana@example.com")

	browser(t, off, w.User, w.Workspace.ID).Get("/settings/telegram").AssertNotFound()
	browser(t, off, w.User, w.Workspace.ID).PostForm("/settings/telegram/connect", url.Values{}).AssertNotFound()
	putForm(browser(t, off, w.User, w.Workspace.ID), "/settings/telegram/alerts", url.Values{"enabled": {"true"}}).AssertNotFound()
	browser(t, off, w.User, w.Workspace.ID).Delete("/settings/telegram/links/1").AssertNotFound()
}

// asFloat reads a number of a decoded JSON answer, whatever decoder made it.
func asFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()

		return f
	default:
		return -1
	}
}
