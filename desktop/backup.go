package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteBackupZip writes a zip with a consistent copy of the database and the
// application key. The key is part of the backup on purpose: without it the
// encrypted fields (AI keys, sessions) of the copy cannot be read. The file is
// as sensitive as the database, so the docs tell people to store it privately.
func WriteBackupZip(ctx context.Context, db *sql.DB, p Paths, w io.Writer) error {
	dir, err := os.MkdirTemp(p.Data, "backup-")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(dir) }()

	snapshot := filepath.Join(dir, "database.sqlite")
	if _, err := db.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(snapshot, "'", "''")+"'"); err != nil {
		return fmt.Errorf("snapshot the database: %w", err)
	}

	zw := zip.NewWriter(w)

	for _, f := range []struct{ name, path string }{
		{"database.sqlite", snapshot},
		{"app.key", p.Key},
	} {
		if err := addFile(zw, f.name, f.path); err != nil {
			_ = zw.Close()
			return err
		}
	}

	return zw.Close()
}

func addFile(zw *zip.Writer, name, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() { _ = src.Close() }()

	dst, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)

	return err
}
