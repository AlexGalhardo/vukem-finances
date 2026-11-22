package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing/fstest"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/config"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/desktopgate"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

const (
	// keepBackups is how many pre-migration backups stay on disk.
	keepBackups = 3
	// sessionLifetime keeps people signed in for 30 days: a desktop user
	// rarely wants to log in again.
	sessionLifetime = "720h"

	logMaxSize  = 5 << 20
	logKeepFile = 3
)

// Server is the embedded web app of the desktop shell: the personal edition on
// a loopback port, behind the gate.
type Server struct {
	paths  Paths
	app    *vukem.App
	ln     net.Listener
	logs   *rotatingFile
	token  string
	cancel context.CancelFunc
	done   chan error
}

// startOptions are the knobs of tests; the zero value is the real app.
type startOptions struct {
	appOptions []vukem.Option
	version    string
}

// StartServer prepares the data directory, migrates the database and starts
// the app on a free loopback port. The returned server is already accepting
// connections (the socket is bound before the app starts, so the port cannot
// be taken in between). Stop it with Stop.
func StartServer(ctx context.Context, paths Paths, opts startOptions) (*Server, error) {
	if opts.version == "" {
		opts.version = appconfig.Version
	}

	key, err := LoadOrCreateKey(paths.Key)
	if err != nil {
		return nil, err
	}

	var lc net.ListenConfig

	ln, err := lc.Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	token, err := desktopgate.NewToken()
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	gate, err := desktopgate.New(token)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	logs, err := newRotatingFile(filepath.Join(paths.Logs, "app.log"), logMaxSize, logKeepFile)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	s := &Server{paths: paths, ln: ln, logs: logs, token: token, done: make(chan error, 1)}

	if err := s.build(paths, desktopEnv(paths, key, ln.Addr().(*net.TCPAddr).Port), gate, opts); err != nil {
		s.close()
		return nil, err
	}

	if err := s.migrate(ctx, opts.version); err != nil {
		s.close()
		return nil, err
	}

	// The server outlives the caller's startup context: only Stop ends it.
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.cancel = cancel

	go func() {
		err := s.app.Run(runCtx)
		_ = s.logs.Close()
		s.done <- err
	}()

	return s, nil
}

// desktopEnv is the whole configuration of the app. It does not read the
// process environment or a .env file, so a stray DATABASE_URL or STRIPE_KEY of
// the user's shell cannot reach the app.
func desktopEnv(p Paths, key string, port int) map[string]string {
	return map[string]string{
		"APP_NAME":                        "vukem-finances",
		"APP_ENV":                         "production",
		"APP_EDITION":                     "personal",
		"APP_KEY":                         key,
		"APP_URL":                         "http://127.0.0.1:" + strconv.Itoa(port),
		"HOST":                            "127.0.0.1",
		"PORT":                            strconv.Itoa(port),
		"DATABASE_URL":                    p.DatabaseURL(),
		"MAIL_MAILER":                     "log",
		"LOG_FORMAT":                      "json",
		"QUEUE_TABLE":                     "queue_jobs",
		"QUEUE_FAILED_TABLE":              "queue_failed_jobs",
		"SESSION_LIFETIME":                sessionLifetime,
		"SESSION_IDLE_TIMEOUT":            sessionLifetime,
		"AUTH_REQUIRE_EMAIL_VERIFICATION": "false",
	}
}

func (s *Server) build(_ Paths, env map[string]string, gate *desktopgate.Gate, opts startOptions) error {
	getenv := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	// An empty file system: no .env is read from the working directory.
	src, err := config.LoadSource(fstest.MapFS{}, getenv, nil)
	if err != nil {
		return err
	}

	all := append(bootstrap.DesktopOptions(gate, s.ln), vukem.WithSource(src), vukem.WithLogOutput(s.logs))
	all = append(all, opts.appOptions...)

	s.app, err = bootstrap.New(all...)

	return err
}

// migrate runs the pending migrations in process. A database that already has
// applied migrations is copied first, so an upgrade that goes wrong can be
// undone from backups/. The destructive-migration prompt of the migrate command
// does not apply here: nobody is at a terminal, and the copy is the safety net.
func (s *Server) migrate(ctx context.Context, version string) error {
	m, err := s.app.Migrator()
	if err != nil {
		return err
	}

	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	status, err := m.Status(ctx)
	if err != nil {
		return err
	}

	if slices.ContainsFunc(status, func(st schema.Status) bool { return st.Ran }) {
		db, err := s.app.DB()
		if err != nil {
			return err
		}

		if _, err := backupBeforeMigrate(ctx, db.SQL(), s.paths.Backups, version, time.Now()); err != nil {
			return fmt.Errorf("back up the database before migrating: %w", err)
		}
	}

	if _, err := m.Up(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

// backupPrefix starts the name of every pre-migration backup.
const backupPrefix = "pre-migrate-"

// backupBeforeMigrate snapshots the live database with VACUUM INTO (a
// consistent copy, never a file copy of a database in WAL mode) into dir and
// keeps the newest keepBackups copies.
func backupBeforeMigrate(ctx context.Context, db *sql.DB, dir, version string, now time.Time) (string, error) {
	name := fmt.Sprintf("%s%s-%s.sqlite", backupPrefix, safeName(version), now.UTC().Format("20060102T150405Z"))
	dest := filepath.Join(dir, name)

	if _, err := db.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(dest, "'", "''")+"'"); err != nil {
		_ = os.Remove(dest)
		return "", err
	}

	return dest, pruneBackups(dir, keepBackups)
}

// pruneBackups deletes the oldest pre-migration backups of dir, keeping keep.
// The timestamp in the name sorts the same way as time does.
func pruneBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var names []string

	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), backupPrefix) && strings.HasSuffix(e.Name(), ".sqlite") {
			names = append(names, e.Name())
		}
	}

	slices.SortFunc(names, func(a, b string) int { return strings.Compare(backupStamp(a), backupStamp(b)) })

	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			return err
		}

		names = names[1:]
	}

	return nil
}

// backupStamp is the timestamp at the end of a backup name.
func backupStamp(name string) string {
	name = strings.TrimSuffix(name, ".sqlite")

	if i := strings.LastIndex(name, "-"); i >= 0 {
		return name[i+1:]
	}

	return name
}

// safeName keeps a version usable in a file name.
func safeName(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '.' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}

		return '_'
	}, v)
}

// EntryURL is the address the window opens first: it spends the launch token.
func (s *Server) EntryURL() string { return s.Origin() + desktopgate.EntryPath(s.token) }

// Origin is the loopback origin of the app, without the token.
func (s *Server) Origin() string { return "http://" + s.ln.Addr().String() }

// Port is the loopback port the app listens on.
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// Done receives the result of the app run once it ends.
func (s *Server) Done() <-chan error { return s.done }

// Stop shuts the app down gracefully (in-flight requests and jobs get the
// framework's SHUTDOWN_TIMEOUT) and waits for it, or for ctx.
func (s *Server) Stop(ctx context.Context) error {
	s.cancel()

	select {
	case err := <-s.done:
		s.done <- err // keep the result readable through Done

		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close releases what StartServer opened when it fails before Run.
func (s *Server) close() {
	_ = s.ln.Close()
	_ = s.logs.Close()
}
