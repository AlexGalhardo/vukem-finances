package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("not a JSON object: %v: %q", err, s)
	}

	return m
}

// fakeToken has the shape of a real token. It is built at run time so the
// secret scan keeps flagging every literal of that shape.
func fakeToken(id, secret string) string {
	return "vf_" + strings.Repeat(id, 26) + "." + strings.Repeat(secret, 26)
}

func TestRemoteEnvTokenNeverGoesToTheStoredServer(t *testing.T) {
	t.Parallel()

	hits := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits <- r.Header.Get("Authorization")

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	store := &cli.FileStore{Path: filepath.Join(t.TempDir(), "credentials.json")}
	if err := store.Save(cli.Credentials{Server: srv.URL, Token: fakeToken("C", "D")}); err != nil {
		t.Fatal(err)
	}

	c, tm := newCLI(t, map[string]string{cli.EnvToken: fakeToken("A", "B")}, nil)
	c.Env.Store = store

	if code := run(t, c, tm, "accounts", "list"); code != cli.ExitUsage {
		t.Fatalf("exit %d, want %d: %s", code, cli.ExitUsage, tm.err.String())
	}

	select {
	case <-hits:
		t.Fatal("the token of the environment reached the server of the stored login")
	default:
	}
}

func TestRemotePathParams(t *testing.T) {
	t.Parallel()

	paths := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.EscapedPath()

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	env := map[string]string{cli.EnvServer: srv.URL, cli.EnvToken: "vf_TEST.TOKEN"}

	c, tm := newCLI(t, env, nil)
	if code := run(t, c, tm, "accounts", "show", "a b/c"); code != cli.ExitNotFound {
		t.Fatalf("exit %d, want %d: %s", code, cli.ExitNotFound, tm.err.String())
	}

	if got, want := <-paths, "/api/v1/accounts/a%20b%2Fc"; got != want {
		t.Fatalf("path = %s, want %s (escaped once)", got, want)
	}

	for _, dots := range []string{".", ".."} {
		c, tm := newCLI(t, env, nil)
		if code := run(t, c, tm, "accounts", "show", dots); code != cli.ExitUsage {
			t.Fatalf("%q: exit %d, want %d", dots, code, cli.ExitUsage)
		}

		select {
		case got := <-paths:
			t.Fatalf("%q reached the server at %s", dots, got)
		default:
		}
	}
}

