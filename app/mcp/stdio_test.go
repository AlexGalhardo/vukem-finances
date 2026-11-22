package mcp_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/config"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// runMain runs `app <args>` the way cmd/app does, with its own streams and a
// private database, and returns what it wrote.
func runMain(t *testing.T, env map[string]string, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errOut bytes.Buffer

	getenv := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	err = vukem.RunMain(t.Context(), append([]string{"app"}, args...), getenv, bootstrap.New,
		vukem.WithStdin(strings.NewReader(stdin)), vukem.WithStdout(&out), vukem.WithStderr(&errOut),
		vukem.WithLogOutput(&errOut))

	return out.String(), errOut.String(), err
}

func commandEnv(t *testing.T, edition string) map[string]string {
	t.Helper()

	return map[string]string{
		"APP_ENV": "test", "APP_KEY": config.GenerateKey(), "APP_EDITION": edition,
		"DATABASE_URL":   "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "app.sqlite")),
		"SESSION_DRIVER": "cookie", "MAIL_MAILER": "array",
	}
}

const meta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`

func TestStdoutOnlyProtocol(t *testing.T) {
	t.Parallel()

	env := commandEnv(t, "personal")

	if _, _, err := runMain(t, env, "", "migrate"); err != nil {
		t.Fatal(err)
	}

	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{` + meta + `}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{` + meta + `}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"accounts_list","arguments":{},` + meta + `}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{},` + meta + `}}`,
		`this is not json`,
	}, "\n") + "\n"

	stdout, stderr, err := runMain(t, env, requests, "mcp:serve", "--stdio")
	if err != nil {
		t.Fatalf("mcp:serve: %v\n%s", err, stderr)
	}

	sc := bufio.NewScanner(strings.NewReader(stdout))
	sc.Buffer(nil, 4<<20)

	lines := 0

	for sc.Scan() {
		lines++

		var m struct {
			JSONRPC string `json:"jsonrpc"`
		}

		if json.Unmarshal(sc.Bytes(), &m) != nil || m.JSONRPC != "2.0" {
			t.Fatalf("stdout line %d is not a protocol message: %q", lines, sc.Text())
		}
	}

	if lines < 4 {
		t.Fatalf("only %d protocol messages on stdout; stderr: %s", lines, stderr)
	}
}

func TestEditionRefusal(t *testing.T) {
	t.Parallel()

	env := commandEnv(t, "saas")

	if _, _, err := runMain(t, env, "", "migrate"); err != nil {
		t.Fatal(err)
	}

	requests := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{` + meta + `}}` + "\n"

	stdout, _, err := runMain(t, env, requests, "mcp:serve", "--stdio")
	if err == nil {
		t.Fatal("mcp:serve --stdio served in the saas edition")
	}

	if !errors.Is(err, appmcp.ErrNotPersonal) {
		t.Fatalf("err = %v, want ErrNotPersonal", err)
	}

	if stdout != "" {
		t.Fatalf("the refusal wrote to stdout: %q", stdout)
	}

	t.Run("without --stdio it is a usage error", func(t *testing.T) {
		_, _, err := runMain(t, commandEnv(t, "personal"), "", "mcp:serve")
		if !errors.Is(err, vukem.ErrUsage) {
			t.Fatalf("err = %v, want a usage error", err)
		}
	})
}
