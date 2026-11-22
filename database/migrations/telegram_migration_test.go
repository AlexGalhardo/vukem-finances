package migrations_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/database/migrations"
)

// TestTelegramMigrationRoundTrip applies the Telegram migration, checks that a
// chat id is unique, rolls it back and applies it again.
func TestTelegramMigrationRoundTrip(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	m, err := schema.NewMigrator(db, migrations.All())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatal(err)
	}

	const ts = "'2026-01-02T03:04:05.006Z'"

	exec(t, db, "INSERT INTO users (name, email, password, created_at, updated_at) VALUES ('A', 'a@example.com', 'x', "+ts+", "+ts+")")
	exec(t, db, "INSERT INTO telegram_links (user_id, chat_id, language, created_at, updated_at) VALUES (1, 55, 'en', "+ts+", "+ts+")")

	if _, err := db.SQL().ExecContext(t.Context(),
		"INSERT INTO telegram_links (user_id, chat_id, language, created_at, updated_at) VALUES (1, 55, 'en', "+ts+", "+ts+")"); err == nil {
		t.Fatal("two links for one chat were accepted")
	}

	if _, err := m.Rollback(t.Context(), migrationsFrom(t, "20261123100000_create_telegram_tables")); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var n int
	if err := db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'telegram_%'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("telegram tables after rollback = %d (%v)", n, err)
	}

	if _, err := m.Up(t.Context()); err != nil {
		t.Fatalf("apply again: %v", err)
	}
}

// migrationsFrom counts the migrations from id to the last one, so a rollback
// that must undo id (and what depends on it) does not break when a later epic
// adds a migration after it.
func migrationsFrom(t *testing.T, id string) int {
	t.Helper()

	all := migrations.All()

	for i, m := range all {
		if m.ID == id {
			return len(all) - i
		}
	}

	t.Fatalf("migration %s is not registered", id)

	return 0
}
