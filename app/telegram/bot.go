package telegram

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Scopes is the fixed scope set of the bot (docs/telegram.md): the same names
// and the same role ceiling as an API token (epic 14). The bot holds no scope
// that can change members, billing, settings, tokens or AI keys, and none that
// deletes: the only delete it can make is the undo of the entry it just saved.
func Scopes() []string {
	return []string{
		api.ScopeAccountsRead, api.ScopeCategoriesRead, api.ScopeTransactionsRead,
		api.ScopeTransactionsWrite, api.ScopeBudgetsRead, api.ScopeSummaryRead,
	}
}

// Bot handles the updates of the Telegram bot.
type Bot struct {
	Cfg    config.Telegram
	Store  *Store
	WS     *workspaces.Service
	Fin    *finance.Service
	Client *Client
	Limits *Limits
	Log    *slog.Logger
	Now    func() time.Time
	// Owner names this instance in the polling lease.
	Owner string

	// afterUpdate runs after each update of a poll batch (tests stop a poll
	// mid-batch with it).
	afterUpdate func(updateID int64)
}

// New builds the bot from the app. It returns nil, nil while TELEGRAM_BOT_TOKEN
// is unset: the bot is off, no route is registered and no worker starts.
func New(app *vukem.App, ws *workspaces.Service, fin *finance.Service) (*Bot, error) {
	cfg, err := config.LoadTelegram(app.Source, ws.Edition)
	if err != nil {
		return nil, err
	}

	if !cfg.Enabled() {
		return nil, nil //nolint:nilnil // nil means the bot is off
	}

	// TELEGRAM_API_BASE points the client at a fake server: the tests and the
	// browser specs use it. Production ignores it, so the token only ever goes
	// to Telegram.
	client := NewClient(cfg.Token)
	if cfg.APIBase != "" && (app.Kind == vukem.KindTest || !app.Config.IsProduction()) {
		client = NewTestClient(cfg.Token, cfg.APIBase)
	}

	// The window limiter counts with the real clock, so the counters expire by it too.
	limits := NewDBLimits(ws.DB)

	return &Bot{
		Cfg: cfg, Store: &Store{DB: ws.DB, Now: app.Now}, WS: ws, Fin: fin, Client: client,
		Limits: limits, Log: app.Log, Now: app.Now, Owner: NewOwner(),
	}, nil
}

// HandleUpdate handles one update exactly once: it begins by claiming the
// update id, and a redelivery of the same id does nothing. An error is an
// internal failure (the database): the claim is released so the redelivery is
// handled, and nothing the person sent is logged.
func (b *Bot) HandleUpdate(ctx context.Context, u Update) error {
	if u.UpdateID <= 0 {
		return nil
	}

	first, err := b.Store.ClaimUpdate(ctx, u.UpdateID)
	if err != nil {
		return err
	}

	if !first {
		return nil
	}

	var herr error

	switch {
	case u.Message != nil:
		herr = b.onMessage(ctx, *u.Message)
	case u.CallbackQuery != nil:
		herr = b.onCallback(ctx, *u.CallbackQuery)
	}

	if herr != nil {
		_ = b.Store.ReleaseUpdate(context.WithoutCancel(ctx), u.UpdateID)
		b.Log.Warn("telegram: update not handled", "update_id", u.UpdateID, "err", herr.Error())
	}

	return herr
}

// send delivers a message. A failure (a blocked bot, Telegram down) is logged
// without the text and never fails the update: the action it answers is done.
func (b *Bot) send(ctx context.Context, chat int64, text string, kb Keyboard) {
	if err := b.Client.SendMessage(ctx, chat, text, kb); err != nil {
		b.Log.Warn("telegram: send failed", "err", err.Error())
	}
}

func (b *Bot) answer(ctx context.Context, id, notice string) {
	if err := b.Client.AnswerCallbackQuery(ctx, id, notice); err != nil {
		b.Log.Warn("telegram: answer failed", "err", err.Error())
	}
}

// guessLanguage maps Telegram's language_code to a language of the bot.
func guessLanguage(code string) string {
	switch {
	case strings.HasPrefix(code, "pt"):
		return lang.PT
	case strings.HasPrefix(code, "es"):
		return lang.ES
	default:
		return lang.EN
	}
}

// access is what a person may do now in one workspace: it is read from the
// membership at every message, so a removed member or a demoted one loses
// the bot at once.
type access struct {
	ctx  context.Context
	ws   models.Workspace
	role string
}

// can reports whether the person holds scope: the scope must be one of the
// bot, and the role of the membership now must hold the ability behind it.
func (a access) can(scope string) bool {
	return slices.Contains(Scopes(), scope) && workspaces.RoleCan(a.role, api.AbilityOf(scope))
}

// accessTo resolves the workspace and the role of userID. ok is false when the
// workspace is unset or deleted, or the person is no longer a member.
func (b *Bot) accessTo(ctx context.Context, userID int64, workspaceID sql.Null[int64]) (access, bool, error) {
	if !workspaceID.Valid {
		return access{}, false, nil
	}

	role, member, err := b.WS.MemberRole(ctx, userID, workspaceID.V)
	if err != nil || !member {
		return access{}, false, err
	}

	c := workspaces.WithRole(workspaces.WithTenant(ctx, workspaceID.V), role)

	ws, err := b.WS.Current(c)
	if errors.Is(err, orm.ErrNotFound) {
		return access{}, false, nil
	}

	if err != nil {
		return access{}, false, err
	}

	return access{ctx: c, ws: ws, role: role}, true, nil
}

var codeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

// splitCommand reads "/cmd@botname rest". cmd is "" for a message that is not a command.
func splitCommand(text string) (cmd, arg string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}

	word, rest, _ := strings.Cut(text, " ")
	word, _, _ = strings.Cut(word, "@")

	return strings.ToLower(word), strings.TrimSpace(rest)
}

func (b *Bot) onMessage(ctx context.Context, m Message) error {
	// Only a person in a private chat is served; every other chat type gets
	// no reply and no data.
	if m.Chat.Type != "private" || m.From == nil || m.From.IsBot || m.From.ID != m.Chat.ID || m.Text == "" {
		return nil
	}

	chat := m.Chat.ID
	guess := guessLanguage(m.From.LanguageCode)

	switch b.Limits.Message(ctx, chat) {
	case Drop:
		return nil
	case Notify:
		b.send(ctx, chat, lang.T(guess, lang.RateNotice), nil)

		return nil
	}

	cmd, arg := splitCommand(m.Text)

	if cmd == "/start" {
		return b.onStart(ctx, m, arg, guess)
	}

	link, linked, err := b.Store.LinkByChat(ctx, chat)
	if err != nil {
		return err
	}

	if !linked {
		b.send(ctx, chat, lang.T(guess, lang.NotLinked), nil)

		return nil
	}

	b.Store.Touch(ctx, link.ID)

	lg := link.Language

	switch cmd {
	case "/help":
		b.send(ctx, chat, lang.T(lg, lang.Help), nil)

		return nil
	case "/unlink":
		if err := b.Store.Revoke(ctx, link.ID); err != nil {
			return err
		}

		b.send(ctx, chat, lang.T(lg, lang.Unlinked), nil)

		return nil
	case "/lang":
		return b.askLanguage(ctx, chat, lg)
	case "/workspace":
		return b.askWorkspace(ctx, link)
	}

	ac, ok, err := b.accessTo(ctx, link.UserID, link.CurrentWorkspaceID)
	if err != nil {
		return err
	}

	if !ok {
		b.send(ctx, chat, lang.T(lg, lang.NoAccess), nil)

		return nil
	}

	switch cmd {
	case "":
		return b.onEntry(ac, link, m.Text)
	case "/balance":
		return b.cmdBalance(ac, link)
	case "/month":
		return b.cmdMonth(ac, link)
	case "/last":
		return b.cmdLast(ac, link)
	case "/budgets":
		return b.cmdBudgets(ac, link)
	case "/account":
		return b.askAccount(ac, link)
	default:
		b.send(ctx, chat, lang.T(lg, lang.Help), nil)

		return nil
	}
}

func (b *Bot) onStart(ctx context.Context, m Message, arg, guess string) error {
	chat := m.Chat.ID

	link, linked, err := b.Store.LinkByChat(ctx, chat)
	if err != nil {
		return err
	}

	lg := guess
	if linked {
		lg = link.Language
	}

	if arg == "" {
		if !linked {
			b.send(ctx, chat, lang.T(lg, lang.Welcome, lang.T(lg, lang.Privacy)), nil)

			return nil
		}

		ac, ok, err := b.accessTo(ctx, link.UserID, link.CurrentWorkspaceID)
		if err != nil {
			return err
		}

		if !ok {
			b.send(ctx, chat, lang.T(lg, lang.NoAccess), nil)

			return nil
		}

		b.send(ctx, chat, lang.T(lg, lang.WelcomeLinked, EscapeHTML(ac.ws.Name)), nil)

		return nil
	}

	if !b.Limits.Start(ctx, chat) {
		b.send(ctx, chat, lang.T(lg, lang.LinkSlow), nil)

		return nil
	}

	if !codeRe.MatchString(arg) {
		b.send(ctx, chat, lang.T(lg, lang.LinkBad), nil)

		return nil
	}

	link, err = b.Store.Redeem(ctx, arg, chat, lg)

	switch {
	case errors.Is(err, ErrBadCode):
		b.send(ctx, chat, lang.T(lg, lang.LinkBad), nil)

		return nil
	case errors.Is(err, ErrChatTaken):
		b.send(ctx, chat, lang.T(lg, lang.LinkTaken), nil)

		return nil
	case err != nil:
		return err
	}

	ac, ok, err := b.accessTo(ctx, link.UserID, link.CurrentWorkspaceID)
	if err != nil {
		return err
	}

	name := ""
	if ok {
		name = ac.ws.Name
	}

	b.send(ctx, chat, lang.T(link.Language, lang.Linked, EscapeHTML(name), lang.T(link.Language, lang.Privacy)), nil)

	return nil
}

// isRefusal reports whether err is a refusal the person may read (a plan
// limit, an archived account) rather than a failure of the server.
func isRefusal(err error) bool {
	var he *errs.HTTPError

	return errors.As(err, &he) && he.Status >= http.StatusBadRequest && he.Status < http.StatusInternalServerError
}

// userFacing returns the text of a refusal and falls back to the generic failure.
func userFacing(lg string, err error) string {
	var he *errs.HTTPError
	if isRefusal(err) && errors.As(err, &he) {
		return EscapeHTML(he.Text())
	}

	return lang.T(lg, lang.Failed)
}

// short cuts s to n characters for a button label.
func short(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	return string([]rune(s)[:n-1]) + "…"
}
