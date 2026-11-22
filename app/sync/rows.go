package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// StoredRow is a row of the workspace as the protocol shows it: wire field
// names, references as uids, every value in its canonical Go type.
type StoredRow struct {
	ID        int64
	UID       string
	Version   int64
	Deleted   bool
	DeletedAt time.Time
	Fields    map[string]any
}

// Wire is the row as sent to a client: uid, version and the fields.
func (r StoredRow) Wire() map[string]any {
	out := make(map[string]any, len(r.Fields)+2)
	for k, v := range r.Fields {
		out[k] = v
	}

	out["uid"], out["version"] = r.UID, r.Version

	return out
}

var timeLayouts = []string{ //nolint:gochecknoglobals // constant list
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
}

func asTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), true
	case string:
		for _, l := range timeLayouts {
			if t, err := time.Parse(l, x); err == nil {
				return t.UTC(), true
			}
		}
	case []byte:
		return asTime(string(x))
	}

	return time.Time{}, false
}

func asInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	case bool:
		if x {
			return 1, true
		}

		return 0, true
	}

	return 0, false
}

// canonical turns a scanned database value into the canonical value of its type.
func canonical(f FieldSpec, v any) (any, error) {
	if b, ok := v.([]byte); ok {
		v = string(b)
	}

	if v == nil {
		return nil, nil //nolint:nilnil // a null column
	}

	switch f.Type {
	case TypeCents, TypeInt:
		n, ok := asInt(v)
		if !ok {
			return nil, fmt.Errorf("sync: %s holds %T, want a number", f.Name, v)
		}

		return n, nil
	case TypeBool:
		n, _ := asInt(v)

		return n != 0, nil
	case TypeTimestamp:
		t, ok := asTime(v)
		if !ok {
			return nil, fmt.Errorf("sync: %s holds %v, want a time", f.Name, v)
		}

		return t.Format(time.RFC3339), nil
	case TypeUIDList:
		text, _ := v.(string)

		list := []any{}
		if err := json.Unmarshal([]byte(text), &list); err != nil {
			return nil, fmt.Errorf("sync: %s holds bad JSON: %w", f.Name, err)
		}

		return list, nil
	default:
		return v, nil
	}
}

// selectRows reads rows of entity e in workspace ws, deleted ones included.
// where is a condition on alias t (the workspace is always added), order and
// limit shape the page.
func selectRows(
	ctx context.Context, ex database.Executor, e changes.Entity, ws int64, where string, args []any, order string, limit int,
) ([]StoredRow, error) {
	fields, missing := Fields(e)
	if len(missing) > 0 {
		return nil, fmt.Errorf("sync: entity %s has fields without a type: %v", e.Name, missing)
	}

	cols := []string{"t.id", "t.uid", "t.version", "t.deleted_at"}
	joins := ""

	for i, f := range fields {
		if f.Type == TypeRef {
			parent, ok := changes.Lookup(f.Ref)
			if !ok {
				return nil, fmt.Errorf("sync: %s refs unknown entity %s", e.Name, f.Ref)
			}

			alias := fmt.Sprintf("p%d", i)
			cols = append(cols, alias+".uid")
			joins += fmt.Sprintf(" LEFT JOIN %s %s ON %s.id = t.%s AND %s.workspace_id = t.workspace_id",
				parent.Table, alias, alias, f.Column, alias)

			continue
		}

		cols = append(cols, "t."+f.Column)
	}

	query := "SELECT " + strings.Join(cols, ", ") + " FROM " + e.Table + " t" + joins + " WHERE t.workspace_id = ?"
	params := []any{ws}

	if where != "" {
		query += " AND (" + where + ")"

		params = append(params, args...)
	}

	if order != "" {
		query += " ORDER BY " + order
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rs, err := ex.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rs.Close() }()

	var out []StoredRow

	for rs.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))

		for i := range vals {
			ptrs[i] = &vals[i]
		}

		if err := rs.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := StoredRow{Fields: make(map[string]any, len(fields))}
		row.ID, _ = asInt(vals[0])
		row.UID, _ = vals[1].(string)
		row.Version, _ = asInt(vals[2])

		if vals[3] != nil {
			row.Deleted = true
			row.DeletedAt, _ = asTime(vals[3])
		}

		for i, f := range fields {
			v, err := canonical(f, vals[4+i])
			if err != nil {
				return nil, err
			}

			row.Fields[f.Name] = v
		}

		out = append(out, row)
	}

	return out, rs.Err()
}

// rowsByUID loads the named rows of one entity, deleted ones included, keyed by uid.
func rowsByUID(ctx context.Context, ex database.Executor, e changes.Entity, ws int64, uids []string) (map[string]StoredRow, error) {
	out := make(map[string]StoredRow, len(uids))

	for start := 0; start < len(uids); start += 500 {
		chunk := uids[start:min(start+500, len(uids))]
		args := make([]any, len(chunk))

		for i, u := range chunk {
			args[i] = u
		}

		rows, err := selectRows(ctx, ex, e, ws, "t.uid IN ("+strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")+")", args, "", 0)
		if err != nil {
			return nil, err
		}

		for _, r := range rows {
			out[r.UID] = r
		}
	}

	return out, nil
}
