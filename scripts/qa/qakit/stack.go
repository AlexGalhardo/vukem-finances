package qakit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
)

// Stack is a throwaway installation: a built binary, one SQLite database and
// the environment to run both against it. Dir is removed by Close.
type Stack struct {
	Dir  string
	Bin  string
	DB   string // the database file, with forward slashes
	Env  []string
	Port string
	Base string

	server *exec.Cmd
	log    *os.File
}

// NewStack builds the app (unless bin is given) into a new temporary
// directory, with a personal-edition environment and a fresh APP_KEY. extraEnv
// entries (KEY=value) win over the defaults.
func NewStack(ctx context.Context, bin string, extraEnv ...string) (*Stack, error) {
	dir, err := os.MkdirTemp("", "vf-qa-")
	if err != nil {
		return nil, err
	}

	s := &Stack{Dir: dir, Bin: bin}

	if s.Bin == "" {
		s.Bin = filepath.Join(dir, "app")
		if runtime.GOOS == "windows" {
			s.Bin += ".exe"
		}

		if err := Build(ctx, s.Bin, "."); err != nil {
			s.Close()
			return nil, err
		}
	}

	if s.Port, err = FreePort(); err != nil {
		s.Close()
		return nil, err
	}

	s.Base = "http://127.0.0.1:" + s.Port
	s.DB = filepath.ToSlash(filepath.Join(dir, "qa.sqlite"))

	key := make([]byte, 32)
	_, _ = rand.Read(key)

	s.Env = append(os.Environ(),
		"APP_NAME=vukem-finances",
		"APP_ENV=development",
		"APP_DEBUG=false",
		"APP_EDITION=personal",
		"APP_KEY=base64:"+base64.StdEncoding.EncodeToString(key),
		"PORT="+s.Port,
		"APP_URL="+s.Base,
		"DATABASE_URL=file:"+s.DB,
		"MAIL_MAILER=log",
		"MCP_TOOLSETS=read",
		"TELEGRAM_BOT_TOKEN=",
	)
	s.Env = append(s.Env, extraEnv...)

	return s, nil
}

// Build compiles ./cmd/app of the module at root into out.
func Build(ctx context.Context, out, root string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./cmd/app")
	cmd.Dir = root
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	return nil
}

// Command runs the binary with args against the stack, and returns its output.
func (s *Stack) Command(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, s.Bin, args...)
	cmd.Env = s.Env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, tail(string(out)))
	}

	return string(out), nil
}

func tail(s string) string {
	if len(s) > 400 {
		return s[len(s)-400:]
	}

	return s
}

// Start runs the server in the background, logging to server.log, and waits
// until it answers.
func (s *Stack) Start(ctx context.Context) error {
	f, err := os.OpenFile(filepath.Join(s.Dir, "server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	s.log = f
	s.server = exec.Command(s.Bin)
	s.server.Env, s.server.Stdout, s.server.Stderr = s.Env, f, f

	if err := s.server.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	return WaitUp(ctx, s.Base+"/login")
}

// Kill stops the server abruptly (no shutdown, as a crash or a power cut
// would), by the PID this process started.
func (s *Stack) Kill() {
	if s.server != nil && s.server.Process != nil {
		_ = s.server.Process.Kill()
		_, _ = s.server.Process.Wait()
	}

	s.server = nil

	if s.log != nil {
		_ = s.log.Close()
		s.log = nil
	}
}

// LogPath is the server log, appended to across restarts.
func (s *Stack) LogPath() string { return filepath.Join(s.Dir, "server.log") }

// Close stops the server and removes the directory.
func (s *Stack) Close() {
	s.Kill()
	_ = os.RemoveAll(s.Dir)
}

// FreePort returns a TCP port that is free on the loopback address.
func FreePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	defer func() { _ = l.Close() }()

	_, port, err := net.SplitHostPort(l.Addr().String())

	return port, err
}

// WaitUp polls url until the server answers anything, for up to a minute.
func WaitUp(ctx context.Context, url string) error {
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		if res, err := http.DefaultClient.Do(req); err == nil {
			_ = res.Body.Close()
			return nil
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("the server did not answer on %s within 60 s", url)
}

// Fingerprint is what a drill compares before and after a failure: the rows
// of the synced tables, their amounts and the distinct uids.
type Fingerprint struct {
	Transactions int64
	DistinctUIDs int64
	AmountSum    int64
	Accounts     int64
	Categories   int64
	Users        int64
}

// Fingerprint reads the database file, which must not be in use by a writer
// that is mid-transaction (a killed server leaves a consistent file).
func (s *Stack) Fingerprint(ctx context.Context) (Fingerprint, error) {
	return FingerprintOf(ctx, s.DB)
}

// FingerprintOf reads the database at path.
func FingerprintOf(ctx context.Context, path string) (Fingerprint, error) {
	db, err := database.Open(ctx, "file:"+path)
	if err != nil {
		return Fingerprint{}, err
	}

	defer func() { _ = db.Close() }()

	var f Fingerprint

	err = db.SQL().QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM transactions WHERE deleted_at IS NULL),
		(SELECT COUNT(DISTINCT uid) FROM transactions WHERE deleted_at IS NULL),
		(SELECT COALESCE(SUM(amount_cents), 0) FROM transactions WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM accounts WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM categories WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM users)`).
		Scan(&f.Transactions, &f.DistinctUIDs, &f.AmountSum, &f.Accounts, &f.Categories, &f.Users)

	return f, err
}

// Scalar runs a query that returns one integer.
func (s *Stack) Scalar(ctx context.Context, query string, args ...any) (int64, error) {
	db, err := database.Open(ctx, "file:"+s.DB)
	if err != nil {
		return 0, err
	}

	defer func() { _ = db.Close() }()

	var n int64

	err = db.SQL().QueryRowContext(ctx, query, args...).Scan(&n)

	return n, err
}
