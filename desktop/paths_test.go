package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsLayout(t *testing.T) {
	data := filepath.Join(t.TempDir(), "vukem-finances")

	p, err := ResolvePaths(data)
	if err != nil {
		t.Fatal(err)
	}

	for _, dir := range []string{p.Data, p.Logs, p.Backups} {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			t.Errorf("%s is not a directory: %v", dir, err)
		}
	}

	for name, got := range map[string]string{"database": p.Database, "key": p.Key, "lock": p.Lock} {
		if filepath.Dir(got) != p.Data {
			t.Errorf("%s = %s, want it inside %s", name, got, p.Data)
		}
	}

	if !strings.HasPrefix(p.DatabaseURL(), "file:") || strings.Contains(p.DatabaseURL(), `\`) {
		t.Errorf("DatabaseURL = %q, want file: with forward slashes", p.DatabaseURL())
	}

	// Resolving again changes nothing.
	if again, err := ResolvePaths(data); err != nil || again != p {
		t.Errorf("second ResolvePaths = %+v, %v", again, err)
	}
}

func TestPathsDefaultIsUnderUserConfigDir(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skip("no user config dir")
	}

	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	p, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(p.Data) != "vukem-finances" || p.Data == filepath.Join(base, "vukem-finances") {
		t.Errorf("Data = %s, want .../vukem-finances under the (overridden) config dir", p.Data)
	}
}
