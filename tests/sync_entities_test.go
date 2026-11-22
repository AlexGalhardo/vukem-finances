package tests

import (
	"fmt"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
)

// sampleOverrides give the few fields whose valid value depends on another row.
var sampleOverrides = map[string]map[string]any{ //nolint:gochecknoglobals // test table
	"account":     {"kind": "checking"},
	"category":    {"kind": "expense"},
	"transaction": {"kind": "expense"},
	"recurrence":  {"kind": "expense"},
}

// sampleFields builds a valid create for an entity from its field specs, so a
// new entity is covered without touching this test.
func sampleFields(t *testing.T, e changes.Entity, made map[string]string, tag string) map[string]any {
	t.Helper()

	specs, missing := syncproto.Fields(e)
	if len(missing) > 0 {
		t.Fatalf("entity %s has fields without a type: %v", e.Name, missing)
	}

	out := map[string]any{}

	for _, f := range specs {
		if v, ok := sampleOverrides[e.Name][f.Name]; ok {
			out[f.Name] = v
			continue
		}

		if f.Nullable && f.Type != syncproto.TypeRef {
			continue
		}

		switch f.Type {
		case syncproto.TypeText:
			if f.Min > 0 {
				out[f.Name] = fmt.Sprintf("%s %s", e.Name, tag)
			}
		case syncproto.TypeEnum:
			if !f.Blank {
				out[f.Name] = f.Enum[0]
			}
		case syncproto.TypeCents:
			out[f.Name] = max(f.Min, 1000)
		case syncproto.TypeInt:
			if f.Required {
				out[f.Name] = f.Min
			}
		case syncproto.TypeDate:
			out[f.Name] = "2026-11-20"
		case syncproto.TypeRef:
			if f.Nullable {
				continue
			}

			uid, ok := made[f.Ref]
			if !ok {
				t.Fatalf("entity %s needs a %s that was not made first", e.Name, f.Ref)
			}

			out[f.Name] = uid
		}
	}

	return out
}

// editOf changes one writable scalar of a row and returns the field and value.
func editOf(t *testing.T, e changes.Entity, row map[string]any) (string, any) {
	t.Helper()

	specs, _ := syncproto.Fields(e)

	for _, f := range specs {
		switch f.Type {
		case syncproto.TypeText:
			if f.Max >= 6 {
				return f.Name, "edited"
			}
		case syncproto.TypeCents:
			if f.Name != "amount_cents" || e.Name != "recurrence" {
				cur, _ := row[f.Name].(float64)

				return f.Name, int64(cur) + 1
			}
		}
	}

	t.Fatalf("entity %s has no field the test can edit", e.Name)

	return "", nil
}

// Every registered entity, including the ones later epics add, round-trips a
// create, an edit, a delete and a restore through push and pull.
func TestSyncEveryRegisteredEntity(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")

	// The token and the device are held to per-minute limits: each entity gets its own.
	fresh := func(name string) syncDeviceClient {
		return newSyncDevice(t, f.app, rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "sync"), name)
	}

	// A credit card needs an account of its own kind.
	cardAccount := newULID()
	if r := d.one(mut("account", cardAccount, "upsert", 0, map[string]any{"name": "Card account", "kind": "credit_card"})); r.Status != "applied" {
		t.Fatalf("card account: %+v", r)
	}

	made := map[string]string{}

	for _, e := range syncproto.Order() {
		t.Run(e.Name, func(t *testing.T) {
			d := fresh(e.Name)
			uid := newULID()
			fields := sampleFields(t, e, made, uid)

			if e.Name == "credit_card" {
				fields["account_uid"] = cardAccount
			}

			created := d.one(mut(e.Name, uid, "upsert", 0, fields))
			if created.Status != "applied" || created.Version != 1 {
				t.Fatalf("create %v = %+v (%+v)", fields, created, created.Error)
			}

			made[e.Name] = uid

			it := rowOf(t, d, uid)
			if it.Op != "upsert" || it.Entity != e.Name {
				t.Fatalf("pull after the create = %+v", it)
			}

			// Every field the client sent comes back, references as uids.
			for name, want := range fields {
				if got := it.Row[name]; fmt.Sprint(got) != fmt.Sprint(want) {
					t.Errorf("%s: pulled %v, pushed %v", name, got, want)
				}
			}

			field, value := editOf(t, e, it.Row)

			edited := d.one(mut(e.Name, uid, "upsert", it.Seq, map[string]any{field: value}))
			if edited.Status == "rejected" {
				t.Fatalf("edit %s=%v: %+v", field, value, edited.Error)
			}

			it = rowOf(t, d, uid)
			if fmt.Sprint(it.Row[field]) != fmt.Sprint(value) || it.Version != 2 {
				t.Errorf("after the edit %s = %v (version %d), want %v (2)", field, it.Row[field], it.Version, value)
			}

			if del := d.one(mut(e.Name, uid, "delete", it.Seq, nil)); del.Status == "rejected" {
				t.Fatalf("delete: %+v", del.Error)
			}

			if it = rowOf(t, d, uid); it.Op != "delete" || it.Row != nil {
				t.Fatalf("pull after the delete = %+v", it)
			}

			if back := d.one(mut(e.Name, uid, "restore", it.Seq, nil)); back.Status == "rejected" {
				t.Fatalf("restore: %+v", back.Error)
			}

			if it = rowOf(t, d, uid); it.Op != "upsert" || fmt.Sprint(it.Row[field]) != fmt.Sprint(value) {
				t.Errorf("pull after the restore = %+v", it)
			}
		})
	}
}
