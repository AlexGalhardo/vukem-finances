package changes_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

type fields = map[string]any

func live(f fields) changes.Row { return changes.Row{Exists: true, Version: 3, Fields: f} }

func tombstone(f fields) changes.Row {
	return changes.Row{Exists: true, Deleted: true, Version: 4, Fields: f}
}

// TestMerge has at least one case per conflict rule of the specification
// (docs/specs/sync-data-model.md); the rule is the prefix of the name.
func TestMerge(t *testing.T) {
	t.Parallel()

	txn := fields{"description": "lunch", "amount_cents": int64(1000), "kind": "expense"}

	cases := []struct {
		name      string
		current   changes.Row
		in        changes.Mutation
		since     []changes.Change
		outcome   string
		apply     fields
		conflicts []string // "field:kind"
		code      string
	}{
		{
			name:    "rule 1: only the sent fields are written",
			current: live(txn),
			in:      changes.Mutation{BaseSeq: 5, Fields: fields{"description": "dinner"}},
			outcome: changes.OutcomeApplied, apply: fields{"description": "dinner"},
		},
		{
			name:    "rule 1: a mutation that changes nothing is a noop",
			current: live(txn),
			in:      changes.Mutation{BaseSeq: 5, Fields: fields{"description": "lunch", "amount_cents": float64(1000)}},
			outcome: changes.OutcomeNoop,
		},
		{
			name:    "rule 2: a field nobody touched applies cleanly beside a later change of another",
			current: live(fields{"description": "lunch", "amount_cents": int64(2000)}),
			in:      changes.Mutation{BaseSeq: 5, Fields: fields{"description": "dinner"}},
			since:   []changes.Change{{Seq: 6, Op: "upsert", Fields: []string{"amount_cents"}}},
			outcome: changes.OutcomeApplied, apply: fields{"description": "dinner"},
		},
		{
			name:    "rule 2: changes at or before base_seq do not conflict",
			current: live(fields{"description": "web edit"}),
			in:      changes.Mutation{BaseSeq: 6, Fields: fields{"description": "phone edit"}},
			since:   []changes.Change{{Seq: 6, Op: "upsert", Fields: []string{"description"}}},
			outcome: changes.OutcomeApplied, apply: fields{"description": "phone edit"},
		},
		{
			name:    "rule 3: the same field changed on both sides, the later arrival wins and the old value is kept",
			current: live(fields{"limit_cents": int64(50000)}),
			in:      changes.Mutation{BaseSeq: 5, Device: "phone", Fields: fields{"limit_cents": float64(60000)}},
			since:   []changes.Change{{Seq: 7, Op: "upsert", Fields: []string{"limit_cents"}, Device: "web"}},
			outcome: changes.OutcomeApplied, apply: fields{"limit_cents": float64(60000)},
			conflicts: []string{"limit_cents:field"},
		},
		{
			name:    "rule 3: a money field is merged like any other",
			current: live(fields{"amount_cents": int64(1000), "description": "a"}),
			in:      changes.Mutation{BaseSeq: 1, Fields: fields{"amount_cents": float64(9007199254740991), "description": "b"}},
			since:   []changes.Change{{Seq: 2, Fields: []string{"amount_cents"}}},
			outcome: changes.OutcomeApplied, apply: fields{"amount_cents": float64(9007199254740991), "description": "b"},
			conflicts: []string{"amount_cents:field"},
		},
		{
			name:    "rule 3: both sides wrote the same value, nothing to report",
			current: live(fields{"name": "Food"}),
			in:      changes.Mutation{BaseSeq: 1, Fields: fields{"name": "Food"}},
			since:   []changes.Change{{Seq: 2, Fields: []string{"name"}}},
			outcome: changes.OutcomeNoop,
		},
		{
			name:    "rule 4: an edit of a row deleted meanwhile is dropped and reported",
			current: tombstone(txn),
			in:      changes.Mutation{BaseSeq: 5, Device: "phone", Fields: fields{"description": "late edit"}},
			since:   []changes.Change{{Seq: 6, Op: "delete", Fields: []string{"deleted_at"}}},
			outcome: changes.OutcomeDropped, conflicts: []string{"description:edited_deleted_row"},
		},
		{
			name:    "rule 4: only an explicit restore brings the row back",
			current: tombstone(txn),
			in:      changes.Mutation{BaseSeq: 5, Restore: true, Fields: fields{}},
			outcome: changes.OutcomeApplied, apply: fields{"deleted_at": false},
		},
		{
			name:    "rule 4: a delete over a concurrent edit deletes",
			current: live(txn),
			in:      changes.Mutation{BaseSeq: 5, Delete: true},
			since:   []changes.Change{{Seq: 6, Op: "upsert", Fields: []string{"description"}}},
			outcome: changes.OutcomeApplied, apply: fields{"deleted_at": true},
		},
		{
			name:    "rule 4: deleting a tombstone again is a noop",
			current: tombstone(txn),
			in:      changes.Mutation{Delete: true},
			outcome: changes.OutcomeNoop,
		},
		{
			name:    "rule 5: a create of an unused uid inserts every field",
			current: changes.Row{},
			in:      changes.Mutation{Create: true, Fields: fields{"name": "Main", "kind": "cash"}},
			outcome: changes.OutcomeApplied, apply: fields{"name": "Main", "kind": "cash"},
		},
		{
			name:    "rule 5: the retry of a create with equal fields is idempotent",
			current: live(fields{"name": "Main", "kind": "cash"}),
			in:      changes.Mutation{Create: true, Fields: fields{"name": "Main", "kind": "cash"}},
			since:   []changes.Change{{Seq: 1, Op: "upsert", Fields: []string{"name", "kind"}}},
			outcome: changes.OutcomeNoop,
		},
		{
			name:    "rule 5: a create of an existing uid with other fields is an edit under rule 3",
			current: live(fields{"name": "Main", "kind": "cash"}),
			in:      changes.Mutation{Create: true, Fields: fields{"name": "Primary", "kind": "cash"}},
			since:   []changes.Change{{Seq: 1, Op: "upsert", Fields: []string{"name", "kind"}}},
			outcome: changes.OutcomeApplied, apply: fields{"name": "Primary"},
			conflicts: []string{"name:field"},
		},
		{
			name:    "rule 6: a missing parent is rejected as unknown_reference",
			current: changes.Row{},
			in: changes.Mutation{
				Create:  true,
				Fields:  fields{"account_id": "01J0000000000000000000000A", "amount_cents": float64(5)},
				Parents: map[string]changes.ParentState{"account_id": changes.ParentMissing},
			},
			outcome: changes.OutcomeRejected, code: changes.CodeUnknownReference,
		},
		{
			name:    "rule 6: a deleted parent is rejected as parent_deleted",
			current: live(txn),
			in: changes.Mutation{
				Fields:  fields{"category_id": "01J0000000000000000000000B"},
				Parents: map[string]changes.ParentState{"category_id": changes.ParentDeleted},
			},
			outcome: changes.OutcomeRejected, code: changes.CodeParentDeleted,
		},
		{
			name:    "rule 6: a null reference is not checked",
			current: live(fields{"category_id": "x"}),
			in: changes.Mutation{
				Fields:  fields{"category_id": nil},
				Parents: map[string]changes.ParentState{"category_id": changes.ParentDeleted},
			},
			outcome: changes.OutcomeApplied, apply: fields{"category_id": nil},
		},
		{
			name:    "an edit of a row the server never saw is rejected",
			current: changes.Row{},
			in:      changes.Mutation{Fields: fields{"name": "x"}},
			outcome: changes.OutcomeRejected, code: changes.CodeUnknownRow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := changes.Merge(tc.current, tc.in, tc.since)

			if got.Outcome != tc.outcome || got.Code != tc.code {
				t.Fatalf("outcome %q code %q, want %q %q (%+v)", got.Outcome, got.Code, tc.outcome, tc.code, got)
			}

			if !reflect.DeepEqual(got.Apply, tc.apply) && !(len(got.Apply) == 0 && len(tc.apply) == 0) {
				t.Errorf("apply = %v, want %v", got.Apply, tc.apply)
			}

			var kinds []string
			for _, c := range got.Conflicts {
				kinds = append(kinds, c.Field+":"+c.Kind)
			}

			if !slices.Equal(kinds, tc.conflicts) {
				t.Errorf("conflicts = %v, want %v", kinds, tc.conflicts)
			}
		})
	}
}

