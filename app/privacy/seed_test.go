package privacy_test

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/schema"
)

var seedCounter atomic.Int64

// workspaceTablesOf lists, from the live schema, every table that holds a
// workspace_id. The purge test is driven by it, so a table added tomorrow is
// seeded and checked without anybody remembering to.
func (w *world) workspaceTablesOf(t *testing.T) []schema.TableInfo {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	var out []schema.TableInfo

	for _, tbl := range snap.Tables {
		if slices.ContainsFunc(tbl.Columns, func(c schema.ColumnInfo) bool { return c.Name == "workspace_id" }) {
			out = append(out, tbl)
		}
	}

	return out
}

// seedEveryTable gives the workspace of w at least one row in every table that
// holds a workspace_id, by inserting what the schema asks for. It returns the
// tables it could not fill, which the caller must name on purpose.
func (w *world) seedEveryTable(t *testing.T, member int64) (unseeded []string) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	tables := w.workspaceTablesOf(t)
	pending := map[string]schema.TableInfo{}

	for _, tbl := range tables {
		pending[tbl.Name] = tbl
	}

	count := func(table string) int {
		var n int
		if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE workspace_id = ?", w.space.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n
	}

	for progress := true; progress && len(pending) > 0; {
		progress = false

		for name, tbl := range pending {
			if count(name) > 0 {
				delete(pending, name)

				progress = true

				continue
			}

			cols, args, ok := w.rowFor(t, tbl, member, nil)
			if !ok {
				continue
			}

			marks := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
			if _, err := db.ExecContext(t.Context(), "INSERT INTO "+name+" ("+strings.Join(cols, ",")+") VALUES ("+marks+")", args...); err != nil {
				t.Logf("seeding %s: %v", name, err)
				continue
			}

			delete(pending, name)

			progress = true
		}
	}

	for name := range pending {
		unseeded = append(unseeded, name)
	}

	slices.Sort(unseeded)

	return unseeded
}

// rowFor builds the columns and values of one valid row of tbl for the
// workspace of w. ok is false while a parent row it needs does not exist yet.
func (w *world) rowFor(t *testing.T, tbl schema.TableInfo, member int64, over map[string]any) (cols []string, args []any, ok bool) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	foreign := map[string]string{}
	for _, fk := range tbl.ForeignKeys {
		if len(fk.Columns) == 1 {
			foreign[fk.Columns[0]] = fk.RefTable
		}
	}

	now := time.Now().UTC()
	n := seedCounter.Add(1)

	for _, c := range tbl.Columns {
		switch {
		case over[c.Name] != nil:
			cols, args = append(cols, c.Name), append(args, over[c.Name])
		case c.Name == "created_at" || c.Name == "updated_at":
			cols, args = append(cols, c.Name), append(args, now)
		case c.Name == "workspace_id":
			cols, args = append(cols, c.Name), append(args, w.space.ID)
		case slices.Contains(tbl.PrimaryKey, c.Name) && strings.Contains(c.Type, "int"):
			continue
		case foreign[c.Name] != "":
			if c.Nullable {
				continue
			}

			var id int64

			switch ref := foreign[c.Name]; ref {
			case "workspaces":
				id = w.space.ID
			case "users":
				id = member
			default:
				err := db.QueryRowContext(t.Context(), "SELECT id FROM "+ref+" WHERE workspace_id = ? LIMIT 1", w.space.ID).Scan(&id)
				if err != nil {
					return nil, nil, false
				}
			}

			cols, args = append(cols, c.Name), append(args, id)
		case c.Nullable || c.Default.Valid || c.Generated:
			continue
		case c.Name == "user_id" || strings.HasSuffix(c.Name, "_by"):
			cols, args = append(cols, c.Name), append(args, member)
		case strings.Contains(c.Type, "int") || strings.Contains(c.Type, "bool"):
			cols, args = append(cols, c.Name), append(args, n)
		case strings.Contains(c.Type, "time") || strings.Contains(c.Type, "date"):
			cols, args = append(cols, c.Name), append(args, now)
		case strings.Contains(c.Type, "real") || strings.Contains(c.Type, "float") || strings.Contains(c.Type, "double") ||
			strings.Contains(c.Type, "decimal"):
			cols, args = append(cols, c.Name), append(args, 0)
		default:
			// Short enough for a varchar(26) uid, unique for the unique indexes.
			cols, args = append(cols, c.Name), append(args, fmt.Sprintf("s%d", n))
		}
	}

	return cols, args, true
}

// insertInto adds one valid row to a table that has no workspace_id, with the
// columns named in over set as given.
func (w *world) insertInto(t *testing.T, table string, member int64, over map[string]any) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	for _, tbl := range snap.Tables {
		if tbl.Name != table {
			continue
		}

		cols, args, ok := w.rowFor(t, tbl, member, over)
		if !ok {
			t.Fatalf("no parent row for %s", table)
		}

		marks := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
		if _, err := db.ExecContext(t.Context(), "INSERT INTO "+table+" ("+strings.Join(cols, ",")+") VALUES ("+marks+")", args...); err != nil {
			t.Fatalf("seeding %s: %v", table, err)
		}

		return
	}

	t.Fatalf("no table %s", table)
}
