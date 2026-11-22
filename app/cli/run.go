package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
)

// Env is everything the CLI takes from its surroundings, so a test can give it
// a fake terminal, environment and server.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) (string, bool)
	// StdinTTY and StdoutTTY say whether the streams are a terminal.
	StdinTTY  bool
	StdoutTTY bool
	// OpenLocal opens the local backend; nil means local mode is not built in.
	OpenLocal func(ctx context.Context) (Backend, error)
	// Store keeps the login of remote mode; HTTP is the client it uses (nil
	// means NewHTTPClient).
	Store CredentialStore
	HTTP  *http.Client
}

// CLI runs the commands of a tree.
type CLI struct {
	Tree *Tree
	Env  Env
}

// globals are the flags every command has.
type globals struct {
	output  string
	input   string
	yes     bool
	server  string
	local   bool
	ifMatch string
	key     string
	out     string
}

func (g *globals) bind(fs *flag.FlagSet) {
	fs.StringVar(&g.output, "output", "", "")
	fs.StringVar(&g.output, "o", "", "")
	fs.StringVar(&g.input, "input", "", "")
	fs.BoolVar(&g.yes, "yes", false, "")
	fs.StringVar(&g.server, "server", "", "")
	fs.BoolVar(&g.local, "local", false, "")
	fs.StringVar(&g.ifMatch, "if-match", "", "")
	fs.StringVar(&g.key, "idempotency-key", "", "")
	fs.StringVar(&g.out, "out", "", "")
}

// fieldFlags holds the values of the flags of one command.
type fieldFlags struct {
	strs   map[string]*string
	bools  map[string]*bool
	ints   map[string]*int64
	lists  map[string]*stringList
	amount map[string]*string
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func (c Command) bind(fs *flag.FlagSet) *fieldFlags {
	ff := &fieldFlags{
		strs: map[string]*string{}, bools: map[string]*bool{}, ints: map[string]*int64{},
		lists: map[string]*stringList{}, amount: map[string]*string{},
	}

	for _, f := range c.Fields {
		if f.Positional >= 0 {
			continue
		}

		switch f.Kind {
		case KindString:
			ff.strs[f.Name] = fs.String(f.Flag, "", "")
		case KindBool:
			ff.bools[f.Name] = fs.Bool(f.Flag, false, "")
		case KindInt:
			ff.ints[f.Name] = fs.Int64(f.Flag, 0, "")

			if f.Cents {
				ff.amount[f.Name] = fs.String(strings.TrimSuffix(f.Flag, "-cents"), "", "")
			}
		case KindStrings:
			l := &stringList{}
			ff.lists[f.Name] = l
			fs.Var(l, f.Flag, "")
		}
	}

	return ff
}

// Run runs `<resource> <verb> ...`; args are what follows the resource.
func (c *CLI) Run(ctx context.Context, resource string, args []string) error {
	if resource == "auth" {
		return c.Auth(ctx, args)
	}

	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		c.printVerbs(resource)

		if len(args) == 0 {
			return usagef("%s needs a verb", resource)
		}

		return nil
	}

	cmd, ok := c.find(resource, args[0])
	if !ok {
		return usagef("unknown command %q: run `%s` to list the verbs", resource+" "+args[0], resource)
	}

	return c.runCommand(ctx, cmd, args[1:])
}

func (c *CLI) find(resource, verb string) (Command, bool) {
	for _, cmd := range c.Tree.Of(resource) {
		if cmd.Verb == verb {
			return cmd, true
		}
	}

	return Command{}, false
}

func (c *CLI) printVerbs(resource string) {
	fmt.Fprintf(c.Env.Stdout, "Usage: %s <verb> [arguments] [flags]\n\nVerbs:\n", resource)

	for _, cmd := range c.Tree.Of(resource) {
		fmt.Fprintf(c.Env.Stdout, "  %-22s %s\n", cmd.Verb, cmd.Op.Summary)
	}
}

