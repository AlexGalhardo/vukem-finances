package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

// scenarioRunner runs CLI commands of one mode and returns the JSON they print,
// with what differs by nature between two installations (ids and clocks)
// replaced by a name.
type scenarioRunner struct {
	t   *testing.T
	cli *cli.CLI
	out *bytes.Buffer
	ids map[string]string
}

var (
	uidPattern  = regexp.MustCompile(`\b[0-9A-HJKMNP-TV-Z]{26}\b`)
	timePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z`)
)

func newScenarioRunner(t *testing.T, env map[string]string, local func(context.Context) (cli.Backend, error)) *scenarioRunner {
	t.Helper()

	tree, err := cli.DefaultTree()
	if err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}

	return &scenarioRunner{t: t, out: out, ids: map[string]string{}, cli: &cli.CLI{Tree: tree, Env: cli.Env{
		Stdin: strings.NewReader(""), Stdout: out, Stderr: &bytes.Buffer{}, OpenLocal: local,
		Getenv: func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	}}}
}

// do runs `resource verb args...`, expects success and returns the JSON as
// bytes with ids and times masked, plus the raw uid of an answer that has one.
func (r *scenarioRunner) do(args ...string) (masked, uid string) {
	r.t.Helper()

	r.out.Reset()

	if err := r.cli.Run(r.t.Context(), args[0], args[1:]); err != nil {
		r.t.Fatalf("%s: %v", strings.Join(args, " "), err)
	}

	var v map[string]any
	if err := json.Unmarshal(r.out.Bytes(), &v); err == nil {
		uid, _ = v["uid"].(string)
	}

	text := r.out.String()

	for _, id := range uidPattern.FindAllString(text, -1) {
		if _, ok := r.ids[id]; !ok {
			r.ids[id] = "ID" + string(rune('A'+len(r.ids)))
		}

		text = strings.ReplaceAll(text, id, r.ids[id])
	}

	return timePattern.ReplaceAllString(text, "TIME"), uid
}

// cliScenario is create, list, update and delete over accounts and transactions.
func cliScenario(r *scenarioRunner) []string {
	var steps []string

	step := func(args ...string) string {
		masked, uid := r.do(args...)
		steps = append(steps, strings.Join(args[:2], " ")+"\n"+masked)

		return uid
	}

	acc := step("accounts", "create", "--name", "Checking", "--kind", "checking", "--opening", "1.234,56")
	tx := step("transactions", "create", "--account-uid", acc, "--kind", "expense", "--amount", "12.50",
		"--date", "2026-11-21", "--description", "Lunch", "--tags", "food")

	step("transactions", "list")
	step("transactions", "update", tx, "--description", "Late lunch", "--amount-cents", "1300")
	step("transactions", "show", tx)
	step("transactions", "delete", tx, "--yes")
	step("transactions", "list")
	step("accounts", "update", acc, "--name", "Main")
	step("accounts", "show", acc)
	step("accounts", "list")

	return steps
}

// TestCLILocalAndRemoteAgree: the same scenario gives equal JSON in local and
// remote mode (epic 31, acceptance 3).
func TestCLILocalAndRemoteAgree(t *testing.T) {
	t.Parallel()

	localApp := newPersonalApp(t)
	signUp(t, localApp, "Ana", "ana@example.com")

	remoteApp := newPersonalApp(t)
	owner := signUp(t, remoteApp, "Ana", "ana@example.com")
	token := fullToken(t, remoteApp, owner)

	srv := httptest.NewServer(remoteApp.Handler())
	t.Cleanup(srv.Close)

	local := cliScenario(newScenarioRunner(t, nil, localBackend(localApp)))
	remote := cliScenario(newScenarioRunner(t, map[string]string{cli.EnvServer: srv.URL, cli.EnvToken: token}, nil))

	if len(local) != len(remote) || len(local) == 0 {
		t.Fatalf("local has %d steps, remote %d", len(local), len(remote))
	}

	if !strings.Contains(local[0], `"opening_cents": 123456`) || !strings.Contains(local[1], "Lunch") || !strings.Contains(local[1], "IDA") {
		t.Fatalf("the scenario did not do what it says:\n%s\n%s", local[0], local[1])
	}

	for i := range local {
		if local[i] != remote[i] {
			t.Errorf("step %d differs\nlocal:\n%s\nremote:\n%s", i+1, local[i], remote[i])
		}
	}
}

func localBackend(app *vukemtest.App) func(context.Context) (cli.Backend, error) {
	return func(ctx context.Context) (cli.Backend, error) {
		l, err := cli.NewLocal(ctx, app.App)
		if err != nil {
			return nil, err
		}

		return l, nil
	}
}
