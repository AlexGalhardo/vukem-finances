package changes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/schema"
)

// Trigger name suffixes: every synced table carries the three.
var triggerSuffixes = []string{"_changes_insert", "_changes_update", "_immutable_ids"} //nolint:gochecknoglobals // constant list

// TriggerNames lists the triggers a synced table must have.
func TriggerNames(table string) []string {
	out := make([]string, len(triggerSuffixes))
	for i, s := range triggerSuffixes {
		out[i] = table + s
	}

	return out
}

// TriggerSQL returns the statements that (re)create the change-log triggers
// of e. Table and column names come from code, never from a request, and are
// checked to be plain identifiers.
//
//   - insert: logs one row with every writable field;
//   - update: when a writable column or deleted_at changed and the writer did
//     not set version itself, logs the changed columns, then bumps version in
//     the database (so imports, jobs and future code are logged without any
//     discipline). A null to non-null deleted_at is a delete, anything else
//     an upsert (a restore included);
//   - immutable ids: uid and workspace_id never change.
//
// The log holds column names and the row's uid, never a value.
func TriggerSQL(e Entity) ([]string, error) {
	for _, name := range append(e.tracked(), e.Table, e.Name, "uid", "version", "workspace_id") {
		if !identOK(name) {
			return nil, fmt.Errorf("changes: %q is not a plain identifier", name)
		}
	}

	fields, err := json.Marshal(e.Fields)
	if err != nil {
		return nil, err
	}

	names := TriggerNames(e.Table)
	t := e.Table

	var changed, pieces []string

	for _, c := range e.tracked() {
		changed = append(changed, fmt.Sprintf("OLD.%s IS NOT NEW.%s", c, c))
		pieces = append(pieces, fmt.Sprintf("SELECT '%s' AS f WHERE OLD.%s IS NOT NEW.%s", c, c, c))
	}

	insert := fmt.Sprintf(`CREATE TRIGGER %s AFTER INSERT ON %s
BEGIN
	INSERT INTO changes (workspace_id, entity, uid, op, row_version, fields)
	VALUES (NEW.workspace_id, '%s', NEW.uid,
		CASE WHEN NEW.deleted_at IS NULL THEN 'upsert' ELSE 'delete' END, NEW.version, '%s');
END`, names[0], t, e.Name, fields)

	update := fmt.Sprintf(`CREATE TRIGGER %s AFTER UPDATE ON %s
WHEN OLD.version IS NEW.version AND (%s)
BEGIN
	INSERT INTO changes (workspace_id, entity, uid, op, row_version, fields)
	VALUES (NEW.workspace_id, '%s', NEW.uid,
		CASE WHEN OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN 'delete' ELSE 'upsert' END,
		OLD.version + 1,
		(SELECT json_group_array(f) FROM (%s)));
	UPDATE %s SET version = OLD.version + 1 WHERE id = NEW.id;
END`, names[1], t, strings.Join(changed, " OR "), e.Name, strings.Join(pieces, " UNION ALL "), t)

	immutable := fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE OF uid, workspace_id ON %s
WHEN OLD.uid IS NOT NEW.uid OR OLD.workspace_id IS NOT NEW.workspace_id
BEGIN
	SELECT RAISE(ABORT, 'uid and workspace_id never change');
END`, names[2], t)

	var out []string

	for i, stmt := range []string{insert, update, immutable} {
		out = append(out, "DROP TRIGGER IF EXISTS "+names[i], stmt)
	}

	return out, nil
}

// InstallTriggers creates the triggers of e through Schema.Raw. Run it in the
// migration that adds the synced table, after every step that can rebuild the
// table (a rebuild drops triggers), and again after a rebuild or a change of
// e.Fields: it drops and recreates, so it is safe to repeat. A migration
// passes a literal Entity, so its checksum does not move when the registry
// changes later.
func InstallTriggers(s *schema.Schema, e Entity) error {
	stmts, err := TriggerSQL(e)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if err := s.Raw(stmt); err != nil {
			return err
		}
	}

	return nil
}

// VerifyTriggers returns an error naming every registered table that lacks
// one of its triggers. The boot runs it outside tests, and a test runs it too.
func VerifyTriggers(ctx context.Context, db *database.DB) error {
	var missing []string

	for _, e := range Entities() {
		for _, name := range TriggerNames(e.Table) {
			var n int

			err := db.SQL().QueryRowContext(ctx,
				"SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = ? AND tbl_name = ?", name, e.Table).Scan(&n)
			if err != nil {
				return fmt.Errorf("changes: verify triggers: %w", err)
			}

			if n == 0 {
				missing = append(missing, name)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("changes: missing triggers %s: reinstall them with changes.InstallTriggers after a table rebuild",
			strings.Join(missing, ", "))
	}

	return nil
}

func identOK(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') {
			continue
		}

		return false
	}

	return true
}
