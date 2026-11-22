package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

func startTestServer(t *testing.T, paths Paths, opts startOptions) *Server {
	t.Helper()

	s, err := StartServer(t.Context(), paths, opts)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_ = s.Stop(ctx)
	})

	return s
}

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func code(t *testing.T, c *http.Client, url string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	return res.StatusCode
}

func TestServerStartsAndMigrates(t *testing.T) {
	paths, err := ResolvePaths(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	s := startTestServer(t, paths, startOptions{})

	if !strings.HasPrefix(s.Origin(), "http://127.0.0.1:") {
		t.Fatalf("origin = %s, want loopback", s.Origin())
	}

	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: noRedirect, Timeout: 20 * time.Second}

	// Another process without the cookie gets nothing, not even /up.
	if got := code(t, &http.Client{Timeout: 20 * time.Second}, s.Origin()+"/up"); got != http.StatusForbidden {
		t.Fatalf("GET /up without cookie = %d, want 403", got)
	}

	if got := code(t, c, s.EntryURL()); got != http.StatusSeeOther {
		t.Fatalf("entry = %d, want 303", got)
	}

	if got := code(t, c, s.EntryURL()); got != http.StatusUnauthorized {
		t.Errorf("second entry = %d, want 401 (the token is spent)", got)
	}

	if got := code(t, c, s.Origin()+"/up"); got != http.StatusOK {
		t.Errorf("GET /up with cookie = %d, want 200", got)
	}

	// First run: no owner yet, so the root leads to the setup page.
	if got := code(t, c, s.Origin()+"/setup"); got != http.StatusOK {
		t.Errorf("GET /setup = %d, want 200", got)
	}

	for _, f := range []string{paths.Database, paths.Key, filepath.Join(paths.Logs, "app.log")} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	db, err := sql.Open("sqlite", paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM users").Scan(&n); err != nil || n != 0 {
		t.Errorf("users table: count = %d, err = %v; want an empty migrated table", n, err)
	}

	// A fresh database has nothing worth backing up.
	if entries, _ := os.ReadDir(paths.Backups); len(entries) != 0 {
		t.Errorf("backups after a first run = %d, want 0", len(entries))
	}

	port := s.Port()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop = %v", err)
	}

	if conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second); err == nil {
		_ = conn.Close()
		t.Error("the port still accepts connections after Stop")
	}
}

func TestPreMigrateBackupKeepsThree(t *testing.T) {
	paths, err := ResolvePaths(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	// Four upgrades in a row: each starts with an applied schema and one new
	// migration, and the fourth leaves only the newest three backups.
	extra := func(n int) vukem.Option {
		all := migrations.All()
		for i := 1; i <= n; i++ {
			all = append(all, schema.Migration{
				ID: "29990101000000_desktop_test_" + string(rune('a'+i)),
				Up: func(s *schema.Schema) error {
					return s.Raw("CREATE TABLE desktop_test_" + string(rune('a'+i)) + " (id INTEGER)")
				},
			})
		}

		return vukem.WithMigrations(all)
	}

	// Run 0 builds the real schema (no backup: nothing was applied yet).
	stop(t, startTestServer(t, paths, startOptions{version: "1.0.0"}))

	for i := 1; i <= 4; i++ {
		time.Sleep(1100 * time.Millisecond) // backup names carry seconds

		stop(t, startTestServer(t, paths, startOptions{version: "1.0." + string(rune('0'+i)), appOptions: []vukem.Option{extra(i)}}))
	}

	entries, err := os.ReadDir(paths.Backups)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}

	if len(names) != 3 {
		t.Fatalf("backups = %v, want the newest 3", names)
	}

	for _, want := range []string{"pre-migrate-1.0.2-", "pre-migrate-1.0.3-", "pre-migrate-1.0.4-"} {
		found := false

		for _, n := range names {
			found = found || strings.HasPrefix(n, want)
		}

		if !found {
			t.Errorf("no backup starting with %s in %v", want, names)
		}
	}

	// A backup is a database that opens and holds the data of before.
	db, err := sql.Open("sqlite", filepath.Join(paths.Backups, names[len(names)-1]))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatalf("the backup does not open: %v", err)
	}
}

func stop(t *testing.T, s *Server) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop = %v", err)
	}
}

func TestPruneBackupsOrdersByTimestamp(t *testing.T) {
	dir := t.TempDir()

	for _, n := range []string{
		"pre-migrate-1_0_0-20260101T000001Z.sqlite", "pre-migrate-0_9_0-20260101T000002Z.sqlite",
		"pre-migrate-2_0_0-20260101T000003Z.sqlite", "pre-migrate-1_5_0-20260101T000004Z.sqlite",
		"notes.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := pruneBackups(dir, 3); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "pre-migrate-1_0_0-20260101T000001Z.sqlite")); err == nil {
		t.Error("the oldest backup was kept")
	}

	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Error("an unrelated file was removed")
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 4 {
		t.Errorf("%d files left, want 3 backups and notes.txt", len(entries))
	}
}
