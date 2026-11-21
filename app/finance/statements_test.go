package finance

import (
	"math/rand/v2"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(isoDate, s)
	if err != nil {
		panic(err)
	}

	return t
}

func TestClosingDate(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		year  int
		month time.Month
		day   int
		want  string
	}{
		{2027, time.February, 31, "2027-02-28"},
		{2028, time.February, 31, "2028-02-29"},
		{2027, time.February, 29, "2027-02-28"},
		{2028, time.February, 29, "2028-02-29"},
		{2027, time.April, 31, "2027-04-30"},
		{2027, time.January, 31, "2027-01-31"},
		{2027, time.March, 15, "2027-03-15"},
		{2027, time.February, 1, "2027-02-01"},
		{2027, time.December, 31, "2027-12-31"},
		{2027, time.Month(13), 10, "2028-01-10"},
		{2100, time.February, 30, "2100-02-28"},
	} {
		if got := closingDate(c.year, c.month, c.day).Format(isoDate); got != c.want {
			t.Errorf("closingDate(%d, %d, %d) = %s, want %s", c.year, c.month, c.day, got, c.want)
		}
	}
}

func TestStatementFor(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		date       string
		closingDay int
		want       string
	}{
		{"2027-02-28", 31, "2027-02-28"},
		{"2027-03-01", 31, "2027-03-31"},
		{"2027-02-27", 31, "2027-02-28"},
		{"2028-02-29", 31, "2028-02-29"},
		{"2028-03-01", 31, "2028-03-31"},
		{"2026-11-10", 10, "2026-11-10"},
		{"2026-11-11", 10, "2026-12-10"},
		{"2026-11-09", 10, "2026-11-10"},
		{"2026-12-10", 10, "2026-12-10"},
		{"2026-12-11", 10, "2027-01-10"},
		{"2026-12-31", 10, "2027-01-10"},
		{"2026-12-31", 31, "2026-12-31"},
		{"2027-01-01", 1, "2027-01-01"},
		{"2027-01-02", 1, "2027-02-01"},
		{"2026-12-02", 1, "2027-01-01"},
		{"2027-01-31", 30, "2027-02-28"},
		{"2027-01-30", 30, "2027-01-30"},
		{"2027-02-28", 30, "2027-02-28"},
		{"2027-03-01", 30, "2027-03-30"},
		{"2027-04-30", 31, "2027-04-30"},
		{"2027-05-01", 31, "2027-05-31"},
	} {
		if got := statementFor(day(c.date), c.closingDay).Format(isoDate); got != c.want {
			t.Errorf("statementFor(%s, %d) = %s, want %s", c.date, c.closingDay, got, c.want)
		}
	}
}

func TestDueDate(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		closing string
		dueDay  int
		want    string
	}{
		{"2026-11-10", 20, "2026-11-20"},
		{"2026-11-20", 5, "2026-12-05"},
		{"2026-12-20", 5, "2027-01-05"},
		{"2027-02-28", 10, "2027-03-10"},
		{"2027-02-28", 31, "2027-03-31"},
		{"2027-01-31", 10, "2027-02-10"},
		{"2027-01-10", 31, "2027-01-31"},
		{"2027-01-10", 10, "2027-02-10"},
		{"2026-12-31", 31, "2027-01-31"},
		{"2028-02-29", 5, "2028-03-05"},
		{"2027-01-05", 30, "2027-01-30"},
		{"2027-01-05", 31, "2027-01-31"},
	} {
		if got := dueDate(day(c.closing), c.dueDay).Format(isoDate); got != c.want {
			t.Errorf("dueDate(%s, %d) = %s, want %s", c.closing, c.dueDay, got, c.want)
		}
	}
}

func TestStatementStatus(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name        string
		today       string
		total, paid int64
		want        string
	}{
		{"before closing", "2026-11-05", 100, 0, StatementOpen},
		{"on closing day", "2026-11-10", 100, 0, StatementOpen},
		{"closed unpaid", "2026-11-11", 100, 0, StatementClosed},
		{"on due day", "2026-11-20", 100, 40, StatementClosed},
		{"overdue", "2026-11-21", 100, 99, StatementOverdue},
		{"paid exactly", "2026-11-21", 100, 100, StatementPaid},
		{"overpaid", "2026-11-12", 100, 150, StatementPaid},
		{"nothing owed", "2026-11-12", 0, 0, StatementPaid},
		{"net credit", "2026-11-12", -50, 0, StatementPaid},
		{"paid while open", "2026-11-05", 100, 100, StatementOpen},
	} {
		if got := statementStatus(c.today, "2026-11-10", "2026-11-20", c.total, c.paid); got != c.want {
			t.Errorf("%s: status = %s, want %s", c.name, got, c.want)
		}
	}
}

// Every date belongs to exactly one statement, a statement never closes
// before the purchase, and statements grow with the date.
func TestStatementForIsMonotonic(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(9, 27))

	for range 200 {
		closingDay := rng.IntN(31) + 1
		prev := time.Time{}
		prevRef := time.Time{}

		for d := day("2026-01-01"); d.Before(day("2030-01-01")); d = d.AddDate(0, 0, 1) {
			ref := statementFor(d, closingDay)

			if ref.Before(d) {
				t.Fatalf("day %d: %s closes on %s, before the purchase", closingDay, d.Format(isoDate), ref.Format(isoDate))
			}

			if ref.Sub(d) > 31*24*time.Hour {
				t.Fatalf("day %d: %s waits more than a month for %s", closingDay, d.Format(isoDate), ref.Format(isoDate))
			}

			if !prev.IsZero() && ref.Before(prevRef) {
				t.Fatalf("day %d: statement went back from %s to %s at %s", closingDay, prevRef.Format(isoDate), ref.Format(isoDate), d.Format(isoDate))
			}

			// The statement of a closing date is itself.
			if statementFor(ref, closingDay) != ref {
				t.Fatalf("day %d: the closing date %s is not in its own statement", closingDay, ref.Format(isoDate))
			}

			prev, prevRef = d, ref
		}
	}
}