func TestRemote(t *testing.T) {
	t.Parallel()

	app := newEdition(t, "personal")
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := makeToken(t, app, ana)

	srv := httptest.NewServer(app.Handler())
	t.Cleanup(srv.Close)

	t.Run("a token from the environment runs the scenario", func(t *testing.T) {
		t.Parallel()

		c, tm := newCLI(t, map[string]string{cli.EnvServer: srv.URL, cli.EnvToken: token}, nil)

		if code := run(t, c, tm, "accounts", "create", "--name", "Checking", "--kind", "checking", "--opening", "10.50"); code != 0 {
			t.Fatalf("create exit %d: %s", code, tm.err.String())
		}

		created := decode(t, tm.out.String())
		if created["opening_cents"].(float64) != 1050 || created["name"] != "Checking" {
			t.Fatalf("created = %v", created)
		}

		uid := created["uid"].(string)

		if code := run(t, c, tm, "accounts", "update", uid, "--name", "Main"); code != 0 {
			t.Fatalf("update exit %d: %s", code, tm.err.String())
		}

		if code := run(t, c, tm, "accounts", "list"); code != 0 || !strings.Contains(tm.out.String(), "Main") {
			t.Fatalf("list exit %d: %s", code, tm.out.String())
		}

		if code := run(t, c, tm, "accounts", "show", "01JD5Z8K3M4N5P6Q7R8S9T0V1W"); code != cli.ExitNotFound {
			t.Fatalf("show of a uid that is not here: exit %d, want %d", code, cli.ExitNotFound)
		}

		if code := run(t, c, tm, "accounts", "create", "--name", "x"); code != cli.ExitUsage {
			t.Fatalf("create with no kind: exit %d, want %d (%s)", code, cli.ExitUsage, tm.err.String())
		}
	})

	t.Run("a wrong token is an auth failure and never printed", func(t *testing.T) {
		t.Parallel()

		bad := fakeToken("A", "B")
		c, tm := newCLI(t, map[string]string{cli.EnvServer: srv.URL, cli.EnvToken: bad}, nil)

		if code := run(t, c, tm, "accounts", "list"); code != cli.ExitAuth {
			t.Fatalf("exit %d, want %d", code, cli.ExitAuth)
		}

		if strings.Contains(tm.out.String()+tm.err.String(), bad) {
			t.Fatal("the token is in the output")
		}
	})

	t.Run("a token is not a flag", func(t *testing.T) {
		t.Parallel()

		c, tm := newCLI(t, map[string]string{cli.EnvServer: srv.URL}, nil)

		if code := run(t, c, tm, "accounts", "list", "--token", token); code != cli.ExitUsage {
			t.Fatalf("exit %d, want %d", code, cli.ExitUsage)
		}

		if strings.Contains(tm.out.String()+tm.err.String(), token) {
			t.Fatal("the token is echoed")
		}
	})

	t.Run("login keeps the token in a private file, logout removes it", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "cfg", "credentials.json")
		c, tm := newCLI(t, nil, nil)
		c.Env.Store = &cli.FileStore{Path: path}

		c.Env.Stdin = strings.NewReader(token + "\n")

		if code := run(t, c, tm, "auth", "login", "--server", srv.URL); code != 0 {
			t.Fatalf("login exit %d: %s", code, tm.err.String())
		}

		if strings.Contains(tm.out.String()+tm.err.String(), token) {
			t.Fatal("login printed the token")
		}

		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("credentials file: %v, %v: want 0600", info, err)
			}
		}

		if code := run(t, c, tm, "accounts", "list"); code != 0 {
			t.Fatalf("list with the stored login: exit %d: %s", code, tm.err.String())
		}

		if code := run(t, c, tm, "auth", "logout"); code != 0 {
			t.Fatalf("logout exit %d", code)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("the login file is still there: %v", err)
		}

		if code := run(t, c, tm, "auth", "login", "--server", srv.URL, "--token", token); code != cli.ExitUsage {
			t.Fatalf("login with --token: exit %d, want %d", code, cli.ExitUsage)
		}
	})

	t.Run("login with a bad token stores nothing", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "credentials.json")
		c, tm := newCLI(t, nil, nil)
		c.Env.Store = &cli.FileStore{Path: path}
		c.Env.Stdin = strings.NewReader(fakeToken("A", "B") + "\n")

		if code := run(t, c, tm, "auth", "login", "--server", srv.URL); code != cli.ExitAuth {
			t.Fatalf("exit %d, want %d", code, cli.ExitAuth)
		}

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("a login was kept: %v", err)
		}
	})
}

func TestRemoteServerRules(t *testing.T) {
	t.Parallel()

	for _, bad := range []string{"", "ftp://x.example", "http://example.com", "https://u:p@example.com", "example.com", "https://example.com/?a=b"} {
		if _, err := cli.ParseServer(bad); cli.ExitOf(err) != cli.ExitUsage {
			t.Errorf("ParseServer(%q) = %v, want a usage error", bad, err)
		}
	}

	for _, ok := range []string{"https://finances.example.com", "https://finances.example.com/", "http://127.0.0.1:8080", "http://localhost:3000"} {
		if _, err := cli.ParseServer(ok); err != nil {
			t.Errorf("ParseServer(%q) = %v", ok, err)
		}
	}
}

// A token is sent only where the person pointed it: a redirect is an error, and
// nothing a server says can print the token.
func TestRemoteNeverLeaksTheToken(t *testing.T) {
	t.Parallel()

	token := fakeToken("C", "D")

	var seen []string

	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
	}))
	t.Cleanup(other.Close)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tags" {
			http.Redirect(w, r, other.URL+"/steal", http.StatusFound)
			return
		}

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)

		_, _ = w.Write([]byte(`{"detail":"boom ` + r.Header.Get("Authorization") + `\u001b[2J"}`))
	}))
	t.Cleanup(srv.Close)

	c, tm := newCLI(t, map[string]string{cli.EnvServer: srv.URL, cli.EnvToken: token}, nil)

	if code := run(t, c, tm, "tags", "list"); code != cli.ExitFailure || len(seen) != 0 {
		t.Fatalf("redirect: exit %d, the other server saw %v", code, seen)
	}

	if code := run(t, c, tm, "accounts", "list"); code != cli.ExitFailure {
		t.Fatalf("exit %d, want %d", code, cli.ExitFailure)
	}

	got := tm.out.String() + tm.err.String()
	if strings.Contains(got, token) {
		t.Fatalf("the token reached the output: %q", got)
	}

	if err := cli.ExitOf(c.Run(t.Context(), "accounts", []string{"list"})); err != cli.ExitFailure {
		t.Fatalf("exit %d", err)
	}
}
