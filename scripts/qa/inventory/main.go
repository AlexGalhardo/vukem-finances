// Command inventory writes docs/qa/inventory.md: everything the product
// exposes (operations with their REST path, CLI command and MCP tool, web
// routes, API and sync routes, page components, Telegram commands, queue jobs,
// scheduled tasks and webhook endpoints) and the test that covers each, so
// coverage is counted from the code and not from memory.
//
//	go run ./scripts/qa/inventory           # rewrite docs/qa/inventory.md
//	go run ./scripts/qa/inventory -check    # fail on an entry with no test, or on a stale file
//	go run ./scripts/qa/inventory -check -hide TestName   # scratch run without one test
//
// A test covers an entry when it names it (see index.go and entries below);
// scripts/qa/inventory/mapping.json adds the coverage a scan cannot see, and
// every test it names must exist.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("inventory", flag.ContinueOnError)
	check := fs.Bool("check", false, "fail when an entry has no covering test or docs/qa/inventory.md is stale")
	root := fs.String("root", ".", "repository root")
	hide := fs.String("hide", "", "comma separated test names or files to leave out (a scratch run of -check)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	hidden := map[string]bool{}

	for _, h := range strings.Split(*hide, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hidden[h] = true
		}
	}

	entries, ops, err := collect(*root, hidden)
	if err != nil {
		fmt.Fprintln(os.Stderr, "inventory:", err)

		return 2
	}

	doc := render(entries, ops)
	out := filepath.Join(*root, "docs", "qa", "inventory.md")

	if !*check {
		if err := os.WriteFile(out, []byte(doc), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "inventory:", err)

			return 2
		}

		fmt.Printf("inventory: %d entries, %d uncovered\n", len(entries), len(uncovered(entries)))

		return 0
	}

	failed := false

	for _, e := range uncovered(entries) {
		fmt.Fprintf(os.Stderr, "no covering test: %s (%s)\n", e.ID(), e.Detail)

		failed = true
	}

	if len(hidden) == 0 {
		if have, err := os.ReadFile(out); err != nil || strings.ReplaceAll(string(have), "\r\n", "\n") != doc {
			fmt.Fprintln(os.Stderr, "docs/qa/inventory.md is stale: run `go run ./scripts/qa/inventory`")

			failed = true
		}
	}

	if failed {
		return 1
	}

	fmt.Printf("inventory: %d entries, all covered\n", len(entries))

	return 0
}

func uncovered(entries []Entry) []Entry {
	var out []Entry

	for _, e := range entries {
		if !e.Covered() {
			out = append(out, e)
		}
	}

	return out
}

// collect builds every entry and finds the tests that cover it.
func collect(root string, hidden map[string]bool) ([]Entry, []operation, error) {
	ix, err := buildIndex(root, hidden)
	if err != nil {
		return nil, nil, err
	}

	ops, err := operations(root)
	if err != nil {
		return nil, nil, err
	}

	app, cleanup, err := buildApp(root)
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	routes, err := app.routes()
	if err != nil {
		return nil, nil, err
	}

	tasks, err := app.tasks()
	if err != nil {
		return nil, nil, err
	}

	var entries []Entry

	opRoutes := map[string]bool{}

	for _, o := range ops {
		opRoutes["api."+o.Name] = true

		detail := strings.Join(o.offered(), ", ")
		entries = append(entries, Entry{Kind: "operation", Name: o.Name, Detail: detail, Cover: ix.coverString(o.Name)})
	}

	for _, r := range routes {
		if opRoutes[r.Name] {
			continue
		}

		kind := "route"

		switch {
		case strings.HasPrefix(r.Path, "/api/v1/sync/"):
			kind = "sync"
		case strings.HasPrefix(r.Path, "/api/v1"):
			kind = "api"
		}

		entries = append(entries, Entry{
			Kind: kind, Name: r.Method + " " + r.Path, Detail: r.Name + " (" + strings.Join(r.Editions, ", ") + ")",
			Cover: ix.coverRequest(r.Method, r.Path),
		})
	}

	hooks, err := webhooks(root)
	if err != nil {
		return nil, nil, err
	}

	for _, h := range hooks {
		entries = append(entries, Entry{Kind: "webhook", Name: "POST " + h, Detail: "provider callback", Cover: ix.coverRequest("POST", h)})
	}

	comps, err := pages(root)
	if err != nil {
		return nil, nil, err
	}

	for _, c := range comps {
		entries = append(entries, Entry{Kind: "page", Name: c, Detail: "resources/js/pages/" + c + ".tsx", Cover: ix.coverString(c)})
	}

	cmds, err := telegramCommands(root)
	if err != nil {
		return nil, nil, err
	}

	for _, c := range cmds {
		entries = append(entries, Entry{Kind: "telegram", Name: c, Detail: "bot command", Cover: ix.coverCommand(c)})
	}

	js, err := jobs(root)
	if err != nil {
		return nil, nil, err
	}

	kinds := map[string]job{}

	for _, j := range js {
		kinds[j.Kind] = j
		cover := slices.Concat(ix.coverString(j.Kind), ix.coverIdent("Handle"+j.Type))
		entries = append(entries, Entry{Kind: "job", Name: j.Kind, Detail: "queue job " + j.Type, Cover: cover})
	}

	for _, t := range tasks {
		name, expr := t[0], t[1]
		cover := ix.coverString(name)

		if kind, ok := strings.CutPrefix(name, "job:"); ok {
			for _, u := range slices.Concat(ix.coverString(kind), ix.coverIdent(kinds[kind].Type)) {
				if ix.scheduleFiles[u.file] {
					cover = append(cover, u)
				}
			}
		}

		entries = append(entries, Entry{Kind: "schedule", Name: name, Detail: "`" + expr + "`", Cover: cover})
	}

	if err := applyMapping(root, ix, entries); err != nil {
		return nil, nil, err
	}

	return entries, ops, nil
}

