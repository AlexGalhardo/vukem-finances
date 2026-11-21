package changes_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"
	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

const insertAccount = "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'Main', 'checking', " + now + ", " + now + ")"

func TestTriggers(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	uid := ulid.NewGenerator().New()

	exec(t, db, insertAccount, uid)

	step := func(name, op string, version int64, fields string) {
		t.Helper()

		got := logOf(t, db, "WHERE uid = ?", uid)
		last := got[len(got)-1]

		if last.Op != op || last.RowVersion != version || last.Entity != "account" || last.Workspace != 1 {
			t.Fatalf("%s: last change = %+v, want op %s version %d", name, last, op, version)
		}

		if !sameFields(last.Fields, fields) {
			t.Fatalf("%s: fields = %s, want %s", name, last.Fields, fields)
		}

		var stored int64
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT version FROM accounts WHERE uid = ?", uid).Scan(&stored); err != nil || stored != version {
			t.Fatalf("%s: stored version = %d (%v), want %d", name, stored, err, version)
		}
	}

	if got := logOf(t, db, "WHERE uid = ?", uid); len(got) != 1 {
		t.Fatalf("insert wrote %d changes, want 1", len(got))
	}

	step("insert", "upsert", 1, `["name","kind","opening_cents","archived_at","position"]`)

	exec(t, db, "UPDATE accounts SET name = 'Primary', updated_at = "+now+" WHERE uid = ?", uid)
	step("rename", "upsert", 2, `["name"]`)

	exec(t, db, "UPDATE accounts SET name = 'Primary', kind = 'savings', opening_cents = 5 WHERE uid = ?", uid)
	step("two fields", "upsert", 3, `["kind","opening_cents"]`)

	n := len(logOf(t, db, "WHERE uid = ?", uid))

	exec(t, db, "UPDATE accounts SET updated_at = datetime('now', '+1 second') WHERE uid = ?", uid)

	if got := len(logOf(t, db, "WHERE uid = ?", uid)); got != n {
		t.Fatalf("an update of updated_at only wrote %d new changes, want 0", got-n)
	}

	exec(t, db, "UPDATE accounts SET deleted_at = "+now+" WHERE uid = ?", uid)
	step("soft delete", "delete", 4, `["deleted_at"]`)

	exec(t, db, "UPDATE accounts SET deleted_at = NULL WHERE uid = ?", uid)
	step("restore", "upsert", 5, `["deleted_at"]`)

	if got := len(logOf(t, db, "WHERE uid = ?", uid)); got != 5 {
		t.Fatalf("log holds %d changes, want exactly one per write (5)", got)
	}
}

func TestTriggersCoverEveryEntityAndNeverLogValues(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	gen := ulid.NewGenerator()
	ts := now + ", " + now

	exec(t, db, "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'Secret account name', 'checking', "+ts+")", gen.New())
	exec(t, db, "INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'Secret category', 'expense', "+ts+")", gen.New())
	exec(t, db, "INSERT INTO transactions (workspace_id, uid, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, ?, 1, 1, 'expense', 987654321, 'Secret memo', '2026-01-01', "+ts+")", gen.New())
	exec(t, db, "INSERT INTO budgets (workspace_id, uid, category_id, limit_cents, created_at, updated_at) VALUES (1, ?, 1, 424242, "+ts+")", gen.New())

	var entities []string

	for _, r := range logOf(t, db, "") {
		entities = append(entities, r.Entity)

		for _, secret := range []string{"Secret", "987654321", "424242"} {
			if strings.Contains(r.Fields, secret) || strings.Contains(r.UID, secret) {
				t.Errorf("the change %+v holds a row value", r)
			}
		}
	}

	slices.Sort(entities)

	if want := []string{"account", "budget", "category", "transaction"}; !slices.Equal(entities, want) {
		t.Fatalf("entities logged = %v, want %v", entities, want)
	}

	if got := logOf(t, db, "WHERE workspace_id = 2"); len(got) != 0 {
		t.Fatalf("workspace 2 sees %d changes of workspace 1", len(got))
	}
}

func TestTriggersKeepIdentityFixed(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	exec(t, db, insertAccount, ulid.NewGenerator().New())

	for _, stmt := range []string{
		"UPDATE accounts SET uid = '01ARZ3NDEKTSV4RRFFQ69G5FAV'",
		"UPDATE accounts SET workspace_id = 2",
	} {
		if _, err := db.SQL().ExecContext(t.Context(), stmt); err == nil {
			t.Errorf("%s succeeded, want the identity trigger to refuse it", stmt)
		}
	}
}

func TestVerifyTriggers(t *testing.T) {
	t.Parallel()

	db := openDB(t)

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}

	exec(t, db, "DROP TRIGGER budgets_changes_update")

	err := changes.VerifyTriggers(t.Context(), db)
	if err == nil || !strings.Contains(err.Error(), "budgets_changes_update") {
		t.Fatalf("VerifyTriggers after a drop = %v, want it to name budgets_changes_update", err)
	}
}

// TestVerifyTriggersAfterRebuild is the documented risk: a rebuild of a synced
// table drops its triggers, VerifyTriggers notices, and InstallTriggers puts
// them back.
func TestVerifyTriggersAfterRebuild(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	s := schema.New(t.Context(), db)

	err := s.Rebuild("categories", func(tb *schema.Table) {
		tb.ID()
		tb.ForeignID("workspace_id").Constrained("workspaces").CascadeOnDelete()
		tb.String("name")
		tb.String("kind")
		tb.Timestamps()
		tb.ULID("uid")
		tb.SoftDeletes()
		tb.BigInteger("version").Default(1)
		tb.UniqueIndex("workspace_id", "uid")
		tb.UniqueIndex("workspace_id", "name").Where("deleted_at IS NULL")
	}, schema.DropTriggers())
	if err != nil {
		t.Fatal(err)
	}

	if err := changes.VerifyTriggers(t.Context(), db); err == nil {
		t.Fatal("VerifyTriggers passed after a rebuild dropped the triggers of categories")
	}

	e, _ := changes.Lookup("category")
	if err := changes.InstallTriggers(s, e); err != nil {
		t.Fatal(err)
	}

	if err := changes.VerifyTriggers(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}

func sameFields(got, want string) bool {
	norm := func(s string) []string {
		parts := strings.Split(strings.Trim(s, "[]"), ",")
		slices.Sort(parts)

		return parts
	}

	return slices.Equal(norm(got), norm(want))
}
