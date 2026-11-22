// Package lang holds the texts the Telegram bot sends, in the languages of the
// web app (en, pt-BR, es). A text is an fmt format string with Telegram HTML
// in it; the arguments are escaped by the caller (telegram.EscapeHTML) before
// they are given to T.
package lang

import (
	"fmt"
	"slices"
)

// The languages.
const (
	EN = "en"
	PT = "pt-BR"
	ES = "es"
)

// Codes lists the languages, in the order /lang shows them.
func Codes() []string { return []string{EN, PT, ES} }

// Valid reports whether code is a language of the bot.
func Valid(code string) bool { return slices.Contains(Codes(), code) }

// Name is how a language calls itself.
func Name(code string) string {
	switch code {
	case PT:
		return "Português (Brasil)"
	case ES:
		return "Español"
	default:
		return "English"
	}
}

// T formats the text of key in code; an unknown language or key falls back to
// English, then to the key itself.
func T(code, key string, args ...any) string {
	text, ok := texts[code][key]
	if !ok {
		text, ok = texts[EN][key]
	}

	if !ok {
		return key
	}

	if len(args) == 0 {
		return text
	}

	return fmt.Sprintf(text, args...)
}

// Keys lists the keys of a language, for the test that keeps them equal.
func Keys(code string) []string {
	out := make([]string, 0, len(texts[code]))
	for k := range texts[code] {
		out = append(out, k)
	}

	slices.Sort(out)

	return out
}

// The keys of the texts.
const (
	Privacy        = "privacy"
	Welcome        = "welcome"
	WelcomeLinked  = "welcome_linked"
	Linked         = "linked"
	LinkBad        = "link_bad"
	LinkTaken      = "link_taken"
	LinkSlow       = "link_slow"
	NotLinked      = "not_linked"
	Help           = "help"
	Unlinked       = "unlinked"
	NoAccess       = "no_access"
	NoWorkspace    = "no_workspace"
	ReadOnly       = "read_only"
	TooLong        = "too_long"
	ParseFail      = "parse_fail"
	ParseAmount    = "parse_amount"
	ParseDate      = "parse_date"
	RateNotice     = "rate_notice"
	Confirm        = "confirm"
	NoCategory     = "no_category"
	Cancel         = "cancel"
	Undo           = "undo"
	Saved          = "saved"
	Cancelled      = "cancelled"
	Expired        = "expired"
	Undone         = "undone"
	UndoLate       = "undo_late"
	Failed         = "failed"
	NoAccount      = "no_account"
	Income         = "income"
	Expense        = "expense"
	BalanceTitle   = "balance_title"
	BalanceLine    = "balance_line"
	NetWorth       = "net_worth"
	MonthSummary   = "month_summary"
	LastTitle      = "last_title"
	LastLine       = "last_line"
	LastNone       = "last_none"
	BudgetsTitle   = "budgets_title"
	BudgetLine     = "budget_line"
	BudgetsNone    = "budgets_none"
	OverallBudget  = "overall_budget"
	AccountChoose  = "account_choose"
	AccountSet     = "account_set"
	WorkspaceChoos = "workspace_choose"
	WorkspaceSet   = "workspace_set"
	WorkspaceNo    = "workspace_no"
	LangChoose     = "lang_choose"
	LangSet        = "lang_set"
	AlertBudget    = "alert_budget"
	AlertOverall   = "alert_overall"
	AlertCard      = "alert_card"
	AlertStatement = "alert_statement"
	AlertGoal      = "alert_goal"
	AlertRecurring = "alert_recurrence"
	AlertGeneric   = "alert_generic"
)

