package telegram

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	refRe = regexp.MustCompile(`^[0-9A-Za-z]{26}$`)
	uidRe = refRe
)

// maxCallbackData is Telegram's limit for the data of a button.
const maxCallbackData = 64

// onCallback handles a tap on an inline button. Like a message, it is served
// only in a private chat, only for a linked chat, and its authorization is
// resolved now: the data names a draft or a choice, never a workspace or a
// transaction the person could pick freely.
func (b *Bot) onCallback(ctx context.Context, q CallbackQuery) error {
	if q.Message == nil || q.Message.Chat.Type != "private" || q.From.IsBot || q.From.ID != q.Message.Chat.ID {
		return nil
	}

	chat := q.Message.Chat.ID

	if !b.Limits.Callback(ctx, chat) {
		return nil
	}

	link, linked, err := b.Store.LinkByChat(ctx, chat)
	if err != nil {
		return err
	}

	if !linked {
		b.answer(ctx, q.ID, lang.T(guessLanguage(q.From.LanguageCode), lang.NotLinked))

		return nil
	}

	if len(q.Data) > maxCallbackData {
		b.answer(ctx, q.ID, "")

		return nil
	}

	kind, rest, _ := strings.Cut(q.Data, ":")

	var notice string

	switch kind {
	case cbConfirm:
		ref, choice, _ := strings.Cut(rest, ":")
		if refRe.MatchString(ref) {
			notice, err = b.confirmEntry(ctx, link, q, ref, choice)
		}
	case cbCancel:
		if refRe.MatchString(rest) {
			notice, err = b.cancelEntry(ctx, link, q, rest)
		}
	case cbUndo:
		if refRe.MatchString(rest) {
			notice, err = b.undoEntry(ctx, link, q, rest)
		}
	case cbAccount:
		notice, err = b.chooseAccount(ctx, link, q, rest)
	case cbWorkspace:
		notice, err = b.chooseWorkspace(ctx, link, q, rest)
	case cbLanguage:
		notice, err = b.chooseLanguage(ctx, link, q, rest)
	}

	if err != nil {
		return err
	}

	b.answer(ctx, q.ID, notice)

	return nil
}

func (b *Bot) chooseAccount(ctx context.Context, link models.TelegramLink, q CallbackQuery, uid string) (string, error) {
	lg := link.Language

	ac, ok, err := b.accessTo(ctx, link.UserID, link.CurrentWorkspaceID)
	if err != nil {
		return "", err
	}

	if !ok {
		return lang.T(lg, lang.NoAccess), nil
	}

	if !ac.can(api.ScopeAccountsRead) || !uidRe.MatchString(uid) {
		return lang.T(lg, lang.Failed), nil
	}

	// The uid is looked up inside the workspace of the membership: an account
	// of another workspace is simply not found.
	acct, err := workspaces.Scoped[models.Account](ac.ctx, b.Store.DB).Where("uid", uid).WhereNull("archived_at").First(ac.ctx)
	if err != nil {
		if errors.Is(err, orm.ErrNotFound) {
			return lang.T(lg, lang.Failed), nil
		}

		return "", err
	}

	if err := b.Store.SetLinkFields(ctx, link.ID, orm.Set{"default_account_uid": acct.UID}); err != nil {
		return "", err
	}

	b.dropKeyboard(ctx, q)
	b.send(ctx, link.ChatID, lang.T(lg, lang.AccountSet, EscapeHTML(acct.Name)), nil)

	return "", nil
}

// chooseWorkspace switches the workspace of the chat. The membership is
// checked now; the chat keeps no role.
func (b *Bot) chooseWorkspace(ctx context.Context, link models.TelegramLink, q CallbackQuery, raw string) (string, error) {
	lg := link.Language

	id, perr := strconv.ParseInt(raw, 10, 64)
	if perr != nil || id <= 0 {
		return lang.T(lg, lang.Failed), nil
	}

	ac, ok, err := b.accessTo(ctx, link.UserID, sql.Null[int64]{V: id, Valid: true})
	if err != nil {
		return "", err
	}

	if !ok {
		return lang.T(lg, lang.WorkspaceNo), nil
	}

	if err := b.Store.SetLinkFields(ctx, link.ID, orm.Set{"current_workspace_id": id, "default_account_uid": nil}); err != nil {
		return "", err
	}

	b.dropKeyboard(ctx, q)
	b.send(ctx, link.ChatID, lang.T(lg, lang.WorkspaceSet, EscapeHTML(ac.ws.Name)), nil)

	return "", nil
}

func (b *Bot) chooseLanguage(ctx context.Context, link models.TelegramLink, q CallbackQuery, code string) (string, error) {
	if !lang.Valid(code) {
		return lang.T(link.Language, lang.Failed), nil
	}

	if err := b.Store.SetLinkFields(ctx, link.ID, orm.Set{"language": code}); err != nil {
		return "", err
	}

	b.dropKeyboard(ctx, q)
	b.send(ctx, link.ChatID, lang.T(code, lang.LangSet, EscapeHTML(lang.Name(code))), nil)

	return "", nil
}