func (c *CLI) runCommand(ctx context.Context, cmd Command, args []string) error {
	var g globals

	fs := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	g.bind(fs)
	ff := cmd.bind(fs)

	pos, err := parseInterleaved(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		c.printUsage(cmd)

		return nil
	}

	if err != nil {
		return usagef("%v (run `%s --help`)", err, cmd.Name())
	}

	body, err := c.body(cmd, &g, fs, ff, pos)
	if err != nil {
		return err
	}

	format, err := c.format(g.output)
	if err != nil {
		return err
	}

	if err := c.confirm(cmd, body, g.yes); err != nil {
		return err
	}

	back, err := c.backend(ctx, &g)
	if err != nil {
		return err
	}

	defer func() { _ = back.Close() }()

	res, err := back.Call(ctx, Request{Cmd: cmd, Body: body, IfMatch: g.ifMatch, Key: g.key})
	if err != nil {
		return err
	}

	return c.write(cmd, res, format, g.out)
}

// confirm asks before a destructive command runs. It runs before the backend
// is opened, so a refusal touches neither the app nor the server. Without a
// terminal there is no one to ask, and --yes is the only way.
func (c *CLI) confirm(cmd Command, body map[string]any, yes bool) error {
	if !cmd.Destructive() || yes {
		return nil
	}

	if !c.Env.StdinTTY {
		return usagef("%s removes data: pass --yes to run it without a terminal", cmd.Name())
	}

	target := ""
	for _, p := range cmd.Params {
		target += fmt.Sprintf(" %v", body[p])
	}

	fmt.Fprintf(c.Env.Stderr, "%s%s removes data. Continue? [y/N] ", cmd.Name(), cleanText(target))

	line, _ := bufio.NewReader(io.LimitReader(c.Env.Stdin, 4096)).ReadString('\n')

	if answer := strings.ToLower(strings.TrimSpace(line)); answer != "y" && answer != "yes" {
		return &Failure{Code: ExitFailure, Msg: "cancelled: nothing was changed"}
	}

	return nil
}

// parseInterleaved parses flags that may come after the positional arguments.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string

	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}

		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}

		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

func (c *CLI) body(cmd Command, g *globals, fs *flag.FlagSet, ff *fieldFlags, pos []string) (map[string]any, error) {
	body := map[string]any{}

	if g.input != "" {
		var err error
		if body, err = c.readInput(g.input); err != nil {
			return nil, err
		}
	}

	if len(pos) > len(cmd.Params) {
		return nil, usagef("%s takes %d argument(s), got %d", cmd.Name(), len(cmd.Params), len(pos))
	}

	for _, f := range cmd.Fields {
		if f.Positional < 0 {
			continue
		}

		if f.Positional >= len(pos) {
			if _, given := body[f.Name]; !given {
				return nil, usagef("%s needs <%s>", cmd.Name(), f.Name)
			}

			continue
		}

		body[f.Name] = pos[f.Positional]
	}

	set := map[string]bool{}

	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	for _, f := range cmd.Fields {
		if f.Positional >= 0 {
			continue
		}

		if err := setFlagValue(body, f, ff, set); err != nil {
			return nil, err
		}
	}

	return body, nil
}

func setFlagValue(body map[string]any, f Field, ff *fieldFlags, set map[string]bool) error {
	switch f.Kind {
	case KindString:
		if set[f.Flag] {
			body[f.Name] = *ff.strs[f.Name]
		}
	case KindBool:
		if set[f.Flag] {
			body[f.Name] = *ff.bools[f.Name]
		}
	case KindInt:
		short := strings.TrimSuffix(f.Flag, "-cents")

		switch {
		case f.Cents && set[short] && set[f.Flag]:
			return usagef("use --%s or --%s, not both", f.Flag, short)
		case f.Cents && set[short]:
			cents, err := finance.ParseAmount(*ff.amount[f.Name])
			if err != nil {
				return usagef("--%s: %q is not an amount", short, *ff.amount[f.Name])
			}

			body[f.Name] = cents
		case set[f.Flag]:
			body[f.Name] = *ff.ints[f.Name]
		}
	case KindStrings:
		if set[f.Flag] {
			list := []string{}

			for _, v := range *ff.lists[f.Name] {
				if v != "" {
					list = append(list, v)
				}
			}

			body[f.Name] = list
		}
	}

	return nil
}

