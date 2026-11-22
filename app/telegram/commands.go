package telegram

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
)

// The callback data of the inline buttons. Telegram allows 64 bytes; the
// longest is a prefix, a 26 character reference and an index.
const (
	cbConfirm   = "ec" // ec:<ref>:<n>, n is 0 for no category or the 1-based choice
	cbCancel    = "ex" // ex:<ref>
	cbUndo      = "eu" // eu:<ref>
	cbAccount   = "ac" // ac:<account uid>
	cbWorkspace = "ws" // ws:<workspace id>
	cbLanguage  = "lg" // lg:<language code>
)

func data(parts ...string) string { return strings.Join(parts, ":") }

// refuse tells the person the role of the membership now does not allow it.
func (b *Bot) refuse(ac access, link models.TelegramLink) {
	b.send(ac.ctx, link.ChatID, lang.T(link.Language, lang.ReadOnly), nil)
}

func (b *Bot) cmdBalance(ac access, link models.TelegramLink) error {
	if !ac.can(api.ScopeAccountsRead) || !ac.can(api.ScopeSummaryRead) {
		b.refuse(ac, link)

		return nil
	}

	accounts, err := b.Fin.Accounts(ac.ctx)
	if err != nil {
		return err
	}

	balances, err := b.Fin.AccountBalances(ac.ctx, "")
	if err != nil {
		return err
	}

	byID := map[int64]int64{}
	for _, bal := range balances {
		byID[bal.AccountID] = bal.BalanceCents
	}

	net, err := b.Fin.NetWorth(ac.ctx, "")
	if err != nil {
		return err
	}

	lg := link.Language
	lines := []string{lang.T(lg, lang.BalanceTitle)}

	for _, a := range accounts {
		if a.ArchivedAt.Valid {
			continue
		}

		lines = append(lines, balanceLine(lg, a.Name, byID[a.ID], ac.ws.Currency))
	}

	lines = append(lines, "", lang.T(lg, lang.NetWorth, finance.FormatAmount(net, ac.ws.Currency)))
	b.send(ac.ctx, link.ChatID, strings.Join(lines, "\n"), nil)

	return nil
}

func (b *Bot) cmdMonth(ac access, link models.TelegramLink) error {
	if !ac.can(api.ScopeSummaryRead) {
		b.refuse(ac, link)

		return nil
	}

	month := b.Fin.Month(ac.ctx)

	d, err := b.Fin.Dashboard(ac.ctx, month)
	if err != nil {
		return err
	}

	var income, expense int64

	for _, p := range d.Months {
		if p.Month == month {
			income, expense = p.IncomeCents, p.ExpenseCents
		}
	}

	cur := ac.ws.Currency
	b.send(ac.ctx, link.ChatID, lang.T(link.Language, lang.MonthSummary, month,
		finance.FormatAmount(income, cur), finance.FormatAmount(expense, cur), finance.FormatAmount(income-expense, cur)), nil)

	return nil
}

func (b *Bot) cmdLast(ac access, link models.TelegramLink) error {
	if !ac.can(api.ScopeTransactionsRead) {
		b.refuse(ac, link)

		return nil
	}

	page, err := b.Fin.TransactionPage(ac.ctx, finance.TxFilter{Page: 1})
	if err != nil {
		return err
	}

	lg := link.Language
	if len(page.Rows) == 0 {
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.LastNone), nil)

		return nil
	}

	lines := []string{lang.T(lg, lang.LastTitle)}

	for _, r := range page.Rows[:min(5, len(page.Rows))] {
		lines = append(lines, lastLine(lg, r, ac.ws.Currency))
	}

	b.send(ac.ctx, link.ChatID, strings.Join(lines, "\n"), nil)

	return nil
}

// describe shows a dash for an empty description.
func describe(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}

	return s
}

func (b *Bot) cmdBudgets(ac access, link models.TelegramLink) error {
	if !ac.can(api.ScopeBudgetsRead) {
		b.refuse(ac, link)

		return nil
	}

	month := b.Fin.Month(ac.ctx)

	list, err := b.Fin.Budgets(ac.ctx, month)
	if err != nil {
		return err
	}

	lg := link.Language
	if len(list) == 0 {
		b.send(ac.ctx, link.ChatID, lang.T(lg, lang.BudgetsNone), nil)

		return nil
	}

	lines := []string{lang.T(lg, lang.BudgetsTitle, month)}

	for _, st := range list {
		lines = append(lines, budgetLine(lg, st, ac.ws.Currency))
	}

	b.send(ac.ctx, link.ChatID, strings.Join(lines, "\n"), nil)

	return nil
}

// liveAccounts lists the accounts that take entries, in the order of the app.
func (b *Bot) liveAccounts(ac access) ([]models.Account, error) {
	all, err := b.Fin.Accounts(ac.ctx)
	if err != nil {
		return nil, err
	}

	out := all[:0:0]

	for _, a := range all {
		if !a.ArchivedAt.Valid {
			out = append(out, a)
		}
	}

	return out, nil
}

func (b *Bot) askAccount(ac access, link models.TelegramLink) error {
	if !ac.can(api.ScopeAccountsRead) {
		b.refuse(ac, link)

		return nil
	}

	accounts, err := b.liveAccounts(ac)
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		b.send(ac.ctx, link.ChatID, lang.T(link.Language, lang.NoAccount), nil)

		return nil
	}

	var kb Keyboard
	for _, a := range accounts[:min(len(accounts), 20)] {
		kb = append(kb, []Button{{Text: short(a.Name, 40), CallbackData: data(cbAccount, a.UID)}})
	}

	b.send(ac.ctx, link.ChatID, lang.T(link.Language, lang.AccountChoose), kb)

	return nil
}

func (b *Bot) askWorkspace(ctx context.Context, link models.TelegramLink) error {
	u, err := orm.QueryOf[models.User](b.Store.DB).Find(ctx, link.UserID)
	if err != nil {
		return err
	}

	mine, err := b.WS.Mine(ctx, u)
	if err != nil {
		return err
	}

	if len(mine) == 0 {
		b.send(ctx, link.ChatID, lang.T(link.Language, lang.NoWorkspace), nil)

		return nil
	}

	var kb Keyboard
	for _, w := range mine[:min(len(mine), 20)] {
		kb = append(kb, []Button{{Text: short(w.Name, 40), CallbackData: data(cbWorkspace, strconv.FormatInt(w.ID, 10))}})
	}

	b.send(ctx, link.ChatID, lang.T(link.Language, lang.WorkspaceChoos), kb)

	return nil
}

func (b *Bot) askLanguage(ctx context.Context, chat int64, lg string) error {
	var kb Keyboard
	for _, code := range lang.Codes() {
		kb = append(kb, []Button{{Text: lang.Name(code), CallbackData: data(cbLanguage, code)}})
	}

	b.send(ctx, chat, lang.T(lg, lang.LangChoose), kb)

	return nil
}

// today is the current date of the workspace as a UTC midnight, for ParseEntry.
func (b *Bot) today(ac access) time.Time {
	d, err := time.Parse("2006-01-02", b.Fin.Today(ac.ctx))
	if err != nil {
		return b.Now().UTC().Truncate(24 * time.Hour)
	}

	return d
}
