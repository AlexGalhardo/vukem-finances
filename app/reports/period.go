// Package reports answers "where did my money go?": income against expense
// by month, spending by category, account and tag, the net worth series and
// the comparison with the previous period. Every report is a function that
// returns plain structs, shared by the page and the public API. Money is
// int64 cents, dates are the ISO calendar dates of the workspace, and every
// query goes through workspaces.Scoped.
package reports

import (
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
)

// Period presets.
const (
	ThisMonth    = "this_month"
	LastMonth    = "last_month"
	Last3Months  = "last_3_months"
	Last12Months = "last_12_months"
	ThisYear     = "this_year"
	Custom       = "custom"
)

// maxCustomDays bounds a period (ten years), so a series stays small.
const maxCustomDays = 3660

const isoDate = "2006-01-02"

// ErrPeriod is the answer to a period that cannot be read.
var ErrPeriod = errs.New(http.StatusUnprocessableEntity,
	"Choose a valid period: dates as YYYY-MM-DD, the start not after the end, at most ten years.")

// Period is a closed range of workspace-local calendar dates.
type Period struct {
	Preset string `json:"preset"`
	From   string `json:"from"`
	To     string `json:"to"`
}

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(isoDate, s)
	if err != nil || t.Format(isoDate) != s {
		return time.Time{}, ErrPeriod
	}

	return t, nil
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func monthEnd(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), timeutil.LastDayOfMonth(t.Year(), t.Month()), 0, 0, 0, 0, time.UTC)
}

// NewPeriod resolves a preset against today (the workspace-local ISO date).
// Month presets are whole calendar months: last_3_months and last_12_months
// are the current month and the ones before it. An empty preset is
// this_month. custom reads from and to.
func NewPeriod(preset, from, to, today string) (Period, error) {
	now, err := parseDate(today)
	if err != nil {
		return Period{}, err
	}

	var start, end time.Time

	switch preset {
	case "", ThisMonth:
		preset, start, end = ThisMonth, monthStart(now), monthEnd(now)
	case LastMonth:
		start = timeutil.AddMonths(monthStart(now), -1)
		end = monthEnd(start)
	case Last3Months:
		start, end = timeutil.AddMonths(monthStart(now), -2), monthEnd(now)
	case Last12Months:
		start, end = timeutil.AddMonths(monthStart(now), -11), monthEnd(now)
	case ThisYear:
		start = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(now.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
	case Custom:
		if start, err = parseDate(from); err != nil {
			return Period{}, err
		}

		if end, err = parseDate(to); err != nil {
			return Period{}, err
		}
	default:
		return Period{}, ErrPeriod
	}

	if end.Before(start) || days(start, end) > maxCustomDays {
		return Period{}, ErrPeriod
	}

	return Period{Preset: preset, From: start.Format(isoDate), To: end.Format(isoDate)}, nil
}

// days counts the dates from a to b, both included.
func days(a, b time.Time) int { return int(b.Sub(a).Hours()/24) + 1 }

// wholeMonths says how many calendar months the period covers when it starts
// on the first and ends on the last day of a month; otherwise 0.
func (p Period) wholeMonths() int {
	from, err1 := parseDate(p.From)
	to, err2 := parseDate(p.To)

	if err1 != nil || err2 != nil || from.Day() != 1 || !to.Equal(monthEnd(to)) {
		return 0
	}

	return (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month()) + 1
}

// Previous is the period of the same length right before p: for whole
// calendar months, the same number of months before (March 2028 is compared
// with February 2028); otherwise the same number of days.
func (p Period) Previous() Period {
	from, _ := parseDate(p.From)
	to, _ := parseDate(p.To)

	if n := p.wholeMonths(); n > 0 {
		start := timeutil.AddMonths(from, -n)

		return Period{Preset: Custom, From: start.Format(isoDate), To: monthEnd(timeutil.AddMonths(from, -1)).Format(isoDate)}
	}

	n := days(from, to)
	end := from.AddDate(0, 0, -1)

	return Period{Preset: Custom, From: end.AddDate(0, 0, -(n - 1)).Format(isoDate), To: end.Format(isoDate)}
}

// Months lists the "YYYY-MM" of every month the period touches, oldest first.
func (p Period) Months() []string {
	from, _ := parseDate(p.From)
	to, _ := parseDate(p.To)

	var out []string

	for m := monthStart(from); !m.After(to); m = timeutil.AddMonths(m, 1) {
		out = append(out, m.Format("2006-01"))
	}

	return out
}
