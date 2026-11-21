package finance

import (
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
)

// Recurrence frequencies.
const (
	FreqDaily   = "daily"
	FreqWeekly  = "weekly"
	FreqMonthly = "monthly"
	FreqYearly  = "yearly"
)

// maxYear bounds the engine: ISO dates stay four digits.
const maxYear = 9999

// Rule is the pure part of a recurrence: docs/specs/recurrence.md. EndDate
// and MaxCount are optional (empty and 0).
type Rule struct {
	Frequency string
	Interval  int
	StartDate string
	EndDate   string
	MaxCount  int
}

// validFrequency reports whether f is one of the four frequencies.
func validFrequency(f string) bool {
	switch f {
	case FreqDaily, FreqWeekly, FreqMonthly, FreqYearly:
		return true
	}

	return false
}

// nth is the date of the k-th occurrence, computed from the start date and k
// alone so a clamped day never drifts. ok is false when the date is past the
// engine's last year or the rule is malformed.
func (r Rule) nth(start time.Time, k int) (time.Time, bool) {
	if r.Interval < 1 || k < 0 {
		return time.Time{}, false
	}

	n := k * r.Interval

	var d time.Time

	switch r.Frequency {
	case FreqDaily:
		d = start.AddDate(0, 0, n)
	case FreqWeekly:
		d = start.AddDate(0, 0, 7*n)
	case FreqMonthly:
		d = timeutil.AddMonths(start, n)
	case FreqYearly:
		d = timeutil.AddMonths(start, 12*n)
	default:
		return time.Time{}, false
	}

	return d, d.Year() <= maxYear
}

// firstIndex is a k that is never later than the first occurrence on or after
// from, so listing a window far from the start does not walk from k = 0.
func (r Rule) firstIndex(start, from time.Time) int {
	if !from.After(start) || r.Interval < 1 {
		return 0
	}

	var units int

	switch r.Frequency {
	case FreqDaily:
		units = int(from.Sub(start).Hours()/24) / r.Interval
	case FreqWeekly:
		units = int(from.Sub(start).Hours()/24) / (7 * r.Interval)
	case FreqMonthly:
		months := (from.Year()-start.Year())*12 + int(from.Month()-start.Month())
		units = months / r.Interval
	case FreqYearly:
		units = (from.Year() - start.Year()) / r.Interval
	}

	return max(0, units-1)
}

// Occurrences lists, ascending, the dates of the occurrences of r with
// from <= date <= to, honoring the end date and the count (which always
// count from the start date). limit caps the length of the answer when it is
// above 0. Malformed input yields no dates.
func (r Rule) Occurrences(from, to string, limit int) []string {
	start, err := time.Parse(isoDate, r.StartDate)
	if err != nil || !validFrequency(r.Frequency) || r.Interval < 1 {
		return nil
	}

	lo, err := time.Parse(isoDate, from)
	if err != nil {
		return nil
	}

	hi, err := time.Parse(isoDate, to)
	if err != nil || hi.Before(lo) {
		return nil
	}

	end := time.Time{}

	if r.EndDate != "" {
		if end, err = time.Parse(isoDate, r.EndDate); err != nil {
			return nil
		}
	}

	var out []string

	for k := r.firstIndex(start, lo); ; k++ {
		if r.MaxCount > 0 && k >= r.MaxCount {
			break
		}

		d, ok := r.nth(start, k)
		if !ok || d.After(hi) || (!end.IsZero() && d.After(end)) {
			break
		}

		if d.Before(lo) {
			continue
		}

		out = append(out, d.Format(isoDate))

		if limit > 0 && len(out) >= limit {
			break
		}
	}

	return out
}