func TestMergeReportsWhatWasLost(t *testing.T) {
	t.Parallel()

	got := changes.Merge(
		live(fields{"limit_cents": int64(50000)}),
		changes.Mutation{BaseSeq: 5, Device: "phone", Fields: fields{"limit_cents": float64(60000)}},
		[]changes.Change{{Seq: 7, Fields: []string{"limit_cents"}, Device: "web"}},
	)

	if len(got.Conflicts) != 1 || got.Conflicts[0].LostValue != int64(50000) || got.Conflicts[0].LostByDevice != "web" {
		t.Fatalf("conflicts = %+v, want the overwritten 50000 of the web device", got.Conflicts)
	}

	dropped := changes.Merge(
		tombstone(fields{"description": "x"}),
		changes.Mutation{Device: "phone", Fields: fields{"description": "late"}},
		nil,
	)

	if len(dropped.Conflicts) != 1 || dropped.Conflicts[0].LostValue != "late" || dropped.Conflicts[0].LostByDevice != "phone" {
		t.Fatalf("dropped conflicts = %+v, want the phone's late edit", dropped.Conflicts)
	}
}

// TestMergeIsPure checks that Merge leaves its inputs alone and gives the
// same answer twice.
func TestMergeIsPure(t *testing.T) {
	t.Parallel()

	current := live(fields{"name": "a"})
	in := changes.Mutation{BaseSeq: 1, Fields: fields{"name": "b"}}
	since := []changes.Change{{Seq: 2, Fields: []string{"name"}}}
	before := maps.Clone(current.Fields)

	first := changes.Merge(current, in, since)
	second := changes.Merge(current, in, since)

	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(before, current.Fields) || in.Fields["name"] != "b" {
		t.Fatalf("Merge is not pure: %+v vs %+v", first, second)
	}

	first.Apply["name"] = "changed by the caller"

	if in.Fields["name"] != "b" {
		t.Fatal("the result shares its map with the mutation")
	}
}
