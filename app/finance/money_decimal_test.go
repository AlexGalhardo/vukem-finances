package finance

import (
	"math"
	"testing"
)

func TestFormatDecimalCents(t *testing.T) {
	t.Parallel()

	check := func(cents int64) {
		t.Helper()

		got := FormatDecimalCents(cents)
		if got == "-0.00" {
			t.Fatalf("FormatDecimalCents(%d) = -0.00", cents)
		}

		dot := len(got) - 3
		if dot < 1 || got[dot] != '.' {
			t.Fatalf("FormatDecimalCents(%d) = %q, want two decimals after a dot", cents, got)
		}

		back, err := ParseAmount(got)
		if err != nil || back != cents {
			t.Fatalf("ParseAmount(%q) = %d, %v, want %d", got, back, err, cents)
		}
	}

	for c := int64(-100000); c <= 100000; c++ {
		check(c)
	}

	for c, want := range map[int64]string{
		0: "0.00", 5: "0.05", -5: "-0.05", 1234: "12.34", -1234: "-12.34", 100: "1.00", -100000: "-1000.00",
		math.MaxInt64:    "92233720368547758.07",
		math.MinInt64:    "-92233720368547758.08",
		9007199254740991: "90071992547409.91",
	} {
		if got := FormatDecimalCents(c); got != want {
			t.Errorf("FormatDecimalCents(%d) = %q, want %q", c, got, want)
		}
	}
}
