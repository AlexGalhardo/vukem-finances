package telegram

import (
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
)

// The functions of this file build the messages that carry text from a person
// or from the database (descriptions, account, category and workspace names).
// Each one escapes every such string with EscapeHTML itself, so a caller
// cannot forget it; money goes through finance.FormatAmount like in the mails.

func confirmText(lg, kind string, cents int64, currency, description, date, account string) string {
	return lang.T(lg, lang.Confirm, kindWord(lg, kind), finance.FormatAmount(cents, currency),
		EscapeHTML(describe(description)), date, EscapeHTML(account))
}

func savedText(lg, kind string, cents int64, currency, description, date, account string) string {
	return lang.T(lg, lang.Saved, kindWord(lg, kind), finance.FormatAmount(cents, currency),
		EscapeHTML(describe(description)), date, EscapeHTML(account))
}

func lastLine(lg string, r finance.TxRow, currency string) string {
	sign := "+"
	if r.Kind == models.KindExpense {
		sign = "-"
	}

	return lang.T(lg, lang.LastLine, r.OccurredOn, sign+finance.FormatAmount(r.AmountCents, currency),
		EscapeHTML(describe(r.Description)), EscapeHTML(r.AccountName))
}

func balanceLine(lg, account string, cents int64, currency string) string {
	return lang.T(lg, lang.BalanceLine, EscapeHTML(account), finance.FormatAmount(cents, currency))
}

func budgetLine(lg string, st finance.BudgetStatus, currency string) string {
	name := st.Category
	if st.Overall {
		name = lang.T(lg, lang.OverallBudget)
	}

	return lang.T(lg, lang.BudgetLine, EscapeHTML(name), finance.FormatAmount(st.SpentCents, currency),
		finance.FormatAmount(st.AvailableCents, currency), st.Percent)
}
