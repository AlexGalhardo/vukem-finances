package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// TestParseAmount reads testdata/amount-vectors.json, the file the mobile
// parser (mobile/src/domain/money.ts) is tested against too.
func TestParseAmount(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../testdata/amount-vectors.json")
	if err != nil {
		t.Fatal(err)
	}

	var vectors []struct {
		In    string `json:"in"`
		Cents *int64 `json:"cents"`
	}

	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}

	if len(vectors) < 60 {
		t.Fatalf("only %d amount vectors, want at least 60", len(vectors))
	}

	for _, tc := range vectors {
		got, err := finance.ParseAmount(tc.In)
		if tc.Cents == nil {
			if err == nil {
				t.Errorf("ParseAmount(%q) = %d, want an error", tc.In, got)
			}

			continue
		}

		if err != nil || got != *tc.Cents {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d", tc.In, got, err, *tc.Cents)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		cents    int64
		currency string
		want     string
	}{
		{123456, "USD", "$1,234.56"},
		{5, "USD", "$0.05"},
		{-500, "BRL", "-R$5.00"},
		{100000000, "EUR", "€1,000,000.00"},
		{1000, "CHF", "CHF 10.00"},
	} {
		if got := finance.FormatAmount(tc.cents, tc.currency); got != tc.want {
			t.Errorf("FormatAmount(%d, %s) = %q, want %q", tc.cents, tc.currency, got, tc.want)
		}
	}
}
