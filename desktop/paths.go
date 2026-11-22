package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// appDirName is the folder under the user config directory that holds all the
// data of the desktop app: %AppData% on Windows, ~/Library/Application Support
// on macOS, ~/.config on Linux.
const appDirName = "vukem-finances"

// Paths are the places of the desktop app on disk. Everything lives in Data, so
// one folder is the whole backup.
type Paths struct {
	Data     string
	Database string
	Logs     string
	Backups  string
	Key      string
	Lock     string
}

// DefaultPaths resolves the data directory of the current user.
func DefaultPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("find the user config directory: %w", err)
	}

	return ResolvePaths(filepath.Join(base, appDirName))
}

// ResolvePaths lays out the files inside data and creates the folders, private
// to the user (0700; on Windows the user profile already restricts them).
func ResolvePaths(data string) (Paths, error) {
	data, err := filepath.Abs(data)
	if err != nil {
		return Paths{}, err
	}

	p := Paths{
		Data:     data,
		Database: filepath.Join(data, "database.sqlite"),
		Logs:     filepath.Join(data, "logs"),
		Backups:  filepath.Join(data, "backups"),
		Key:      filepath.Join(data, "app.key"),
		Lock:     filepath.Join(data, "lock"),
	}

	for _, dir := range []string{p.Data, p.Logs, p.Backups} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Paths{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return p, nil
}

// DatabaseURL is the DATABASE_URL of the app: the SQLite file in the data
// directory. The path uses forward slashes so a Windows drive letter survives.
func (p Paths) DatabaseURL() string {
	return "file:" + filepath.ToSlash(p.Database)
}
