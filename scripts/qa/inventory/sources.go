package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Entry is one thing the product exposes and a test must cover.
type Entry struct {
	Kind   string
	Name   string
	Detail string
	// Cover is what the index says covers it, Mapped what the mapping file adds.
	Cover  []unit
	Mapped []string
}

// ID is the key of an entry in the mapping file.
func (e Entry) ID() string { return e.Kind + ":" + e.Name }

// Covered reports whether at least one test covers the entry.
func (e Entry) Covered() bool { return len(e.Cover) > 0 || len(e.Mapped) > 0 }

// Roles are the workspace roles an authorization case runs as.
var Roles = []string{"owner", "admin", "member", "viewer"}

type operation struct {
	Name      string      `json:"name"`
	Scope     string      `json:"scope"`
	Ability   string      `json:"ability"`
	Kind      string      `json:"kind"`
	Surfaces  []opSurface `json:"surfaces"`
	CLI, Tool string      `json:"-"`
}

type opSurface struct {
	Surface string `json:"surface"`
	Offered bool   `json:"offered"`
	Route   string `json:"route"`
}

// operations reads the committed inventory of the operation registry, and the
// CLI command and MCP tool of each from the generated references.
func operations(root string) ([]operation, error) {
	raw, err := os.ReadFile(filepath.Join(root, "resources", "operations.json"))
	if err != nil {
		return nil, err
	}

	var doc struct {
		Operations []operation `json:"operations"`
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("reading resources/operations.json: %w", err)
	}

	cli, err := referenceNames(filepath.Join(root, "docs", "reference", "cli.md"), regexp.MustCompile("^### `([^`]+)`"), "- Operation: `")
	if err != nil {
		return nil, err
	}

	tools := map[string]string{}

	mcpDoc, err := os.ReadFile(filepath.Join(root, "docs", "reference", "mcp.md"))
	if err != nil {
		return nil, err
	}

	row := regexp.MustCompile("^\\| `([a-z_]+)` \\| `([a-z.]+)` \\|")
	for _, line := range strings.Split(string(mcpDoc), "\n") {
		if m := row.FindStringSubmatch(line); m != nil {
			tools[m[2]] = m[1]
		}
	}

	out := make([]operation, 0, len(doc.Operations))

	for _, op := range doc.Operations {
		op.CLI = cli[op.Name]
		op.Tool = tools[op.Name]
		out = append(out, op)
	}

	return out, nil
}

// referenceNames maps the operation named after prefix to the last heading
// that matched head before it.
func referenceNames(path string, head *regexp.Regexp, prefix string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	current := ""

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")

		if m := head.FindStringSubmatch(line); m != nil {
			current = m[1]
		}

		if rest, ok := strings.CutPrefix(line, prefix); ok && current != "" {
			name, _, _ := strings.Cut(rest, "`")
			out[name] = current
		}
	}

	return out, nil
}

// offered lists the surfaces an operation is offered on.
func (o operation) offered() []string {
	var out []string

	for _, s := range o.Surfaces {
		if s.Offered {
			out = append(out, s.Surface)
		}
	}

	return out
}

func (o operation) route() string {
	for _, s := range o.Surfaces {
		if s.Surface == "rest" {
			return s.Route
		}
	}

	return ""
}

// secret returns a random value, so no fake token is a literal in the source.
func secret(prefix string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)

	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

// appRun builds the app once and runs its console commands.
type appRun struct {
	bin string
	dir string
}

func buildApp(root string) (*appRun, func(), error) {
	dir, err := os.MkdirTemp("", "inventory-")
	if err != nil {
		return nil, nil, err
	}

	bin := filepath.Join(dir, "app.exe")

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/app")
	cmd.Dir = root

	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir)

		return nil, nil, fmt.Errorf("building the app: %w\n%s", err, out)
	}

	return &appRun{bin: bin, dir: root}, func() { _ = os.RemoveAll(dir) }, nil
}

