package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AlexGalhardo/vukem/config"
)

func TestKeystoreCreatesThenReadsTheSameKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.key")

	first, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := config.ParseKey(config.NewSecret(first)); err != nil {
		t.Fatalf("generated key is not valid: %v", err)
	}

	for range 3 {
		again, err := LoadOrCreateKey(path)
		if err != nil || again != first {
			t.Fatalf("later run = %q, %v; want the same key %q", again, err, first)
		}
	}
}

func TestKeystoreNeverReplacesAnExistingFile(t *testing.T) {
	for name, content := range map[string]string{"empty": "", "garbage": "not a key", "short": "base64:AAAA"} {
		path := filepath.Join(t.TempDir(), "app.key")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		if key, err := LoadOrCreateKey(path); err == nil {
			t.Errorf("%s: got key %q, want an error", name, key)
		}

		if got, _ := os.ReadFile(path); string(got) != content {
			t.Errorf("%s: the file was rewritten to %q", name, got)
		}
	}
}

func TestKeystoreFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.key")
	if _, err := LoadOrCreateKey(path); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		assertOnlyCurrentUserCanRead(t, path)
		return
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", st.Mode().Perm())
	}
}