// applyMapping adds the tests named in mapping.json, and refuses a test that
// does not exist or an entry that is not in the inventory.
func applyMapping(root string, ix *index, entries []Entry) error {
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "qa", "inventory", "mapping.json"))
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	var mapping map[string][]string
	if err := json.Unmarshal(raw, &mapping); err != nil {
		return fmt.Errorf("reading mapping.json: %w", err)
	}

	known := map[string]int{}
	for i, e := range entries {
		known[e.ID()] = i
	}

	for id, tests := range mapping {
		i, ok := known[id]
		if !ok {
			return fmt.Errorf("mapping.json names %q, which is not in the inventory", id)
		}

		for _, t := range tests {
			if !ix.tests[t] {
				// A test hidden by -hide is simply gone.
				continue
			}

			entries[i].Mapped = append(entries[i].Mapped, t)
		}
	}

	return nil
}

func render(entries []Entry, ops []operation) string {
	var b strings.Builder

	b.WriteString("# QA inventory\n\n")
	b.WriteString("Generated by `go run ./scripts/qa/inventory`; do not edit. `go run ./scripts/qa/inventory -check` fails when an entry has no covering test or this file is stale.\n\n")
	b.WriteString("A test covers an entry when it names it: the operation name, the HTTP method and path it requests, the page component, the command it sends the bot, the job kind. `scripts/qa/inventory/mapping.json` adds the tests a scan cannot see.\n\n")

	cases := 0
	pairs := 0

	for _, o := range ops {
		pairs += len(o.offered())
	}

	cases = pairs * len(Roles)

	fmt.Fprintf(&b, "## Authorization matrix\n\n")
	fmt.Fprintf(&b, "Operations: %d. Operation and surface pairs (rest, cli, mcp, as offered): %d. Roles: %d (%s).\n\n", len(ops), pairs, len(Roles), strings.Join(Roles, ", "))
	fmt.Fprintf(&b, "Authorization cases: %d\n\n", cases)

	b.WriteString("## Operations\n\n| Operation | Kind | Scope | Ability | REST | CLI | MCP tool | Covered by |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")

	byName := map[string]Entry{}

	for _, e := range entries {
		if e.Kind == "operation" {
			byName[e.Name] = e
		}
	}

	for _, o := range ops {
		cli, tool := cell(o.CLI), cell(o.Tool)
		if !slices.Contains(o.offered(), "cli") {
			cli = "-"
		}

		if !slices.Contains(o.offered(), "mcp") {
			tool = "-"
		}

		fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` | %s | %s | %s | %s |\n",
			o.Name, o.Kind, o.Scope, o.Ability, cell(o.route()), cli, tool, covered(byName[o.Name]))
	}

	titles := []struct{ kind, title string }{
		{"route", "Web routes"}, {"api", "API routes outside the registry"}, {"sync", "Sync routes"},
		{"webhook", "Webhook endpoints"}, {"page", "Page components"}, {"telegram", "Telegram commands"},
		{"job", "Queue jobs"}, {"schedule", "Scheduled tasks"},
	}

	for _, t := range titles {
		fmt.Fprintf(&b, "\n## %s\n\n| Entry | Detail | Covered by |\n| --- | --- | --- |\n", t.title)

		for _, e := range entries {
			if e.Kind == t.kind {
				fmt.Fprintf(&b, "| `%s` | %s | %s |\n", e.Name, strings.ReplaceAll(e.Detail, "|", "/"), covered(e))
			}
		}
	}

	return b.String()
}

func cell(s string) string {
	if s == "" {
		return "-"
	}

	return "`" + s + "`"
}

// covered names the tests of an entry: up to three, then a count.
func covered(e Entry) string {
	var names []string

	seen := map[string]bool{}

	for _, u := range e.Cover {
		if n := u.name(); !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}

	for _, n := range e.Mapped {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}

	if len(names) == 0 {
		return "**none**"
	}

	slices.Sort(names)

	if len(names) > 3 {
		return fmt.Sprintf("`%s` and %d more", strings.Join(names[:3], "`, `"), len(names)-3)
	}

	return "`" + strings.Join(names, "`, `") + "`"
}
