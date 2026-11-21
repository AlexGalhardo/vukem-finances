package finance

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestImportQuotaCountsTheMonthsOfTheFile(t *testing.T) {
	t.Parallel()

	lim := workspaces.Limits{Plan: "test", Transactions: 10}

	if err := overQuota(lim, map[string]int{"2026-03": 7}, map[string]int{"2026-03": 3, "2026-04": 10}); err != nil {
		t.Errorf("exactly at the limit: %v", err)
	}

	if err := overQuota(lim, map[string]int{"2026-03": 8}, map[string]int{"2026-03": 3}); err == nil {
		t.Error("11 in a month under a limit of 10 must be refused")
	}

	if err := overQuota(workspaces.Pro, map[string]int{"2026-03": 1 << 30}, map[string]int{"2026-03": 1 << 30}); err != nil {
		t.Errorf("an unlimited plan: %v", err)
	}
}

func TestImportHashIsStable(t *testing.T) {
	t.Parallel()

	a := StatementRow{Date: "2026-03-01", Description: "  Coffee   SHOP", Cents: -350}
	b := StatementRow{Date: "2026-03-01", Description: "coffee shop", Cents: -350}

	if importHash("A", a) != importHash("A", b) {
		t.Error("case and spacing must not change the key")
	}

	for name, other := range map[string]string{"account": importHash("B", a)} {
		if other == importHash("A", a) {
			t.Errorf("the %s must be part of the key", name)
		}
	}

	c := a
	c.Cents = 350

	if importHash("A", c) == importHash("A", a) {
		t.Error("the sign is part of the key")
	}

	f := StatementRow{FITID: "F1", Date: "2026-03-01", Description: "x", Cents: 1}
	if h := importHash("A", f); len(h) != 70 || h[:6] != "fitid:" {
		t.Errorf("fitid key = %q, want fitid: and 64 hex characters", h)
	}

	g := f
	g.Description = "other"

	if importHash("A", f) != importHash("A", g) {
		t.Error("with a FITID the text does not matter")
	}
}

func TestSpreadsheetSafeDefusesFormulas(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"=HYPERLINK(\"http://x\")": "'=HYPERLINK(\"http://x\")",
		"+1+1":                     "'+1+1",
		"-2+3":                     "'-2+3",
		"@SUM(A1)":                 "'@SUM(A1)",
		"\t=1":                     "'\t=1",
		"\r=1":                     "'\r=1",
		"Coffee":                   "Coffee",
		"a=b":                      "a=b",
		"":                         "",
	} {
		if got := SpreadsheetSafe(in); got != want {
			t.Errorf("SpreadsheetSafe(%q) = %q, want %q", in, got, want)
		}
	}
}
