package finance

import (
	"slices"
	"testing"
	"time"
)

func TestInstallmentSplit(t *testing.T) {
	t.Parallel()

	if got := SplitInstallments(10000, 3); !slices.Equal(got, []int64{3334, 3333, 3333}) {
		t.Errorf("100.00 in 3 = %v, want 33.34, 33.33, 33.33", got)
	}

	for parts := 2; parts <= 12; parts++ {
		for total := int64(parts); total <= 500; total++ {
			var sum int64

			for i, c := range SplitInstallments(total, parts) {
				if c < 1 {
					t.Fatalf("total %d in %d: part %d is %d cents", total, parts, i, c)
				}

				sum += c
			}

			if sum != total {
				t.Fatalf("total %d in %d sums to %d", total, parts, sum)
			}
		}
	}
}

func TestInstallmentDates(t *testing.T) {
	t.Parallel()

	first := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)

	want := []string{"2027-01-31", "2027-02-28", "2027-03-31", "2027-04-30"}
	if got := InstallmentDates(first, 4); !slices.Equal(got, want) {
		t.Errorf("dates = %v, want %v", got, want)
	}
}
