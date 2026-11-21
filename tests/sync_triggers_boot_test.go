package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// TestVerifyTriggersAtBoot checks that outside the test environment a synced
// table without its change-log triggers stops the boot, and that a healthy
// database boots.
func TestVerifyTriggersAtBoot(t *testing.T) {
	t.Parallel()

	// Not t.TempDir: the app whose boot fails keeps the file open until the
	// process ends, and Windows refuses to remove an open file.
	dir, err := os.MkdirTemp("", "triggers-boot")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := filepath.ToSlash(filepath.Join(dir, "database.sqlite"))
	env := map[string]string{"DATABASE_URL": "file:" + path, "APP_ENV": "development", "APP_EDITION": "personal"}

	first, err := vukemtest.Build(t, bootstrap.New, vukemtest.Options{Env: env})
	if err != nil {
		t.Fatal(err)
	}

	db, err := first.DB()
	if err != nil {
		t.Fatal(err)
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatalf("a migrated database must pass the check: %v", err)
	}

	if _, err := db.SQL().ExecContext(t.Context(), "DROP TRIGGER transactions_changes_insert"); err != nil {
		t.Fatal(err)
	}

	_, err = vukemtest.Build(t, bootstrap.New, vukemtest.Options{Env: env, SkipMigrations: true})
	if err == nil || !strings.Contains(err.Error(), "transactions_changes_insert") {
		t.Fatalf("boot with a missing trigger = %v, want it to fail naming the trigger", err)
	}
}