// console runs `app <args>` for an edition with throwaway settings: nothing
// here reads or writes real data, and the keys are made at run time.
func (a *appRun) console(edition string, args ...string) (string, error) {
	cmd := exec.Command(a.bin, args...)
	cmd.Dir = a.dir
	cmd.Env = append(os.Environ(),
		"APP_KEY=base64:"+base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)),
		"APP_EDITION="+edition,
		"APP_URL=https://inventory.invalid",
		"DB_DATABASE="+filepath.Join(os.TempDir(), "inventory-unused.sqlite"),
		"STRIPE_KEY="+secret("rk_test_"),
		"STRIPE_WEBHOOK_SECRET="+secret("whsec_"),
		"STRIPE_PRICE_PRO="+secret("price_"),
	)

	var stdout, stderr bytes.Buffer

	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", edition, strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}

type route struct {
	Method, Path, Name string
	Editions           []string
}

// routes lists the routes of both editions, joined by method and path.
func (a *appRun) routes() ([]route, error) {
	byKey := map[string]*route{}

	for _, edition := range []string{"personal", "saas"} {
		out, err := a.console(edition, "route:list")
		if err != nil {
			return nil, err
		}

		for i, line := range strings.Split(out, "\n") {
			f := strings.Fields(line)
			if i == 0 || len(f) < 3 || !strings.HasPrefix(f[1], "/") {
				continue
			}

			key := f[0] + " " + f[1]
			if byKey[key] == nil {
				byKey[key] = &route{Method: f[0], Path: f[1], Name: f[2]}
			}

			byKey[key].Editions = append(byKey[key].Editions, edition)
		}
	}

	out := make([]route, 0, len(byKey))
	for _, r := range byKey {
		out = append(out, *r)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}

		return out[i].Method < out[j].Method
	})

	return out, nil
}

// tasks lists the scheduled tasks by name and cron expression.
func (a *appRun) tasks() ([][2]string, error) {
	out, err := a.console("personal", "schedule:list")
	if err != nil {
		return nil, err
	}

	var tasks [][2]string

	for i, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if i == 0 || len(f) < 6 {
			continue
		}

		tasks = append(tasks, [2]string{f[0], strings.Join(f[1:6], " ")})
	}

	return tasks, nil
}

var (
	jobKind     = regexp.MustCompile(`func \((\w+)\) Kind\(\) string \{ return "(\w+)" \}`)
	webhookPath = regexp.MustCompile(`(?m)^\s*(?:const )?(\w*WebhookPath)\s*=\s*"(/[^"]+)"`)
	botCommand  = regexp.MustCompile(`(?:case|==)\s*"(/[a-z]+)"`)
)

type job struct{ Type, Kind string }

func jobs(root string) ([]job, error) {
	files, err := filepath.Glob(filepath.Join(root, "app", "jobs", "*.go"))
	if err != nil {
		return nil, err
	}

	var out []job

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}

		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}

		for _, m := range jobKind.FindAllStringSubmatch(string(raw), -1) {
			out = append(out, job{Type: m[1], Kind: m[2]})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })

	return out, nil
}

// webhooks reads the endpoints declared as `...WebhookPath` constants; the
// ones that take the secret in the path (`/{secret}`) are listed twice.
func webhooks(root string) ([]string, error) {
	var out []string

	for _, dir := range []string{"app", "config"} {
		err := walkGo(filepath.Join(root, dir), func(path string, raw []byte) {
			for _, m := range webhookPath.FindAllStringSubmatch(string(raw), -1) {
				out = append(out, m[2])

				if strings.Contains(string(raw), m[1]+` + "/{secret}"`) || strings.Contains(string(raw), m[1]+`+"/{secret}"`) {
					out = append(out, m[2]+"/{secret}")
				}
			}
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(out)

	return out, nil
}

func walkGo(dir string, fn func(path string, raw []byte)) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fn(path, raw)

		return nil
	})
}

// telegramCommands reads the commands the bot dispatches on.
func telegramCommands(root string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "app", "telegram", "bot.go"))
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}

	for _, m := range botCommand.FindAllStringSubmatch(string(raw), -1) {
		seen[m[1]] = true
	}

	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}

	sort.Strings(out)

	return out, nil
}

// pages lists the page components, named as the server renders them.
func pages(root string) ([]string, error) {
	base := filepath.Join(root, "resources", "js", "pages")

	var out []string

	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tsx") {
			return err
		}

		rel, _ := filepath.Rel(base, path)
		out = append(out, strings.TrimSuffix(filepath.ToSlash(rel), ".tsx"))

		return nil
	})

	sort.Strings(out)

	return out, err
}
