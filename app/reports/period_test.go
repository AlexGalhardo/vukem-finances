package reports

import "testing"

func TestPeriod(t *testing.T) {
	t.Parallel()

	const today = "2028-03-15"

	cases := []struct {
		preset, from, to string
		want             Period
		prev             Period
	}{
		{ThisMonth, "", "", Period{ThisMonth, "2028-03-01", "2028-03-31"}, Period{Custom, "2028-02-01", "2028-02-29"}},
		{"", "", "", Period{ThisMonth, "2028-03-01", "2028-03-31"}, Period{Custom, "2028-02-01", "2028-02-29"}},
		{LastMonth, "", "", Period{LastMonth, "2028-02-01", "2028-02-29"}, Period{Custom, "2028-01-01", "2028-01-31"}},
		{Last3Months, "", "", Period{Last3Months, "2028-01-01", "2028-03-31"}, Period{Custom, "2027-10-01", "2027-12-31"}},
		{Last12Months, "", "", Period{Last12Months, "2027-04-01", "2028-03-31"}, Period{Custom, "2026-04-01", "2027-03-31"}},
		{ThisYear, "", "", Period{ThisYear, "2028-01-01", "2028-12-31"}, Period{Custom, "2027-01-01", "2027-12-31"}},
		{Custom, "2028-03-01", "2028-03-10", Period{Custom, "2028-03-01", "2028-03-10"}, Period{Custom, "2028-02-20", "2028-02-29"}},
		{Custom, "2028-02-29", "2028-02-29", Period{Custom, "2028-02-29", "2028-02-29"}, Period{Custom, "2028-02-28", "2028-02-28"}},
		{Custom, "2028-03-01", "2028-03-31", Period{Custom, "2028-03-01", "2028-03-31"}, Period{Custom, "2028-02-01", "2028-02-29"}},
	}

	for _, c := range cases {
		got, err := NewPeriod(c.preset, c.from, c.to, today)
		if err != nil || got != c.want {
			t.Errorf("NewPeriod(%q,%q,%q) = %+v, %v, want %+v", c.preset, c.from, c.to, got, err, c.want)
			continue
		}

		if prev := got.Previous(); prev != c.prev {
			t.Errorf("Previous of %+v = %+v, want %+v", got, prev, c.prev)
		}
	}

	t.Run("a non leap february", func(t *testing.T) {
		t.Parallel()

		got, err := NewPeriod(LastMonth, "", "", "2027-03-31")
		if err != nil || got.From != "2027-02-01" || got.To != "2027-02-28" {
			t.Errorf("last month of 2027-03-31 = %+v, %v", got, err)
		}
	})

	t.Run("bad input is refused", func(t *testing.T) {
		t.Parallel()

		for _, c := range [][3]string{
			{"weekly", "", ""}, {Custom, "2028-03-10", "2028-03-01"}, {Custom, "2028-02-30", "2028-03-01"},
			{Custom, "", ""}, {Custom, "2000-01-01", "2028-01-01"}, {Custom, "2028-3-1", "2028-03-05"},
		} {
			if _, err := NewPeriod(c[0], c[1], c[2], today); err == nil {
				t.Errorf("NewPeriod(%q,%q,%q) was accepted", c[0], c[1], c[2])
			}
		}
	})

	t.Run("months", func(t *testing.T) {
		t.Parallel()

		p, _ := NewPeriod(Custom, "2027-11-20", "2028-02-02", today)
		got := p.Months()

		if len(got) != 4 || got[0] != "2027-11" || got[3] != "2028-02" {
			t.Errorf("Months = %v", got)
		}
	})
}