var texts = map[string]map[string]string{
	EN: {
		Privacy:       "Privacy: messages pass through Telegram's servers and are not end-to-end encrypted, and /balance shows balances in this chat. Use /unlink at any time.",
		Welcome:       "Hi! I record your income and expenses. To connect this chat, open <b>Settings, Telegram</b> in the app, press Connect and tap the link it gives you.\n\n%s",
		WelcomeLinked: "You are connected to <b>%s</b>. Send <code>-12,50 lunch</code> for an expense, <code>+3000 salary</code> for income, or /help.",
		Linked:        "Connected to <b>%s</b>. Send <code>-12,50 lunch</code> for an expense, <code>+3000 salary</code> for income, or /help.\n\n%s",
		LinkBad:       "That link code is invalid, expired or already used. Create a new one in the app (Settings, Telegram).",
		LinkTaken:     "This chat is already connected to an account. Send /unlink first.",
		LinkSlow:      "Too many attempts. Try again in a minute.",
		NotLinked:     "This chat is not connected. Open Settings, Telegram in the app to connect it.",
		Help: "<b>Record</b>\n<code>-12,50 lunch</code> expense (the minus is optional)\n<code>+3000 salary</code> income\nAdd <code>yesterday</code>, <code>15/11</code> or <code>2026-11-15</code> at the end for another date.\n\n" +
			"<b>Commands</b>\n/balance balances\n/month this month\n/last last 5 entries\n/budgets budget progress\n/account default account\n/workspace switch workspace\n/lang language\n/unlink disconnect\n/help this text",
		Unlinked:       "Disconnected. To remove the messages from this chat, delete the chat history in Telegram.",
		NoAccess:       "You no longer have access to the workspace this chat used. Use /workspace to pick another one.",
		NoWorkspace:    "Your account belongs to no workspace.",
		ReadOnly:       "Your role in this workspace is read only: you cannot record entries.",
		TooLong:        "That message is too long (500 characters at most).",
		ParseFail:      "I could not read that. Try <code>-12,50 lunch</code> or <code>+3000 salary</code>, or send /help.",
		ParseAmount:    "I could not read the amount. Use a number like <code>12,50</code> or <code>1.234,56</code>, above zero.",
		ParseDate:      "That date does not exist. Use <code>today</code>, <code>yesterday</code>, <code>15/11</code> or <code>2026-11-15</code>.",
		RateNotice:     "Too many messages. Try again in a minute.",
		Confirm:        "<b>%s %s</b>\n%s\nDate: %s\nAccount: %s\n\nChoose a category:",
		NoCategory:     "No category",
		Cancel:         "Cancel",
		Undo:           "Undo",
		Saved:          "Saved: <b>%s %s</b> (%s, %s). Account: %s.",
		Cancelled:      "Cancelled.",
		Expired:        "That draft expired or was already used. Send the entry again.",
		Undone:         "Undone.",
		UndoLate:       "The undo window (60 seconds) is over. Edit or delete it in the app.",
		Failed:         "Something went wrong. Try again in a moment.",
		NoAccount:      "There is no account to record in. Create one in the app first.",
		Income:         "Income",
		Expense:        "Expense",
		BalanceTitle:   "<b>Balances</b>",
		BalanceLine:    "%s: %s",
		NetWorth:       "<b>Net worth: %s</b>",
		MonthSummary:   "<b>%s</b>\nIncome: %s\nExpense: %s\nNet: %s",
		LastTitle:      "<b>Last entries</b>",
		LastLine:       "%s %s %s (%s)",
		LastNone:       "No entries yet.",
		BudgetsTitle:   "<b>Budgets of %s</b>",
		BudgetLine:     "%s: %s of %s (%d%%)",
		BudgetsNone:    "No budgets yet.",
		OverallBudget:  "Overall",
		AccountChoose:  "Choose the default account:",
		AccountSet:     "Default account: <b>%s</b>.",
		WorkspaceChoos: "Choose the workspace:",
		WorkspaceSet:   "Workspace: <b>%s</b>.",
		WorkspaceNo:    "You are not a member of that workspace.",
		LangChoose:     "Choose the language:",
		LangSet:        "Language: %s.",
		AlertBudget:    "<b>Budget alert</b>: %s reached %d%% (%s of %s).",
		AlertOverall:   "<b>Budget alert</b>: the overall budget reached 100%% (%s of %s).",
		AlertCard:      "<b>Card alert</b>: %s used %s of its %s limit.",
		AlertStatement: "<b>Statement due</b>: %s is due on %s (%s left).",
		AlertGoal:      "<b>Goal reached</b>: %s (%s).",
		AlertRecurring: "<b>Recurring entry failed</b>: \"%s\" due on %s was not posted.",
		AlertGeneric:   "<b>Alert</b> in %s.",
	},
	PT: {
		Privacy:       "Privacidade: as mensagens passam pelos servidores do Telegram e não têm criptografia de ponta a ponta, e /balance mostra saldos neste chat. Use /unlink quando quiser.",
		Welcome:       "Olá! Eu registro suas receitas e despesas. Para conectar este chat, abra <b>Configurações, Telegram</b> no app, toque em Conectar e abra o link que aparecer.\n\n%s",
		WelcomeLinked: "Você está conectado a <b>%s</b>. Envie <code>-12,50 almoço</code> para uma despesa, <code>+3000 salário</code> para uma receita, ou /help.",
		Linked:        "Conectado a <b>%s</b>. Envie <code>-12,50 almoço</code> para uma despesa, <code>+3000 salário</code> para uma receita, ou /help.\n\n%s",
		LinkBad:       "Esse código é inválido, expirou ou já foi usado. Crie outro no app (Configurações, Telegram).",
		LinkTaken:     "Este chat já está conectado a uma conta. Envie /unlink antes.",
		LinkSlow:      "Tentativas demais. Tente de novo em um minuto.",
		NotLinked:     "Este chat não está conectado. Abra Configurações, Telegram no app para conectar.",
		Help: "<b>Registrar</b>\n<code>-12,50 almoço</code> despesa (o menos é opcional)\n<code>+3000 salário</code> receita\nAcrescente <code>ontem</code>, <code>15/11</code> ou <code>2026-11-15</code> no fim para outra data.\n\n" +
			"<b>Comandos</b>\n/balance saldos\n/month este mês\n/last últimos 5 lançamentos\n/budgets progresso dos orçamentos\n/account conta padrão\n/workspace trocar de espaço\n/lang idioma\n/unlink desconectar\n/help este texto",
		Unlinked:       "Desconectado. Para apagar as mensagens deste chat, limpe o histórico da conversa no Telegram.",
		NoAccess:       "Você não tem mais acesso ao espaço que este chat usava. Use /workspace para escolher outro.",
		NoWorkspace:    "Sua conta não pertence a nenhum espaço.",
		ReadOnly:       "Seu papel neste espaço é só leitura: você não pode registrar lançamentos.",
		TooLong:        "Mensagem longa demais (no máximo 500 caracteres).",
		ParseFail:      "Não entendi. Tente <code>-12,50 almoço</code> ou <code>+3000 salário</code>, ou envie /help.",
		ParseAmount:    "Não entendi o valor. Use um número como <code>12,50</code> ou <code>1.234,56</code>, maior que zero.",
		ParseDate:      "Essa data não existe. Use <code>hoje</code>, <code>ontem</code>, <code>15/11</code> ou <code>2026-11-15</code>.",
		RateNotice:     "Mensagens demais. Tente de novo em um minuto.",
		Confirm:        "<b>%s %s</b>\n%s\nData: %s\nConta: %s\n\nEscolha a categoria:",
		NoCategory:     "Sem categoria",
		Cancel:         "Cancelar",
		Undo:           "Desfazer",
		Saved:          "Salvo: <b>%s %s</b> (%s, %s). Conta: %s.",
		Cancelled:      "Cancelado.",
		Expired:        "Esse rascunho expirou ou já foi usado. Envie o lançamento de novo.",
		Undone:         "Desfeito.",
		UndoLate:       "O prazo para desfazer (60 segundos) acabou. Edite ou apague no app.",
		Failed:         "Algo deu errado. Tente de novo em instantes.",
		NoAccount:      "Não há conta para registrar. Crie uma no app primeiro.",
		Income:         "Receita",
		Expense:        "Despesa",
		BalanceTitle:   "<b>Saldos</b>",
		BalanceLine:    "%s: %s",
		NetWorth:       "<b>Patrimônio: %s</b>",
		MonthSummary:   "<b>%s</b>\nReceitas: %s\nDespesas: %s\nSaldo: %s",
		LastTitle:      "<b>Últimos lançamentos</b>",
		LastLine:       "%s %s %s (%s)",
		LastNone:       "Nenhum lançamento ainda.",
		BudgetsTitle:   "<b>Orçamentos de %s</b>",
		BudgetLine:     "%s: %s de %s (%d%%)",
		BudgetsNone:    "Nenhum orçamento ainda.",
		OverallBudget:  "Geral",
		AccountChoose:  "Escolha a conta padrão:",
		AccountSet:     "Conta padrão: <b>%s</b>.",
		WorkspaceChoos: "Escolha o espaço:",
		WorkspaceSet:   "Espaço: <b>%s</b>.",
		WorkspaceNo:    "Você não é membro desse espaço.",
		LangChoose:     "Escolha o idioma:",
		LangSet:        "Idioma: %s.",
		AlertBudget:    "<b>Alerta de orçamento</b>: %s chegou a %d%% (%s de %s).",
		AlertOverall:   "<b>Alerta de orçamento</b>: o orçamento geral chegou a 100%% (%s de %s).",
		AlertCard:      "<b>Alerta de cartão</b>: %s usou %s do limite de %s.",
		AlertStatement: "<b>Fatura a vencer</b>: %s vence em %s (faltam %s).",
		AlertGoal:      "<b>Meta atingida</b>: %s (%s).",
		AlertRecurring: "<b>Lançamento recorrente falhou</b>: \"%s\" de %s não foi lançado.",
		AlertGeneric:   "<b>Alerta</b> em %s.",
	},
	ES: {
		Privacy:       "Privacidad: los mensajes pasan por los servidores de Telegram y no tienen cifrado de extremo a extremo, y /balance muestra saldos en este chat. Usa /unlink cuando quieras.",
		Welcome:       "¡Hola! Registro tus ingresos y gastos. Para conectar este chat, abre <b>Ajustes, Telegram</b> en la app, pulsa Conectar y abre el enlace que aparece.\n\n%s",
		WelcomeLinked: "Estás conectado a <b>%s</b>. Envía <code>-12,50 almuerzo</code> para un gasto, <code>+3000 sueldo</code> para un ingreso, o /help.",
		Linked:        "Conectado a <b>%s</b>. Envía <code>-12,50 almuerzo</code> para un gasto, <code>+3000 sueldo</code> para un ingreso, o /help.\n\n%s",
		LinkBad:       "Ese código no es válido, caducó o ya se usó. Crea otro en la app (Ajustes, Telegram).",
		LinkTaken:     "Este chat ya está conectado a una cuenta. Envía /unlink antes.",
		LinkSlow:      "Demasiados intentos. Prueba de nuevo en un minuto.",
		NotLinked:     "Este chat no está conectado. Abre Ajustes, Telegram en la app para conectarlo.",
		Help: "<b>Registrar</b>\n<code>-12,50 almuerzo</code> gasto (el menos es opcional)\n<code>+3000 sueldo</code> ingreso\nAñade <code>ayer</code>, <code>15/11</code> o <code>2026-11-15</code> al final para otra fecha.\n\n" +
			"<b>Comandos</b>\n/balance saldos\n/month este mes\n/last últimos 5 movimientos\n/budgets avance de presupuestos\n/account cuenta predeterminada\n/workspace cambiar de espacio\n/lang idioma\n/unlink desconectar\n/help este texto",
		Unlinked:       "Desconectado. Para borrar los mensajes de este chat, elimina el historial en Telegram.",
		NoAccess:       "Ya no tienes acceso al espacio que usaba este chat. Usa /workspace para elegir otro.",
		NoWorkspace:    "Tu cuenta no pertenece a ningún espacio.",
		ReadOnly:       "Tu rol en este espacio es de solo lectura: no puedes registrar movimientos.",
		TooLong:        "Mensaje demasiado largo (máximo 500 caracteres).",
		ParseFail:      "No lo entendí. Prueba <code>-12,50 almuerzo</code> o <code>+3000 sueldo</code>, o envía /help.",
		ParseAmount:    "No entendí el importe. Usa un número como <code>12,50</code> o <code>1.234,56</code>, mayor que cero.",
		ParseDate:      "Esa fecha no existe. Usa <code>hoy</code>, <code>ayer</code>, <code>15/11</code> o <code>2026-11-15</code>.",
		RateNotice:     "Demasiados mensajes. Prueba de nuevo en un minuto.",
		Confirm:        "<b>%s %s</b>\n%s\nFecha: %s\nCuenta: %s\n\nElige una categoría:",
		NoCategory:     "Sin categoría",
		Cancel:         "Cancelar",
		Undo:           "Deshacer",
		Saved:          "Guardado: <b>%s %s</b> (%s, %s). Cuenta: %s.",
		Cancelled:      "Cancelado.",
		Expired:        "Ese borrador caducó o ya se usó. Envía el movimiento de nuevo.",
		Undone:         "Deshecho.",
		UndoLate:       "El plazo para deshacer (60 segundos) terminó. Edítalo o bórralo en la app.",
		Failed:         "Algo salió mal. Prueba de nuevo en un momento.",
		NoAccount:      "No hay cuenta donde registrar. Crea una en la app primero.",
		Income:         "Ingreso",
		Expense:        "Gasto",
		BalanceTitle:   "<b>Saldos</b>",
		BalanceLine:    "%s: %s",
		NetWorth:       "<b>Patrimonio: %s</b>",
		MonthSummary:   "<b>%s</b>\nIngresos: %s\nGastos: %s\nNeto: %s",
		LastTitle:      "<b>Últimos movimientos</b>",
		LastLine:       "%s %s %s (%s)",
		LastNone:       "Aún no hay movimientos.",
		BudgetsTitle:   "<b>Presupuestos de %s</b>",
		BudgetLine:     "%s: %s de %s (%d%%)",
		BudgetsNone:    "Aún no hay presupuestos.",
		OverallBudget:  "General",
		AccountChoose:  "Elige la cuenta predeterminada:",
		AccountSet:     "Cuenta predeterminada: <b>%s</b>.",
		WorkspaceChoos: "Elige el espacio:",
		WorkspaceSet:   "Espacio: <b>%s</b>.",
		WorkspaceNo:    "No eres miembro de ese espacio.",
		LangChoose:     "Elige el idioma:",
		LangSet:        "Idioma: %s.",
		AlertBudget:    "<b>Alerta de presupuesto</b>: %s llegó al %d%% (%s de %s).",
		AlertOverall:   "<b>Alerta de presupuesto</b>: el presupuesto general llegó al 100%% (%s de %s).",
		AlertCard:      "<b>Alerta de tarjeta</b>: %s usó %s de su límite de %s.",
		AlertStatement: "<b>Estado de cuenta por vencer</b>: %s vence el %s (faltan %s).",
		AlertGoal:      "<b>Meta alcanzada</b>: %s (%s).",
		AlertRecurring: "<b>Falló un movimiento recurrente</b>: \"%s\" del %s no se registró.",
		AlertGeneric:   "<b>Alerta</b> en %s.",
	},
}
