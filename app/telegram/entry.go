package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// draft is the payload of a pending entry. Money is int64 cents, read by
// finance.ParseAmount when the message came in.
type draft struct {
	Kind        string  `json:"kind"`
	Cents       int64   `json:"cents"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
	AccountID   int64   `json:"account_id"`
	Categories  []int64 `json:"categories"`
}

// centsText writes cents as the "12.50" that finance.ParseAmount reads back.
func centsText(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func kindWord(lg, kind string) string {
	if kind == models.KindIncome {
		return lang.T(lg, lang.Income)
	}

	return lang.T(lg, lang.Expense)
}

// defaultAccount is the account an entry goes to: the one the person chose
// with /account while it is still there, else the first live account that is
// not a credit card.
func (b *Bot) defaultAccount(ac access, link models.TelegramLink) (models.Account, bool, error) {
	accounts, err := b.liveAccounts(ac)
	if err != nil || len(accounts) == 0 {
		return models.Account{}, false, err
	}

	if link.DefaultAccountUID.Valid {
		for _, a := range accounts {
			if a.UID == link.DefaultAccountUID.V {
				return a, true, nil
			}
		}
	}

	for _, a := range accounts {
		if a.Kind != models.AccountCreditCard {
			return a, true, nil
		}
	}

	return accounts[0], true, nil
}

// categoryChoices lists up to five categories of kind: the most recently used
// first, then the others by name.
func (b *Bot) categoryChoices(ac access, kind string) ([]models.Category, error) {
	active, err := b.Fin.ActiveCategories(ac.ctx)
	if err != nil {
		return nil, err
	}

	byID := map[int64]models.Category{}

	for _, c := range active {
		if c.Kind == kind {
			byID[c.ID] = c
		}
	}

	recent, err := workspaces.Scoped[models.Transaction](ac.ctx, b.Store.DB).Where("kind", kind).
		WhereNotNull("category_id").OrderBy("id", orm.Desc).Limit(200).Get(ac.ctx)
	if err != nil {
		return nil, err
	}

	var out []models.Category

	taken := map[int64]bool{}

	for _, t := range recent {
		if c, ok := byID[t.CategoryID.V]; ok && !taken[c.ID] && len(out) < 5 {
			out, taken[c.ID] = append(out, c), true
		}
	}

	for _, c := range active {
		if c.Kind == kind && !taken[c.ID] && len(out) < 5 {
			out, taken[c.ID] = append(out, c), true
		}
	}

	return out, nil
}

// onEntry reads a free-text message as an entry and asks for confirmation.
func (b *Bot) onEntry(ac access, link models.TelegramLink, text string) error {
	lg := link.Language

	if !ac.can(api.ScopeTransactionsWrite) {
		b.refuse(ac, link)

		return nil
	}

	e, err := ParseEntry(text, b.today(ac))

	switch {
	case errors.Is(err, ErrTooLong):
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.TooLong), nil)

		return nil
	case errors.Is(err, ErrBadAmount):
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.ParseAmount), nil)

		return nil
	case errors.Is(err, ErrBadDate):
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.ParseDate), nil)

		return nil
	case err != nil:
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.ParseFail), nil)

		return nil
	}

	acct, ok, err := b.defaultAccount(ac, link)
	if err != nil {
		return err
	}

	if !ok {
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.NoAccount), nil)

		return nil
	}

	cats, err := b.categoryChoices(ac, e.Kind)
	if err != nil {
		return err
	}

	d := draft{Kind: e.Kind, Cents: e.Cents, Description: e.Description, Date: e.Date, AccountID: acct.ID}
	for _, c := range cats {
		d.Categories = append(d.Categories, c.ID)
	}

	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}

	ref := models.NewUID()
	if err := b.Store.NewPending(ac.ctx, link.ChatID, link.UserID, ac.ws.ID, ref, string(payload)); err != nil {
		return err
	}

	var kb Keyboard

	for i, c := range cats {
		kb = append(kb, []Button{{Text: short(c.Name, 40), CallbackData: data(cbConfirm, ref, strconv.Itoa(i+1))}})
	}

	kb = append(kb, []Button{
		{Text: lang.T(lg, lang.NoCategory), CallbackData: data(cbConfirm, ref, "0")},
		{Text: lang.T(lg, lang.Cancel), CallbackData: data(cbCancel, ref)},
	})

	b.send(ac.ctx, link.ChatID, confirmText(lg, e.Kind, e.Cents, ac.ws.Currency, e.Description, e.Date, acct.Name), kb)

	return nil
}

// draftAccess finds the draft of a tap and resolves the access of its
// workspace now. A draft of another chat or of a person who no longer owns the
// chat is as good as unknown. notice is what to answer when ok is false.
func (b *Bot) draftAccess(ctx context.Context, link models.TelegramLink, ref string) (p models.TelegramPending, ac access, notice string, ok bool, err error) {
	lg := link.Language

	p, err = orm.QueryOf[models.TelegramPending](b.Store.DB).Where("ref", ref).Where("chat_id", link.ChatID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) || (err == nil && p.UserID != link.UserID) {
		return p, ac, lang.T(lg, lang.Expired), false, nil
	}

	if err != nil {
		return p, ac, "", false, err
	}

	ac, member, err := b.accessTo(ctx, link.UserID, sql.Null[int64]{V: p.ForWorkspaceID, Valid: true})
	if err != nil {
		return p, ac, "", false, err
	}

	if !member {
		return p, ac, lang.T(lg, lang.NoAccess), false, nil
	}

	if !ac.can(api.ScopeTransactionsWrite) {
		return p, ac, lang.T(lg, lang.ReadOnly), false, nil
	}

	return p, ac, "", true, nil
}

// confirmEntry is the tap on a category (or on "No category"): it consumes the
// draft atomically, then saves the transaction through the normal service
// path with the draft reference as its uid, so a double tap makes one row.
func (b *Bot) confirmEntry(ctx context.Context, link models.TelegramLink, q CallbackQuery, ref, choice string) (string, error) {
	lg := link.Language

	p, ac, notice, ok, err := b.draftAccess(ctx, link, ref)
	if err != nil || !ok {
		return notice, err
	}

	if _, claimed, err := b.Store.Consume(ctx, link.ChatID, ref); err != nil || !claimed {
		return lang.T(lg, lang.Expired), err
	}

	var d draft
	if err := json.Unmarshal([]byte(p.Payload), &d); err != nil {
		return lang.T(lg, lang.Failed), err
	}

	var category int64

	n, convErr := strconv.Atoi(choice)

	switch {
	case convErr != nil || n < 0 || n > len(d.Categories):
		_ = b.Store.Unconsume(ctx, ref)

		return lang.T(lg, lang.Expired), nil
	case n > 0:
		category = d.Categories[n-1]
	}

	tx, err := b.saveDraft(ac, d, category, ref)
	if err != nil {
		_ = b.Store.Unconsume(ctx, ref)

		b.send(ctx, link.ChatID, userFacing(lg, err), nil)

		if isRefusal(err) {
			return "", nil
		}

		return "", err
	}

	if err := b.Store.MarkCreated(ctx, ref, tx.UID); err != nil {
		return "", err
	}

	b.dropKeyboard(ctx, q)

	acct, err := workspaces.Scoped[models.Account](ac.ctx, b.Store.DB).Where("id", tx.AccountID).First(ac.ctx)
	if err != nil {
		return "", err
	}

	b.send(ctx, link.ChatID, savedText(lg, tx.Kind, tx.AmountCents, ac.ws.Currency, tx.Description, tx.OccurredOn, acct.Name),
		Keyboard{{{Text: lang.T(lg, lang.Undo), CallbackData: data(cbUndo, ref)}}})

	return "", nil
}

// saveDraft creates the transaction. A transaction that already has the uid
// (the tap was repeated after a failure to record it) is returned as it is.
func (b *Bot) saveDraft(ac access, d draft, category int64, uid string) (models.Transaction, error) {
	existing, err := workspaces.Scoped[models.Transaction](ac.ctx, b.Store.DB).Where("uid", uid).First(ac.ctx)
	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, orm.ErrNotFound) {
		return models.Transaction{}, err
	}

	return b.Fin.CreateTransaction(ac.ctx, finance.TxInput{
		AccountID: d.AccountID, CategoryID: category, Kind: d.Kind, Amount: centsText(d.Cents),
		Description: d.Description, Date: d.Date, UID: uid,
	})
}

// cancelEntry spends a draft without saving anything.
func (b *Bot) cancelEntry(ctx context.Context, link models.TelegramLink, q CallbackQuery, ref string) (string, error) {
	lg := link.Language

	_, claimed, err := b.Store.Consume(ctx, link.ChatID, ref)
	if err != nil {
		return "", err
	}

	b.dropKeyboard(ctx, q)

	if !claimed {
		return lang.T(lg, lang.Expired), nil
	}

	return lang.T(lg, lang.Cancelled), nil
}

// undoEntry soft-deletes the transaction a tap just created, inside the 60
// second window. It is the only delete the bot has, and it can only name the
// transaction through the reference of a draft this chat consumed.
func (b *Bot) undoEntry(ctx context.Context, link models.TelegramLink, q CallbackQuery, ref string) (string, error) {
	lg := link.Language

	_, ac, notice, ok, err := b.draftAccess(ctx, link, ref)
	if err != nil || !ok {
		return notice, err
	}

	p, claimed, late, err := b.Store.ClaimUndo(ctx, link.ChatID, ref)
	if err != nil {
		return "", err
	}

	if !claimed {
		if late {
			b.dropKeyboard(ctx, q)

			return lang.T(lg, lang.UndoLate), nil
		}

		return lang.T(lg, lang.Expired), nil
	}

	row, err := workspaces.Scoped[models.Transaction](ac.ctx, b.Store.DB).Where("uid", p.TransactionUID.V).First(ac.ctx)
	if err == nil {
		err = b.Fin.DeleteTransaction(ac.ctx, row.ID)
	}

	if err != nil && !errors.Is(err, orm.ErrNotFound) {
		_ = b.Store.ReleaseUndo(ctx, ref)

		return lang.T(lg, lang.Failed), err
	}

	b.dropKeyboard(ctx, q)

	return lang.T(lg, lang.Undone), nil
}

func (b *Bot) dropKeyboard(ctx context.Context, q CallbackQuery) {
	if q.Message == nil {
		return
	}

	if err := b.Client.EditMessageReplyMarkup(ctx, q.Message.Chat.ID, q.Message.MessageID); err != nil {
		b.Log.Warn("telegram: edit failed", "err", err.Error())
	}
}
