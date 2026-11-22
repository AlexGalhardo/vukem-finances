package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Credentials are what `auth login` keeps: the server and the token of
// epic 14 for it.
type Credentials struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// CredentialStore keeps the login between runs.
type CredentialStore interface {
	Load() (Credentials, bool, error)
	Save(Credentials) error
	Delete() error
}

// FileStore keeps the login in a file only its owner can read (0600, in a 0700
// folder). No operating system credential store is used: reaching one from Go
// needs a dependency, or puts the token in the arguments of a helper program,
// which the CLI never does (decision in plans/decisions-log.md).
type FileStore struct {
	Path string
}

// DefaultStore is the file of the current person, under the user config folder.
func DefaultStore() (*FileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return &FileStore{Path: filepath.Join(dir, "vukem-finances", "credentials.json")}, nil
}

// Load reads the login; ok is false when there is none. On a system with Unix
// permissions it refuses a file others can read.
func (s *FileStore) Load() (Credentials, bool, error) {
	info, err := os.Stat(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, false, nil
	}

	if err != nil {
		return Credentials{}, false, err
	}

	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return Credentials{}, false, fmt.Errorf("%s can be read by others: run chmod 600 on it, or log in again", s.Path)
	}

	raw, err := os.ReadFile(s.Path)
	if err != nil {
		return Credentials{}, false, err
	}

	var c Credentials
	if err := json.Unmarshal(raw, &c); err != nil || c.Server == "" || c.Token == "" {
		return Credentials{}, false, fmt.Errorf("%s is not a login file: log in again", s.Path)
	}

	return c, true, nil
}

// Save writes the login through a temporary file, so the token is never in a
// file with looser permissions, not even for a moment.
func (s *FileStore) Save(c Credentials) error {
	dir := filepath.Dir(s.Path)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()

		return err
	}

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()

		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), s.Path)
}

// Delete forgets the login.
func (s *FileStore) Delete() error {
	err := os.Remove(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
