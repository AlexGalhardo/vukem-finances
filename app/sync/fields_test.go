package sync_test

import (
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
)

// Every writable column of every registered entity has a wire type, so the
// push can validate it. Adding a column to the registry without one fails here.
func TestEveryRegisteredFieldHasAType(t *testing.T) {
	t.Parallel()

	for _, e := range changes.Entities() {
		fields, missing := syncproto.Fields(e)
		if len(missing) > 0 {
			t.Errorf("entity %s: columns without a field type: %v", e.Name, missing)
		}

		if len(fields) != len(e.Fields) {
			t.Errorf("entity %s: %d typed fields for %d columns", e.Name, len(fields), len(e.Fields))
		}

		for _, f := range fields {
			if f.Type == "" || !f.Writable {
				t.Errorf("entity %s field %s has no type or is not writable: %+v", e.Name, f.Name, f)
			}
		}
	}
}

func TestFieldsReportsAnUntypedColumn(t *testing.T) {
	t.Parallel()

	e := changes.Entity{Name: "account", Table: "accounts", Fields: []string{"name", "mystery"}}

	fields, missing := syncproto.Fields(e)
	if !slices.Equal(missing, []string{"mystery"}) || len(fields) != 1 {
		t.Errorf("Fields = %v, missing %v; want one typed field and mystery missing", fields, missing)
	}
}

// Parents come before the rows that point at them.
func TestOrderPutsParentsFirst(t *testing.T) {
	t.Parallel()

	pos := map[string]int{}
	for i, e := range syncproto.Order() {
		pos[e.Name] = i
	}

	for _, e := range changes.Entities() {
		for _, r := range e.Refs {
			if r.Entity != e.Name && pos[r.Entity] >= pos[e.Name] {
				t.Errorf("%s comes before its parent %s", e.Name, r.Entity)
			}
		}
	}
}
