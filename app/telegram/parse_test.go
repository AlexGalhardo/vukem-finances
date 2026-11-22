package telegram_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
)

var today = time.Date(2026, time.November, 21, 0, 0, 0, 0, time.UTC)

func TestParseEntry(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in     string
		kind   string
		cents  int64
		desc   string
		date   string
		errIs  error
		reason string
	}{
		{in: "+3000 salary", kind: "income", cents: 300000, desc: "salary", date: "2026-11-21"},
		{in: "-12,50 almoço ontem", kind: "expense", cents: 1250, desc: "almoço", date: "2026-11-20"},
		{in: "1.234,56 rent", kind: "expense", cents: 123456, desc: "rent", date: "2026-11-21"},
		{in: "12.50 coffee", kind: "expense", cents: 1250, desc: "coffee", date: "2026-11-21"},
		{in: "12,5 coffee", kind: "expense", cents: 1250, desc: "coffee", date: "2026-11-21"},
		{in: "1,234.56 laptop", kind: "expense", cents: 123456, desc: "laptop", date: "2026-11-21"},
		{in: "5", kind: "expense", cents: 500, desc: "", date: "2026-11-21"},
		{in: "-5", kind: "expense", cents: 500, desc: "", date: "2026-11-21"},
		{in: "+5", kind: "income", cents: 500, desc: "", date: "2026-11-21"},
		{in: "+ 3000 salary", kind: "income", cents: 300000, desc: "salary", date: "2026-11-21"},
		{in: "- 8 bus", kind: "expense", cents: 800, desc: "bus", date: "2026-11-21"},
		{in: "$9.99 app", kind: "expense", cents: 999, desc: "app", date: "2026-11-21"},
		{in: "R$ 9,99 app", errIs: telegram.ErrNoAmount, reason: "a space splits the symbol from the digits"},
		{in: "R$9,99 app", kind: "expense", cents: 999, desc: "app", date: "2026-11-21"},
		{in: "0.01 tiny", kind: "expense", cents: 1, desc: "tiny", date: "2026-11-21"},
		{in: "10 lunch today", kind: "expense", cents: 1000, desc: "lunch", date: "2026-11-21"},
		{in: "10 lunch yesterday", kind: "expense", cents: 1000, desc: "lunch", date: "2026-11-20"},
		{in: "10 almoço hoje", kind: "expense", cents: 1000, desc: "almoço", date: "2026-11-21"},
		{in: "10 almuerzo ayer", kind: "expense", cents: 1000, desc: "almuerzo", date: "2026-11-20"},
		{in: "10 taxi 15/11", kind: "expense", cents: 1000, desc: "taxi", date: "2026-11-15"},
		{in: "10 taxi 5/1", kind: "expense", cents: 1000, desc: "taxi", date: "2026-01-05"},
		{in: "10 taxi 15/11/2025", kind: "expense", cents: 1000, desc: "taxi", date: "2025-11-15"},
		{in: "10 taxi 2026-10-31", kind: "expense", cents: 1000, desc: "taxi", date: "2026-10-31"},
		{in: "10 TODAY", kind: "expense", cents: 1000, desc: "", date: "2026-11-21"},
		{in: "10 ONTEM", kind: "expense", cents: 1000, desc: "", date: "2026-11-20"},
		{in: "  10    two   spaces  ", kind: "expense", cents: 1000, desc: "two spaces", date: "2026-11-21"},
		{in: "10 a\tb\nc", kind: "expense", cents: 1000, desc: "a b c", date: "2026-11-21"},
		{in: "10 yesterday coffee", kind: "expense", cents: 1000, desc: "yesterday coffee", date: "2026-11-21", reason: "the date word counts only at the end"},
		{in: "10 <b>x</b> & y", kind: "expense", cents: 1000, desc: "<b>x</b> & y", date: "2026-11-21", reason: "escaping is for output"},
		{in: "10 tip 31/02", errIs: telegram.ErrBadDate},
		{in: "10 tip 32/01", errIs: telegram.ErrBadDate},
		{in: "10 tip 15/13", errIs: telegram.ErrBadDate},
		{in: "10 tip 2026-02-30", errIs: telegram.ErrBadDate},
		{in: "", errIs: telegram.ErrEmpty},
		{in: "   \n ", errIs: telegram.ErrEmpty},
		{in: "lunch 12", errIs: telegram.ErrNoAmount},
		{in: "+", errIs: telegram.ErrNoAmount},
		{in: "-", errIs: telegram.ErrNoAmount},
		{in: "12abc", errIs: telegram.ErrNoAmount, reason: "ParseAmount alone would read 12"},
		{in: "abc12", errIs: telegram.ErrNoAmount},
		{in: "0", errIs: telegram.ErrBadAmount},
		{in: "0,00 free", errIs: telegram.ErrBadAmount},
		{in: "--5 x", errIs: telegram.ErrNoAmount},
		{in: "+-5 x", errIs: telegram.ErrNoAmount},
		{in: "1.234.56 x", errIs: telegram.ErrBadAmount},
		{in: "1.5e3 x", errIs: telegram.ErrNoAmount},
		{in: "12,345 x", errIs: telegram.ErrBadAmount, reason: "three decimals are an error, not a rounding"},
		{in: "99999999999999999999 x", errIs: telegram.ErrBadAmount},
		{in: "90071992547409.92 x", errIs: telegram.ErrBadAmount, reason: "beyond 2^53-1 cents"},
		{in: "90071992547409.91 x", kind: "expense", cents: 9007199254740991, desc: "x", date: "2026-11-21"},
	}

	for _, tc := range cases {
		got, err := telegram.ParseEntry(tc.in, today)

		if tc.errIs != nil {
			if !errors.Is(err, tc.errIs) {
				t.Errorf("ParseEntry(%q) error = %v, want %v", tc.in, err, tc.errIs)
			}

			continue
		}

		if err != nil {
			t.Errorf("ParseEntry(%q) error = %v", tc.in, err)

			continue
		}

		if got.Kind != tc.kind || got.Cents != tc.cents || got.Description != tc.desc || got.Date != tc.date {
			t.Errorf("ParseEntry(%q) = %+v, want %s %d %q %s", tc.in, got, tc.kind, tc.cents, tc.desc, tc.date)
		}
	}

	if len(cases) < 30 {
		t.Fatalf("only %d cases", len(cases))
	}
}

