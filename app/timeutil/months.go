// Package timeutil holds the calendar arithmetic of the app. time.AddDate
// normalizes an overflowing day (Jan 31 plus one month is March 3, or March
// 2), which is the bug of month-based features; AddDate with a month
// argument is allowed only here (a test enforces it).
package timeutil

import "time"

// LastDayOfMonth is the number of days of the month: 28 to 31.
func LastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// AddMonths moves t by n months and clamps the day to the last day of the
// target month: Jan 31 plus 1 is Feb 28 (Feb 29 in a leap year), Mar 31
// minus 1 is Feb 28 or 29, Dec 31 plus 2 is Feb 28 or 29. The time of day
// and the location are kept.
func AddMonths(t time.Time, n int) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location()).
		AddDate(0, n, 0)

	day := min(t.Day(), LastDayOfMonth(first.Year(), first.Month()))

	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}
