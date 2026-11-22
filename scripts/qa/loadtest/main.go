// Command loadtest measures the application under load (epic 24, task 10).
// It builds the binary, migrates a throwaway database, seeds one workspace of
// 100000 transactions and 50 of 2000, starts the server and drives it with 20
// concurrent clients through the dashboard, the transaction list, the reports,
// the CSV export, the API list, sync pull and push and the MCP tools, then
// compares each p95 with docs/qa/perf-budgets.md. With -check a missed budget,
// a failed request or a "database is locked" error exits non-zero.
//
// The database lives in a temporary directory that is removed on exit.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem-finances/scripts/qa/qakit"
)

type config struct {
	check      bool
	budgets    string
	bin        string
	bigRows    int
	workspaces int
	perRows    int
	clients    int
	requests   int
	keep       bool
}

func main() {
	var c config

	flag.BoolVar(&c.check, "check", false, "exit non-zero when a budget is missed or a request fails")
	flag.StringVar(&c.budgets, "budgets", "docs/qa/perf-budgets.md", "the budgets file")
	flag.StringVar(&c.bin, "bin", "", "a built app binary (default: build ./cmd/app)")
	flag.IntVar(&c.bigRows, "big", 100000, "transactions in the big workspace")
	flag.IntVar(&c.workspaces, "workspaces", 50, "small workspaces")
	flag.IntVar(&c.perRows, "per", 2000, "transactions in each small workspace")
	flag.IntVar(&c.clients, "clients", 20, "concurrent clients")
	flag.IntVar(&c.requests, "requests", 300, "requests per scenario")
	flag.BoolVar(&c.keep, "keep", false, "keep the database directory")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	code, err := run(ctx, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loadtest:", err)
		code = 2
	}

	os.Exit(code)
}

func run(ctx context.Context, c config) (int, error) {
	budgets, err := parseBudgets(c.budgets)
	if err != nil {
		return 0, err
	}

	fmt.Println("building the app")

	stack, err := qakit.NewStack(ctx, c.bin)
	if err != nil {
		return 0, err
	}

	if c.keep {
		fmt.Println("keeping", stack.Dir)
	} else {
		defer stack.Close()
	}

	if out, err := stack.Command(ctx, "migrate"); err != nil {
		return 0, fmt.Errorf("migrate: %w (%s)", err, out)
	}

	start := time.Now()

	fmt.Println("seeding", c.bigRows, "+", c.workspaces, "x", c.perRows, "transactions")

	workspaces, err := qakit.Seed(ctx, stack.DB, c.bigRows, c.workspaces, c.perRows, c.clients)
	if err != nil {
		return 0, fmt.Errorf("seed: %w", err)
	}

	fmt.Println("seeded in", time.Since(start).Round(time.Second))

	if err := stack.Start(ctx); err != nil {
		return 0, err
	}

	// The server is stopped by the PID this process started, and always.
	defer stack.Kill()

	h := &harness{cfg: c, base: stack.Base, ws: workspaces, bin: stack.Bin, env: stack.Env}

	results, err := h.runAll(ctx)
	if err != nil {
		return 0, err
	}

	// A lock error may also be logged by the server without reaching a client.
	locked := countIn(stack.LogPath(), "database is locked")

	return report(results, budgets, locked, c.check), nil
}

func report(results []result, budgets map[string]budget, serverLocked int, check bool) int {
	failed := false

	fmt.Printf("\n| scenario | requests | p50 (ms) | p95 (ms) | max (ms) | budget (ms) | failed | result |\n")
	fmt.Printf("| --- | --- | --- | --- | --- | --- | --- | --- |\n")

	for _, r := range results {
		b, hasBudget := budgets[r.Name]
		verdict := "ok"

		switch {
		case r.Failed > 0:
			verdict, failed = "FAILED REQUESTS", true
		case !hasBudget:
			verdict = "no budget"
			failed = true
		case ms(r.percentile(0.95)) > b.P95ms:
			verdict, failed = "OVER BUDGET", true
		}

		fmt.Printf("| `%s` | %d | %.1f | %.1f | %.1f | %.0f | %d | %s |\n", r.Name, len(r.Samples)+r.Failed,
			ms(r.percentile(0.5)), ms(r.percentile(0.95)), ms(r.percentile(1)), b.P95ms, r.Failed, verdict)

		for _, f := range r.Failures {
			fmt.Printf("|  | first failure: %s |\n", strings.ReplaceAll(f, "|", "/"))
		}
	}

	locked := serverLocked
	for _, r := range results {
		locked += r.Locked
	}

	fmt.Printf("\n\"database is locked\" errors: %d\n", locked)

	if locked > 0 {
		failed = true
	}

	if check && failed {
		fmt.Println("loadtest: FAILED")
		return 1
	}

	if failed {
		fmt.Println("loadtest: budgets or requests failed (no -check, exit 0)")
	} else {
		fmt.Println("loadtest: all budgets met")
	}

	return 0
}

func countIn(path, needle string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	return strings.Count(string(raw), needle)
}
