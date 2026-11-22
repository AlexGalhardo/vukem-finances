package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// The environment variables of remote mode, for scripts and CI. The token is
// never a flag: a flag lands in the shell history and in the process list.
const (
	EnvToken  = "FINANCES_TOKEN"
	EnvServer = "FINANCES_SERVER"
)

func (c *CLI) env(key string) string {
	if c.Env.Getenv == nil {
		return ""
	}

	v, _ := c.Env.Getenv(key)

	return strings.TrimSpace(v)
}

// backend picks the mode. --local runs here; a server (--server, FINANCES_SERVER
// or the one of `auth login`) runs there, with FINANCES_TOKEN or the stored
// token; with neither the CLI runs locally.
func (c *CLI) backend(ctx context.Context, g *globals) (Backend, error) {
	if g.local && g.server != "" {
		return nil, usagef("use --local or --server, not both")
	}

	if g.local {
		return c.localBackend(ctx)
	}

	server, token, err := c.target(g.server)
	if err != nil {
		return nil, err
	}

	if server == "" {
		if token != "" {
			return nil, usagef("%s needs a server: set %s, pass --server, or run `auth login --server URL`", EnvToken, EnvServer)
		}

		return c.localBackend(ctx)
	}

	u, err := ParseServer(server)
	if err != nil {
		return nil, err
	}

	if token == "" {
		return nil, &Failure{Code: ExitAuth, Msg: "no token for " + u.String() + ": run `auth login --server " + u.String() + "` or set " + EnvToken}
	}

	return NewRemote(u, token, c.Env.HTTP), nil
}

// target is the server and token the call should use; either may be empty.
func (c *CLI) target(flagServer string) (server, token string, err error) {
	server = flagServer
	if server == "" {
		server = c.env(EnvServer)
	}

	token = c.env(EnvToken)

	if c.Env.Store == nil {
		return server, token, nil
	}

	saved, ok, err := c.Env.Store.Load()
	if err != nil {
		return "", "", err
	}

	if !ok {
		return server, token, nil
	}

	// A token from the environment never goes to the server of a stored login:
	// whoever sets it must also say where it goes.
	if server == "" && token == "" {
		server = saved.Server
	}

	if token == "" && sameServer(server, saved.Server) {
		token = saved.Token
	}

	return server, token, nil
}

func sameServer(a, b string) bool {
	ua, errA := ParseServer(a)
	ub, errB := ParseServer(b)

	return errA == nil && errB == nil && ua.String() == ub.String()
}

// Auth runs `auth login|logout|status`.
func (c *CLI) Auth(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(c.Env.Stdout, "Usage: auth <verb>\n\nVerbs:\n  login --server URL   store a token (read from the standard input) for a server\n  logout               forget the stored token\n  status               show the server and workspace of the stored login")

		if len(args) == 0 {
			return usagef("auth needs a verb")
		}

		return nil
	}

	switch args[0] {
	case "login":
		return c.login(ctx, args[1:])
	case "logout":
		return c.logout()
	case "status":
		return c.status(ctx)
	}

	return usagef("unknown command %q: auth has login, logout and status", "auth "+args[0])
}

func (c *CLI) login(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	server := fs.String("server", "", "")

	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		return usagef("usage: auth login --server URL (the token is read from the standard input; it is never a flag)")
	}

	if *server == "" {
		*server = c.env(EnvServer)
	}

	u, err := ParseServer(*server)
	if err != nil {
		return err
	}

	if c.Env.Store == nil {
		return errors.New("no place to keep the login on this system")
	}

	if c.Env.StdinTTY {
		fmt.Fprintln(c.Env.Stderr, "Paste the token and press Enter (the input is shown; to avoid that, pipe it in: `... | auth login`).")
	}

	line, err := bufio.NewReader(io.LimitReader(c.Env.Stdin, 4096)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return usagef("reading the token: %v", err)
	}

	token := strings.TrimSpace(line)
	if token == "" {
		return usagef("no token on the standard input")
	}

	me, err := c.me(ctx, NewRemote(u, token, c.Env.HTTP))
	if err != nil {
		return err
	}

	if err := c.Env.Store.Save(Credentials{Server: u.String(), Token: token}); err != nil {
		return fmt.Errorf("keeping the login: %w", err)
	}

	if fsStore, ok := c.Env.Store.(*FileStore); ok {
		fmt.Fprintf(c.Env.Stderr, "warning: the token is kept in %s (readable only by you); no system credential store is used.\n", fsStore.Path)
	}

	fmt.Fprintf(c.Env.Stdout, "Logged in to %s: %s\n", u, me)

	return nil
}

func (c *CLI) logout() error {
	if c.Env.Store == nil {
		return nil
	}

	if err := c.Env.Store.Delete(); err != nil {
		return fmt.Errorf("forgetting the login: %w", err)
	}

	fmt.Fprintln(c.Env.Stdout, "Logged out. The token still works until you revoke it on the Tokens page.")

	return nil
}

func (c *CLI) status(ctx context.Context) error {
	server, token, err := c.target("")
	if err != nil {
		return err
	}

	if server == "" || token == "" {
		fmt.Fprintln(c.Env.Stdout, "Not logged in: commands run locally (personal edition).")

		return nil
	}

	u, err := ParseServer(server)
	if err != nil {
		return err
	}

	me, err := c.me(ctx, NewRemote(u, token, c.Env.HTTP))
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Env.Stdout, "Logged in to %s: %s\n", u, me)

	return nil
}

// me asks the server who the token is, which also proves the token works.
func (c *CLI) me(ctx context.Context, r *Remote) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base.String()+"/api/v1/me", nil)
	if err != nil {
		return "", usagef("bad server: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json")

	res, err := r.hc.Do(req)
	if err != nil {
		return "", &Failure{Code: ExitFailure, Msg: r.scrub("cannot reach the server: " + urlError(err))}
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", r.problem(res)
	}

	var me struct {
		Role      string   `json:"role"`
		Scopes    []string `json:"scopes"`
		Workspace struct {
			Name string `json:"name"`
		} `json:"workspace"`
	}

	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&me); err != nil {
		return "", &Failure{Code: ExitFailure, Msg: "the server did not answer with JSON: is this a vukem-finances server?"}
	}

	return fmt.Sprintf("workspace %q as %s, %d scopes", cleanText(me.Workspace.Name), cleanText(me.Role), len(me.Scopes)), nil
}
