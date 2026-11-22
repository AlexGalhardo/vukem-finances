package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseBudgetsReadsTheP95Column(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "budgets.md")
	doc := "# Budgets\n\n| Scenario | What it does | p95 budget (ms) |\n| --- | --- | --- |\n| `dashboard` | the page | 300 |\n| `export_csv_50k` | the file | 1000 |\n"

	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := parseBudgets(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got["dashboard"].P95ms != 300 || got["export_csv_50k"].P95ms != 1000 {
		t.Errorf("budgets = %v", got)
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	r := result{}
	for i := 1; i <= 100; i++ {
		r.Samples = append(r.Samples, time.Duration(i)*time.Millisecond)
	}

	if got := r.percentile(0.95); got != 95*time.Millisecond {
		t.Errorf("p95 = %s, want 95ms", got)
	}

	if got := r.percentile(0.5); got != 50*time.Millisecond {
		t.Errorf("p50 = %s, want 50ms", got)
	}
}

func TestReportFailsOnAMissedBudget(t *testing.T) {
	t.Parallel()

	slow := result{Name: "dashboard", Samples: []time.Duration{time.Second}}
	if code := report([]result{slow}, map[string]budget{"dashboard": {P95ms: 300}}, 0, true); code != 1 {
		t.Errorf("code = %d, want 1 for a missed budget", code)
	}

	if code := report([]result{slow}, map[string]budget{"dashboard": {P95ms: 2000}}, 1, true); code != 1 {
		t.Errorf("code = %d, want 1 for a lock error", code)
	}

	if code := report([]result{slow}, map[string]budget{"dashboard": {P95ms: 2000}}, 0, true); code != 0 {
		t.Errorf("code = %d, want 0 inside the budget", code)
	}
}
