package reports

import "testing"

func TestCompareBasisPoints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		current, previous, want int64
		ok                      bool
	}{
		{110, 100, 1000, true},
		{101, 100, 100, true},
		{1, 3, -6667, true},
		{0, 100, -10000, true},
		{5, 0, 0, false},
		{0, 0, 0, false},
		{-50, -100, 5000, true},
		{3, 2, 5000, true},
		{1, 1 << 55, -10000, true},
	}

	for _, c := range cases {
		row := compareRow("k", c.current, c.previous)
		if row.ChangeBps != c.want || row.NotApplicable == c.ok {
			t.Errorf("compare(%d, %d) = %+v, want %d bps (applicable %v)", c.current, c.previous, row, c.want, c.ok)
		}
	}

	big := compareRow("k", 1<<55, 1)
	if big.NotApplicable || big.ChangeBps <= 0 {
		t.Errorf("a huge ratio overflowed: %+v", big)
	}
}
