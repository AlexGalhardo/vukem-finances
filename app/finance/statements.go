package finance

import (
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
)

// Statement status, derived from today, the closing and due dates and what
// has been paid; never stored.
const (
	StatementOpen    = "open"
	StatementClosed  = "closed"
	StatementPaid    = "paid"
	StatementOverdue = "overdue"
)

// closingDate is the day a card statement closes in a month: the closing day,
// or the last day of a shorter month (day 31 closes on Feb 28 or 29). month
// may overflow (13 is January of the next year).
func closingDate(year int, month time.Month, closingDay int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	return time.Date(first.Year(), first.Month(), min(closingDay, timeutil.LastDayOfMonth(first.Year(), first.Month())), 0, 0, 0, 0, time.UTC)
}

// statementFor returns the closing date of the statement a purchase made on
// date belongs to: the closing of its own month when the purchase is on or
// before it, the next month's otherwise. The statement is identified by that
// closing date.
func statementFor(date time.Time, closingDay int) time.Time {
	c := closingDate(date.Year(), date.Month(), closingDay)
	if date.After(c) {
		return closingDate(date.Year(), date.Month()+1, closingDay)
	}

	return c
}

// dueDate is the first occurrence of the due day strictly after the closing
// date, clamped to the length of the month.
func dueDate(closing time.Time, dueDay int) time.Time {
	d := closingDate(closing.Year(), closing.Month(), dueDay)
	if !d.After(closing) {
		return closingDate(closing.Year(), closing.Month()+1, dueDay)
	}

	return d
}

// statementRefFor is statementFor on ISO dates. An invalid date yields "".
func statementRefFor(date string, closingDay int) string {
	t, err := time.Parse(isoDate, date)
	if err != nil {
		return ""
	}

	return statementFor(t, closingDay).Format(isoDate)
}

// statementStatus derives the status of a statement. today, closing and due
// are ISO dates (they compare as text); total is what was bought (net of
// refunds) and paid what the payments tagged with the statement add up to.
func statementStatus(today, closing, due string, total, paid int64) string {
	switch {
	case today <= closing:
		return StatementOpen
	case paid >= total:
		return StatementPaid
	case today > due:
		return StatementOverdue
	default:
		return StatementClosed
	}
}
