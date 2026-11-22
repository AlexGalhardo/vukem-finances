package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// result is what one scenario measured.
type result struct {
	Name     string
	Samples  []time.Duration
	Failures []string // the first few distinct failures
	Failed   int
	Locked   int
	Elapsed  time.Duration
}

type recorder struct {
	mu sync.Mutex
	r  result
}

func (rc *recorder) ok(d time.Duration) {
	rc.mu.Lock()
	rc.r.Samples = append(rc.r.Samples, d)
	rc.mu.Unlock()
}

func (rc *recorder) fail(msg string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.r.Failed++

	if strings.Contains(msg, "database is locked") {
		rc.r.Locked++
	}

	if len(rc.r.Failures) < 3 {
		rc.r.Failures = append(rc.r.Failures, msg)
	}
}

func (r result) percentile(p float64) time.Duration {
	if len(r.Samples) == 0 {
		return 0
	}

	s := append([]time.Duration(nil), r.Samples...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })

	i := int(float64(len(s))*p+0.9999999) - 1
	if i < 0 {
		i = 0
	}

	return s[min(i, len(s)-1)]
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// budget is the p95 allowed for one scenario, in milliseconds.
type budget struct {
	P95ms float64
}

// parseBudgets reads the table of docs/qa/perf-budgets.md: a row whose first
// cell is a backticked scenario name and whose column "p95 budget (ms)" holds
// a number.
func parseBudgets(path string) (map[string]budget, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out := map[string]budget{}
	col := -1
	cellSplit := regexp.MustCompile(`\s*\|\s*`)

	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}

		cells := cellSplit.Split(strings.Trim(line, "| "), -1)

		for i, c := range cells {
			if strings.EqualFold(c, "p95 budget (ms)") {
				col = i
			}
		}

		if col < 0 || col >= len(cells) || !strings.HasPrefix(cells[0], "`") {
			continue
		}

		v, err := strconv.ParseFloat(cells[col], 64)
		if err != nil {
			return nil, fmt.Errorf("budget of %s: %w", cells[0], err)
		}

		out[strings.Trim(cells[0], "`")] = budget{P95ms: v}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no budget rows found", path)
	}

	return out, nil
}
