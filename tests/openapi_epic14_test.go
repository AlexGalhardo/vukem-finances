package tests

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// epic14Snapshot is docs/openapi.json as epic 14 shipped it (commit d76cb53,
// the base of epic 30). It never changes: it is the contract the registry
// refactor must not move.
const epic14Snapshot = "../testdata/openapi/epic14.json"

// addedByEpic30 are the operation prefixes epic 30 task 6b adds.
var addedByEpic30 = []string{"cards.", "statements.", "recurrences.", "goals.", "reports.", "export."} //nolint:gochecknoglobals // a table

func readDocument(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}

	return doc
}

// resolved returns node with every $ref into components replaced by what it
// points at, so two operations are compared by what a client sees and not by
// how the schemas are named.
func resolved(doc map[string]any, node any, seen map[string]bool) any {
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			if seen[ref] {
				return map[string]any{"$ref": ref}
			}

			target := doc
			for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
				target, _ = target[part].(map[string]any)
			}

			next := map[string]bool{ref: true}
			for k := range seen {
				next[k] = true
			}

			return resolved(doc, target, next)
		}

		out := make(map[string]any, len(v))
		for k, x := range v {
			out[k] = resolved(doc, x, seen)
		}

		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = resolved(doc, x, seen)
		}

		return out
	}

	return node
}

func operationsOf(doc map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	paths, _ := doc["paths"].(map[string]any)

	for path, methods := range paths {
		for method, op := range methods.(map[string]any) {
			out[strings.ToUpper(method)+" "+path], _ = op.(map[string]any)
		}
	}

	return out
}

// TestOpenAPIOperationsOfEpic14DidNotChange: every operation of the epic 14
// document is in the current one, byte for byte once its schemas are resolved
// (parameters, body, answers, scope), and the only additions are the operations
// of epic 30 task 6b.
func TestOpenAPIOperationsOfEpic14DidNotChange(t *testing.T) {
	t.Parallel()

	before := readDocument(t, epic14Snapshot)
	after := readDocument(t, openAPIFile)

	old, now := operationsOf(before), operationsOf(after)

	if len(old) == 0 {
		t.Fatal("the epic 14 snapshot has no operations")
	}

	for key, op := range old {
		cur, ok := now[key]
		if !ok {
			t.Errorf("%s (%v) is gone from the document", key, op["operationId"])
			continue
		}

		if !reflect.DeepEqual(resolved(before, op, nil), resolved(after, cur, nil)) {
			t.Errorf("%s (%v) changed", key, op["operationId"])
		}
	}

	var added []string

	for key, op := range now {
		if _, ok := old[key]; ok {
			continue
		}

		id, _ := op["operationId"].(string)
		added = append(added, id)

		if !slices.ContainsFunc(addedByEpic30, func(p string) bool { return strings.HasPrefix(id, p) }) {
			t.Errorf("%s (%s) is new and is not one of cards, statements, recurrences, goals, reports, export", key, id)
		}
	}

	for _, p := range addedByEpic30 {
		if !slices.ContainsFunc(added, func(id string) bool { return strings.HasPrefix(id, p) }) {
			t.Errorf("the document has no %s operation", p)
		}
	}

	// What the operations share outside them did not move either.
	for _, k := range []string{"openapi", "info", "servers"} {
		if !reflect.DeepEqual(before[k], after[k]) {
			t.Errorf("%s changed", k)
		}
	}

	bc, _ := before["components"].(map[string]any)
	ac, _ := after["components"].(map[string]any)

	for _, k := range []string{"securitySchemes", "responses"} {
		if !reflect.DeepEqual(bc[k], ac[k]) {
			t.Errorf("components.%s changed", k)
		}
	}

	// A component of epic 14 is still there as it was (the new ones are added).
	bs, _ := bc["schemas"].(map[string]any)
	as, _ := ac["schemas"].(map[string]any)

	for name, schema := range bs {
		if !reflect.DeepEqual(resolved(before, schema, nil), resolved(after, as[name], nil)) {
			t.Errorf("schema %s changed", name)
		}
	}
}
