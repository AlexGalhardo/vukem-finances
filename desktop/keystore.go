package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/AlexGalhardo/vukem/config"
)

// LoadOrCreateKey returns APP_KEY from path, creating the file with a new
// random key only when it does not exist. An existing file that is empty or
// malformed is an error and is never replaced: a new key would make every
// encrypted field (AI keys, sessions) unreadable.
func LoadOrCreateKey(path string) (string, error) {
	raw, err := os.ReadFile(path)

	switch {
	case err == nil:
		key := strings.TrimSpace(string(raw))
		if _, err := config.ParseKey(config.NewSecret(key)); err != nil {
			return "", fmt.Errorf("%s is not a valid application key (restore it from your backup, it is never regenerated): %w", path, err)
		}

		return key, nil
	case !errors.Is(err, fs.ErrNotExist):
		return "", err
	}

	key := config.GenerateKey()

	// O_EXCL: if another process created the file in the meantime, read that.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return LoadOrCreateKey(path)
	}

	if err != nil {
		return "", err
	}

	if err := restrictToUser(path); err != nil {
		_ = f.Close()
		_ = os.Remove(path)

		return "", fmt.Errorf("restrict %s to the current user: %w", path, err)
	}

	if _, err := f.WriteString(key + "\n"); err != nil {
		_ = f.Close()
		_ = os.Remove(path)

		return "", err
	}

	if err := f.Close(); err != nil {
		return "", err
	}

	return key, nil
}
