package timeutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 9, 30, 0, 0, time.UTC) }

func TestAddMonths(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		from time.Time
		n    int
		want time.Time
	}{
		{"jan 31 plus 1, common year", date(2027, time.January, 31), 1, date(2027, time.February, 28)},
		{"jan 31 plus 1, leap year", date(2028, time.January, 31), 1, date(2028, time.February, 29)},
		{"mar 31 minus 1", date(2027, time.March, 31), -1, date(2027, time.February, 28)},
		{"mar 31 minus 1, leap year", date(2028, time.March, 31), -1, date(2028, time.February, 29)},
		{"dec 31 plus 2", date(2026, time.December, 31), 2, date(2027, time.February, 28)},
		{"dec 31 plus 2, leap year", date(2027, time.December, 31), 2, date(2028, time.February, 29)},
		{"jan 31 plus 3 keeps 30 out of april", date(2026, time.January, 31), 3, date(2026, time.April, 30)},
		{"plain day", date(2026, time.November, 15), 1, date(2026, time.December, 15)},
		{"year boundary back", date(2026, time.January, 15), -2, date(2025, time.November, 15)},
		{"twelve months", date(2028, time.February, 29), 12, date(2029, time.February, 28)},
		{"zero", date(2026, time.May, 31), 0, date(2026, time.May, 31)},
	} {
		if got := AddMonths(tc.from, tc.n); !got.Equal(tc.want) {
			t.Errorf("%s: AddMonths(%s, %d) = %s, want %s", tc.name, tc.from.Format(time.DateOnly), tc.n,
				got.Format(time.DateOnly), tc.want.Format(time.DateOnly))
		}
	}
}

func TestLastDayOfMonth(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		year  int
		month time.Month
		want  int
	}{
		{2027, time.February, 28}, {2028, time.February, 29}, {2100, time.February, 28}, {2000, time.February, 29},
		{2026, time.April, 30}, {2026, time.December, 31},
	} {
		if got := LastDayOfMonth(tc.year, tc.month); got != tc.want {
			t.Errorf("LastDayOfMonth(%d, %s) = %d, want %d", tc.year, tc.month, got, tc.want)
		}
	}
}

// addDateMonths finds AddDate calls whose month argument is not a literal
// zero, whatever the spacing.
var addDateMonths = regexp.MustCompile(`AddDate\(\s*[^,]+,\s*[^0\s]`)

func TestNoAddDateWithMonthsOutsideTimeutil(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			switch d.Name() {
			case "timeutil", "node_modules", ".git", "vendor":
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if addDateMonths.Match(src) {
			t.Errorf("%s calls AddDate with a month argument: use timeutil.AddMonths", path)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
