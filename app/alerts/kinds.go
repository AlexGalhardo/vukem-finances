// Package alerts turns the state of a workspace (budgets, cards, goals,
// recurrences) into notifications and delivers them through channels.
//
// Evaluate reads the current data and returns the notifications that should
// exist; Run stores the missing ones (the unique key is the dedupe) and sends
// the mail of the rows that still need it. Nothing is derived from "what just
// changed", so a retried, doubled or concurrent run makes and sends nothing
// twice, and an edit or a delete is handled like any other change.
package alerts

// The kinds of notification, and their subject keys (the second half of the
// dedupe key; the first is the person):
//
//	budget_80, budget_100      budget:<budget uid>:<YYYY-MM>
//	overall_budget_100         budget:<budget uid>:<YYYY-MM>
//	card_limit_80              card:<card uid>:<YYYY-MM>
//	statement_due              card:<card uid>:<statement closing date>
//	goal_reached               goal:<goal uid>
//	recurrence_failed          recurrence:<rule uid>:<date of the occurrence that failed>
//
// A client renders TitleKey ("alerts.<kind>") with Params in its own
// language; no prose is stored.
const (
	KindBudget80         = "budget_80"
	KindBudget100        = "budget_100"
	KindOverallBudget100 = "overall_budget_100"
	KindCardLimit80      = "card_limit_80"
	KindStatementDue     = "statement_due"
	KindGoalReached      = "goal_reached"
	KindRecurrenceFailed = "recurrence_failed"
)

// Kinds lists every kind, in the order the preferences page shows them.
func Kinds() []string {
	return []string{
		KindBudget80, KindBudget100, KindOverallBudget100, KindCardLimit80,
		KindStatementDue, KindGoalReached, KindRecurrenceFailed,
	}
}

// ValidKind reports whether kind is one of Kinds.
func ValidKind(kind string) bool {
	for _, k := range Kinds() {
		if k == kind {
			return true
		}
	}

	return false
}

// TitleKey is the translation key of a kind.
func TitleKey(kind string) string { return "alerts." + kind }

// statementDueDays is how many days before the due date a statement alerts.
const statementDueDays = 3

// Candidate is a notification that should exist for the workspace; the
// recipients are added by Run.
type Candidate struct {
	Kind    string
	Subject string
	Params  map[string]any
}
