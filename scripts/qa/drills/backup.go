package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/scripts/qa/qakit"
)

// seeded makes a stack with the schema applied and one workspace of rows.
func seeded(ctx context.Context, bin string, rows int) (*qakit.Stack, qakit.Workspace, error) {
	s, err := qakit.NewStack(ctx, bin)
	if err != nil {
		return nil, qakit.Workspace{}, err
	}

	if out, err := s.Command(ctx, "migrate"); err != nil {
		s.Close()
		return nil, qakit.Workspace{}, fmt.Errorf("migrate: %w (%s)", err, out)
	}

	ws, err := qakit.Seed(ctx, s.DB, rows, 0, 0, 2)
	if err != nil {
		s.Close()
		return nil, qakit.Workspace{}, fmt.Errorf("seed: %w", err)
	}

	return s, ws[0], nil
}

// backupRestore takes a snapshot with db:backup, destroys the live database,
// restores into another directory and checks the data and the API.
func backupRestore(ctx context.Context) error {
	s, ws, err := seeded(ctx, "", 1500)
	if err != nil {
		return err
	}

	defer s.Close()

	before, err := s.Fingerprint(ctx)
	if err != nil {
		return err
	}

	snapshot := filepath.Join(s.Dir, "backups", "snapshot.sqlite")
	if err := os.MkdirAll(filepath.Dir(snapshot), 0o700); err != nil {
		return err
	}

	if out, err := s.Command(ctx, "db:backup", "-to", snapshot); err != nil {
		return fmt.Errorf("backup: %w (%s)", err, out)
	}

	// The disaster: the live database and its write-ahead files are gone.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(s.DB + suffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	restored := filepath.ToSlash(filepath.Join(s.Dir, "restore", "database.sqlite"))
	if err := os.MkdirAll(filepath.Dir(restored), 0o700); err != nil {
		return err
	}

	if out, err := s.Command(ctx, "db:backup", "-restore", "-from", snapshot, "-to", restored); err != nil {
		return fmt.Errorf("restore: %w (%s)", err, out)
	}

	after, err := qakit.FingerprintOf(ctx, restored)
	if err != nil {
		return err
	}

	if before != after {
		return fmt.Errorf("the restored data differs: before %+v, after %+v", before, after)
	}

	// The restored file serves the same installation: the seeded token still works.
	if err := os.Rename(restored, s.DB); err != nil {
		return err
	}

	if err := s.Start(ctx); err != nil {
		return err
	}

	return expectMe(ctx, s, ws.Tokens[0])
}

func expectMe(ctx context.Context, s *qakit.Stack, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Base+"/api/v1/me", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET /api/v1/me answered %d: %.120s", res.StatusCode, body)
	}

	return nil
}

// upgrade builds the previous release from its tag, lets it make the
// database, then runs this tree's migrations and server over it.
func upgrade(ctx context.Context) error {
	tag, err := previousTag(ctx)
	if err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "vf-qa-prev-")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(dir) }()

	if err := extractTag(ctx, tag, dir); err != nil {
		return err
	}

	prevBin := filepath.Join(dir, "prev-app")
	if exe := exeSuffix(); exe != "" {
		prevBin += exe
	}

	if err := qakit.Build(ctx, prevBin, dir); err != nil {
		return fmt.Errorf("build %s: %w", tag, err)
	}

	// The previous release makes the schema and the rows.
	s, ws, err := seeded(ctx, prevBin, 1500)
	if err != nil {
		return err
	}

	defer s.Close()

	before, err := s.Fingerprint(ctx)
	if err != nil {
		return err
	}

	// This tree takes over the same database.
	current := filepath.Join(s.Dir, "current")
	if exe := exeSuffix(); exe != "" {
		current += exe
	}

	if err := qakit.Build(ctx, current, "."); err != nil {
		return err
	}

	s.Bin = current

	if out, err := s.Command(ctx, "migrate"); err != nil {
		return fmt.Errorf("migrate from %s: %w (%s)", tag, err, out)
	}

	after, err := s.Fingerprint(ctx)
	if err != nil {
		return err
	}

	if before != after {
		return fmt.Errorf("the upgrade changed the data: before %+v, after %+v", before, after)
	}

	if err := s.Start(ctx); err != nil {
		return err
	}

	fmt.Printf("      upgraded a %s database to this tree\n", tag)

	return expectMe(ctx, s, ws.Tokens[0])
}

func previousTag(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*").Output()
	if err != nil {
		return "", fmt.Errorf("no release tag to upgrade from: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// extractTag unpacks the tree of tag into dir, with git archive piped to tar.
func extractTag(ctx context.Context, tag, dir string) error {
	archive := exec.CommandContext(ctx, "git", "archive", "--format=tar", tag)
	untar := exec.CommandContext(ctx, "tar", "-x")
	untar.Dir = dir // a "C:" path after -C reads as a host name to GNU tar

	pipe, err := archive.StdoutPipe()
	if err != nil {
		return err
	}

	untar.Stdin = pipe
	untar.Stderr = os.Stderr

	if err := untar.Start(); err != nil {
		return err
	}

	if err := archive.Start(); err != nil {
		_ = untar.Process.Kill()
		return fmt.Errorf("git archive %s: %w", tag, err)
	}

	archiveErr := archive.Wait()

	if err := untar.Wait(); err != nil {
		return fmt.Errorf("unpack %s: %w", tag, err)
	}

	if archiveErr != nil {
		return fmt.Errorf("git archive %s: %w", tag, archiveErr)
	}

	return nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}
