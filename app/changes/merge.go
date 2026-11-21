package changes

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Outcomes of Merge.
const (
	// OutcomeApplied: Result.Apply holds the columns to write.
	OutcomeApplied = "applied"
	// OutcomeNoop: nothing to write (an idempotent retry, or equal values).
	OutcomeNoop = "noop"
	// OutcomeDropped: the mutation was discarded (an edit of a deleted row).
	OutcomeDropped = "dropped"
	// OutcomeRejected: the mutation is refused; see Result.Code.
	OutcomeRejected = "rejected"
)

// Rejection codes.
const (
	CodeUnknownReference = "unknown_reference"
	CodeParentDeleted    = "parent_deleted"
	CodeUnknownRow       = "unknown_row"
)

// ParentState says what the server found when it looked a parent up by uid.
type ParentState int

// Parent states; the zero value is a parent that was found alive.
const (
	ParentOK ParentState = iota
	ParentMissing
	ParentDeleted
)

// Row is the current state of a row on the server. Exists is false when the
// uid is unused; Deleted marks a tombstone. Fields hold the writable columns
// by name.
type Row struct {
	Exists  bool
	Deleted bool
	Version int64
	Fields  map[string]any
}

// Mutation is what a client sends for one row: only the fields it changed,
// and the last seq it had seen for the row.
type Mutation struct {
	// Create is true when the client made the row (a retry of a lost create
	// arrives as Create again).
	Create  bool
	Delete  bool
	Restore bool
	BaseSeq int64
	Device  string
	Fields  map[string]any
	// Parents is the state of the parent behind each reference field the
	// mutation sets (for example "account_id"), looked up by the caller.
	Parents map[string]ParentState
}

// Change is one logged change of the row after the client's base_seq.
type Change struct {
	Seq    int64
	Op     string
	Fields []string
	Device string
}

// Conflict is a value that lost: it is stored in sync_conflicts and returned
// to the client so it can tell the user.
type Conflict struct {
	Field        string
	Kind         string
	LostValue    any
	LostByDevice string
}

// Result is the verdict of Merge.
type Result struct {
	Outcome string
	// Code and Field name a rejection.
	Code  string
	Field string
	// Apply is the set of columns to write; "deleted_at" is a bool here:
	// true deletes, false restores.
	Apply     map[string]any
	Conflicts []Conflict
}

// Merge decides what happens when a mutation reaches the server, following
// docs/specs/sync-data-model.md. It is pure: it reads nothing and writes
// nothing.
//
//  1. A mutation carries base_seq and only the fields it changed.
//  2. A field no later change touched applies cleanly.
//  3. A field both sides changed is a conflict: the later arrival (this
//     mutation) wins and the value it overwrote is reported.
//  4. Delete wins over a concurrent edit: an edit of a deleted row is dropped,
//     reported as edited_deleted_row; only an explicit restore revives it.
//  5. Creating an existing uid with equal fields is an idempotent retry;
//     with different fields it is an edit (rule 3).
//  6. A missing or deleted parent rejects the mutation; the caller carries
//     on with the rest of the batch.
//  7. Derived values are not fields and never reach Merge.
func Merge(current Row, in Mutation, since []Change) Result {
	if res, ok := checkParents(in); ok {
		return res
	}

	if !current.Exists {
		if !in.Create {
			return Result{Outcome: OutcomeRejected, Code: CodeUnknownRow}
		}

		return Result{Outcome: OutcomeApplied, Apply: maps.Clone(in.Fields)}
	}

	if current.Deleted {
		return mergeIntoDeleted(current, in, since)
	}

	if in.Restore {
		return Result{Outcome: OutcomeNoop}
	}

	if in.Delete {
		return Result{Outcome: OutcomeApplied, Apply: map[string]any{"deleted_at": true}}
	}

	return mergeFields(current, in, since)
}

func checkParents(in Mutation) (Result, bool) {
	if in.Delete {
		return Result{}, false
	}

	for _, field := range slices.Sorted(maps.Keys(in.Parents)) {
		if v, set := in.Fields[field]; !set || v == nil {
			continue
		}

		switch in.Parents[field] {
		case ParentMissing:
			return Result{Outcome: OutcomeRejected, Code: CodeUnknownReference, Field: field}, true
		case ParentDeleted:
			return Result{Outcome: OutcomeRejected, Code: CodeParentDeleted, Field: field}, true
		case ParentOK:
		}
	}

	return Result{}, false
}

func mergeIntoDeleted(current Row, in Mutation, since []Change) Result {
	switch {
	case in.Restore:
		res := mergeFields(Row{Exists: true, Fields: current.Fields}, in, since)
		if res.Apply == nil {
			res.Apply = map[string]any{}
		}

		res.Outcome = OutcomeApplied
		res.Apply["deleted_at"] = false

		return res
	case in.Delete:
		return Result{Outcome: OutcomeNoop}
	}

	res := Result{Outcome: OutcomeDropped}

	for _, field := range slices.Sorted(maps.Keys(in.Fields)) {
		res.Conflicts = append(res.Conflicts, Conflict{
			Field: field, Kind: models.ConflictEditedDeleted, LostValue: in.Fields[field], LostByDevice: in.Device,
		})
	}

	return res
}

func mergeFields(current Row, in Mutation, since []Change) Result {
	touched := map[string]Change{}

	for _, c := range since {
		if c.Seq <= in.BaseSeq {
			continue
		}

		for _, f := range c.Fields {
			if prev, ok := touched[f]; !ok || c.Seq > prev.Seq {
				touched[f] = c
			}
		}
	}

	res := Result{Outcome: OutcomeNoop}

	for _, field := range slices.Sorted(maps.Keys(in.Fields)) {
		want := in.Fields[field]
		have := current.Fields[field]

		if equalValues(have, want) {
			continue
		}

		if c, clash := touched[field]; clash {
			res.Conflicts = append(res.Conflicts, Conflict{
				Field: field, Kind: models.ConflictField, LostValue: have, LostByDevice: c.Device,
			})
		}

		if res.Apply == nil {
			res.Apply = map[string]any{}
		}

		res.Apply[field] = want
		res.Outcome = OutcomeApplied
	}

	return res
}

// equalValues compares values that may come from JSON (float64) or from the
// database (int64): numbers compare as numbers, which is exact up to 2^53 - 1,
// the money cap.
func equalValues(a, b any) bool {
	fa, aok := number(a)
	fb, bok := number(b)

	if aok && bok {
		return fa == fb
	}

	return reflect.DeepEqual(a, b)
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()

		return f, err == nil
	}

	return 0, false
}
