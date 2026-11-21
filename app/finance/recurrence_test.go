package finance

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

type vector struct {
	Name string `json:"name"`
	Rule struct {
		Frequency string  `json:"frequency"`
		Interval  int     `json:"interval"`
		StartDate string  `json:"start_date"`
		EndDate   *string `json:"end_date"`
		MaxCount  *int    `json:"max_count"`
	} `json:"rule"`
	From   string   `json:"from"`
	To     string   `json:"to"`
	Expect []string `json:"expect"`
}

func TestRecurrenceVectors(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/recurrence-vectors.json")
	if err != nil {
		t.Fatal(err)
	}

	var vectors []vector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}

	if len(vectors) < 60 {
		t.Fatalf("%d vectors, want at least 60", len(vectors))
	}

	for _, v := range vectors {
		r := Rule{Frequency: v.Rule.Frequency, Interval: v.Rule.Interval, StartDate: v.Rule.StartDate}
		if v.Rule.EndDate != nil {
			r.EndDate = *v.Rule.EndDate
		}

		if v.Rule.MaxCount != nil {
			r.MaxCount = *v.Rule.MaxCount
		}

		got := r.Occurrences(v.From, v.To, 0)
		if !slices.Equal(got, append([]string(nil), v.Expect...)) && !(len(got) == 0 && len(v.Expect) == 0) {
			t.Errorf("%s: got %v, want %v", v.Name, got, v.Expect)
		}
	}
}

func TestRecurrenceJan31(t *testing.T) {
	t.Parallel()

	got := Rule{Frequency: FreqMonthly, Interval: 1, StartDate: "2027-01-31"}.Occurrences("2027-01-01", "2027-04-30", 0)
	want := []string{"2027-01-31", "2027-02-28", "2027-03-31", "2027-04-30"}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Listing two adjacent windows equals listing their union.
func TestRecurrenceWindowsCompose(t *testing.T) {
	t.Parallel()

	for _, r := range []Rule{
		{Frequency: FreqMonthly, Interval: 1, StartDate: "2027-01-31"},
		{Frequency: FreqWeekly, Interval: 3, StartDate: "2027-01-04", MaxCount: 30},
		{Frequency: FreqYearly, Interval: 1, StartDate: "2028-02-29"},
		{Frequency: FreqDaily, Interval: 7, StartDate: "2027-01-01", EndDate: "2028-06-01"},
	} {
		whole := r.Occurrences("2027-01-01", "2030-12-31", 0)
		parts := append(r.Occurrences("2027-01-01", "2028-02-28", 0), r.Occurrences("2028-02-29", "2030-12-31", 0)...)

		if !slices.Equal(whole, parts) {
			t.Errorf("%+v: whole %v != parts %v", r, whole, parts)
		}
	}
}

// The engine stops at the year 9999 and never walks without end.
func TestRecurrenceYearBound(t *testing.T) {
	t.Parallel()

	got := Rule{Frequency: FreqMonthly, Interval: 1, StartDate: "9999-11-30"}.Occurrences("9999-01-01", "9999-12-31", 0)
	if want := []string{"9999-11-30", "9999-12-30"}; !slices.Equal(got, want) {
		t.Errorf("monthly at the bound = %v, want %v", got, want)
	}

	if got := (Rule{Frequency: FreqYearly, Interval: 1, StartDate: "9999-02-28"}).Occurrences("9999-01-01", "9999-12-31", 0); len(got) != 1 {
		t.Errorf("yearly at the bound = %v, want one date", got)
	}

	// A five-digit year is not a date, whichever field carries it.
	for _, r := range []Rule{
		{Frequency: FreqDaily, Interval: 1, StartDate: "10000-01-01"},
		{Frequency: FreqDaily, Interval: 1, StartDate: "2027-01-01", EndDate: "10000-01-01"},
	} {
		if got := r.Occurrences("2027-01-01", "9999-12-31", 0); len(got) != 0 {
			t.Errorf("%+v = %v, want nothing", r, got)
		}
	}

	if got := (Rule{Frequency: FreqMonthly, Interval: 1, StartDate: "2027-01-01"}).Occurrences("2027-01-01", "10000-01-01", 0); len(got) != 0 {
		t.Errorf("a five-digit window = %v, want nothing", got)
	}

	// The widest window of a daily rule is cut at maxWalk, not at 3.6 million.
	wide := Rule{Frequency: FreqDaily, Interval: 1, StartDate: "0001-01-01"}.Occurrences("0001-01-01", "9999-12-31", 0)
	if len(wide) != maxWalk {
		t.Errorf("the widest daily window gave %d dates, want %d", len(wide), maxWalk)
	}

	if got := (Rule{Frequency: FreqDaily, Interval: 1, StartDate: "0001-01-01"}).Occurrences("9990-01-01", "9990-01-03", 0); len(got) != 3 {
		t.Errorf("a far window = %v, want 3 dates", got)
	}
}

// A window more than 292 years from the start (a time.Duration would saturate).
func TestRecurrenceVeryFarWindow(t *testing.T) {
	t.Parallel()

	got := Rule{Frequency: FreqDaily, Interval: 1, StartDate: "1900-01-01"}.Occurrences("2990-01-01", "2990-01-03", 0)
	if want := []string{"2990-01-01", "2990-01-02", "2990-01-03"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
