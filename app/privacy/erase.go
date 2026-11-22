package privacy

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
)

// DefaultDatabaseURL is the DATABASE_URL the framework uses when none is set.
const DefaultDatabaseURL = "file:storage/database.sqlite"

// EraseWord is what the person types to confirm "erase all local data".
const EraseWord = "ERASE"

var (
	errNotPersonal = errs.New(http.StatusNotFound, "Not found.")
	errWrongErase  = errs.New(http.StatusUnprocessableEntity, "Type "+EraseWord+" exactly to confirm.")
)

// ErrNoDatabaseFile is returned for a database that is not a file (in memory,
// or a server): there is nothing here to remove.
var ErrNoDatabaseFile = errors.New("privacy: the database is not a SQLite file")

// DatabaseFiles returns the files SQLite keeps for the database named by
// databaseURL: the file itself, its write-ahead log, its shared-memory index and
// a rollback journal if one was left. An in-memory or server database has none.
func DatabaseFiles(databaseURL string) ([]string, error) {
	raw := strings.TrimSpace(databaseURL)
	if raw == "" {
		raw = DefaultDatabaseURL
	}

	if !strings.HasPrefix(raw, "file:") && !strings.HasPrefix(raw, "sqlite:") {
		return nil, ErrNoDatabaseFile
	}

	if strings.Contains(raw, "mode=memory") {
		return nil, ErrNoDatabaseFile
	}

	path := strings.TrimPrefix(strings.TrimPrefix(raw, "sqlite:"), "file:")
	path, _, _ = strings.Cut(path, "?")

	if path == "" || path == ":memory:" {
		return nil, ErrNoDatabaseFile
	}

	return []string{path, path + "-wal", path + "-shm", path + "-journal"}, nil
}

// EraseLocalData closes the database and removes its file with the -wal and -shm
// files, so no byte of the old data stays on disk. It is the personal edition's
// "erase all local data": the caller then ends the process, and the next start
// (the container policy, or the person opening the desktop app again) migrates a
// new empty database and shows the setup.
//
// It removes the files in the order SQLite would not read them back (the log
// before the database) and fails, naming the file, if one cannot be removed or
// is still there afterwards.
func EraseLocalData(db *database.DB, databaseURL string) error {
	files, err := DatabaseFiles(databaseURL)
	if err != nil {
		return err
	}

	// Closing checkpoints and releases the files; Windows cannot remove an open one.
	if err := db.Close(); err != nil {
		return fmt.Errorf("closing the database: %w", err)
	}

	for _, f := range []string{files[1], files[2], files[3], files[0]} {
		if err := os.Remove(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("removing %s: %w", f, err)
		}
	}

	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			return fmt.Errorf("%s is still there", f)
		}
	}

	return nil
}

// DefaultEraseDelay is how long the erase waits after the request answered, so the
// browser receives its response (and the page that says the data is gone) first.
const DefaultEraseDelay = 2 * time.Second

// ScheduleErase checks that the edition and the typed word allow erasing all
// local data, then erases it after a short delay and calls Restart. The caller has
// already checked the role and re-authenticated the person.
func (s *Service) ScheduleErase(word string) error {
	if !s.WS.Edition.Personal() {
		return errNotPersonal
	}

	if word != EraseWord {
		return errWrongErase
	}

	// A double submit must not close the database under the first erase.
	if !s.erasing.CompareAndSwap(false, true) {
		return nil
	}

	time.AfterFunc(s.EraseDelay, func() {
		err := EraseLocalData(s.DB, s.DatabaseURL)

		switch {
		case errors.Is(err, ErrNoDatabaseFile):
			s.erasing.Store(false)
			s.Log.Error("privacy: erasing the local data failed", "err", err)

			return
		case err != nil:
			// The database is closed by now, so the process cannot serve anything:
			// end it, and the next start shows the data that is still there.
			s.Log.Error("privacy: erasing the local data failed", "err", err)
		default:
			s.Log.Info("local data erased")
		}

		if s.Restart != nil {
			s.Restart()
		}
	})

	return nil
}