const maxInput = 1 << 20

func (c *CLI) readInput(src string) (map[string]any, error) {
	var (
		raw []byte
		err error
	)

	if src == "-" {
		raw, err = io.ReadAll(io.LimitReader(c.Env.Stdin, maxInput+1))
	} else {
		raw, err = readFileLimited(src)
	}

	if err != nil {
		return nil, usagef("reading the input: %v", err)
	}

	if len(raw) > maxInput {
		return nil, usagef("the input is larger than %d bytes", maxInput)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil || body == nil {
		return nil, usagef("the input is not a JSON object")
	}

	return body, nil
}

func readFileLimited(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return io.ReadAll(io.LimitReader(f, maxInput+1))
}

func (c *CLI) format(flagValue string) (string, error) {
	switch flagValue {
	case "":
		if c.Env.StdoutTTY {
			return "table", nil
		}

		return "json", nil
	case "table", "json", "csv":
		return flagValue, nil
	}

	return "", usagef("--output must be table, json or csv, not %q", flagValue)
}

func (c *CLI) localBackend(ctx context.Context) (Backend, error) {
	if c.Env.OpenLocal == nil {
		return nil, usagef("local mode is not available in this build")
	}

	b, err := c.Env.OpenLocal(ctx)
	if err != nil {
		if _, ok := errors.AsType[*Failure](err); ok {
			return nil, err
		}

		return nil, fmt.Errorf("local mode: %w", err)
	}

	return b, nil
}

func (c *CLI) printUsage(cmd Command) {
	w := c.Env.Stdout

	fmt.Fprintf(w, "Usage: %s", cmd.Name())

	for _, p := range cmd.Params {
		fmt.Fprintf(w, " <%s>", p)
	}

	fmt.Fprintf(w, " [flags]\n\n%s\n", cmd.Op.Summary)

	if cmd.Destructive() {
		fmt.Fprintln(w, "\nThis command removes data: it asks for confirmation on a terminal and needs --yes otherwise.")
	}

	fmt.Fprintln(w, "\nFlags:")

	for _, line := range flagLines(cmd) {
		fmt.Fprintln(w, "  "+line)
	}
}

// flagLines describes the flags of a command: its own, then the shared ones.
func flagLines(cmd Command) []string {
	var out []string

	for _, f := range cmd.Fields {
		if f.Positional >= 0 {
			continue
		}

		switch {
		case f.Cents:
			short := strings.TrimSuffix(f.Flag, "-cents")
			out = append(out, "--"+f.Flag+" N   whole cents", "--"+short+" TEXT   an amount such as 12.50 or 1.234,56")
		case f.Kind == KindBool:
			out = append(out, "--"+f.Flag)
		case f.Kind == KindStrings:
			out = append(out, "--"+f.Flag+" VALUE   repeat for more than one")
		case f.Kind == KindInt:
			out = append(out, "--"+f.Flag+" N")
		default:
			out = append(out, "--"+f.Flag+" TEXT")
		}
	}

	out = append(out,
		"--input FILE|-   the whole input as JSON",
		"--output table|json|csv",
		"--server URL | --local   where to run",
	)

	if cmd.Op.Kind != "read" {
		out = append(out, "--if-match VERSION", "--idempotency-key KEY")
	}

	if cmd.Destructive() {
		out = append(out, "--yes")
	}

	return out
}
