package migrations_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// TestFullRollbackFromFreshDatabase migrates a fresh database all the way up
// and rolls every migration back, then migrates up again.
func TestFullRollbackFromFreshDatabase(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	m, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatalf("up: %v", err)
	}

	if _, err := m.Rollback(t.Context(), len(migrations.All())); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}