func TestParseUsesParseAmount(t *testing.T) {
	t.Parallel()

	got, err := telegram.ParseEntry("1.234,56", today)
	if err != nil || got.Cents != 123456 {
		t.Fatalf("1.234,56 = %+v, %v; want 123456 cents", got, err)
	}

	src, err := os.ReadFile("parse.go")
	if err != nil {
		t.Fatal(err)
	}

	for _, banned := range []string{"ParseFloat", "float64", "Atoi(first", "ParseInt"} {
		if strings.Contains(string(src), banned) {
			t.Errorf("parse.go uses %s: money is read by finance.ParseAmount only", banned)
		}
	}

	if !strings.Contains(string(src), "finance.ParseAmount") {
		t.Error("parse.go does not call finance.ParseAmount")
	}
}

func TestParseLimits(t *testing.T) {
	t.Parallel()

	atLimit := "10 " + strings.Repeat("é", 497)
	if _, err := telegram.ParseEntry(atLimit, today); err != nil {
		t.Errorf("500 characters: %v", err)
	}

	if _, err := telegram.ParseEntry(atLimit+"x", today); !errors.Is(err, telegram.ErrTooLong) {
		t.Errorf("501 characters: %v, want ErrTooLong", err)
	}

	if _, err := telegram.ParseEntry(strings.Repeat("9", 100000), today); !errors.Is(err, telegram.ErrTooLong) {
		t.Errorf("a huge message: %v, want ErrTooLong", err)
	}
}
