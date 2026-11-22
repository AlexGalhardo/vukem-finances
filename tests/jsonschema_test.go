package tests

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

// validateSchema checks doc against schema. It understands the subset the
// export schema uses (no dependency exists for more): $ref into #/$defs, type
// (one name or a list), const, enum, properties, required,
// additionalProperties false, items, pattern, minLength and minimum. It
// returns the paths of every violation.
func validateSchema(schema map[string]any, doc any) []string {
	var problems []string

	var walk func(s map[string]any, v any, at string)

	walk = func(s map[string]any, v any, at string) {
		if ref, ok := s["$ref"].(string); ok {
			target := schema
			for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
				target, _ = target[part].(map[string]any)
			}

			if target == nil {
				problems = append(problems, at+": unknown $ref "+ref)

				return
			}

			walk(target, v, at)

			return
		}

		if c, ok := s["const"]; ok && fmt.Sprint(c) != fmt.Sprint(v) {
			problems = append(problems, fmt.Sprintf("%s: %v is not the constant %v", at, v, c))
		}

		if e, ok := s["enum"].([]any); ok && !slices.ContainsFunc(e, func(x any) bool { return fmt.Sprint(x) == fmt.Sprint(v) }) {
			problems = append(problems, fmt.Sprintf("%s: %v is not one of %v", at, v, e))
		}

		if t, ok := s["type"]; ok && !typeOK(t, v) {
			problems = append(problems, fmt.Sprintf("%s: %v (%T) is not of type %v", at, v, v, t))

			return
		}

		switch x := v.(type) {
		case string:
			if p, ok := s["pattern"].(string); ok && !regexp.MustCompile(p).MatchString(x) {
				problems = append(problems, fmt.Sprintf("%s: %q does not match %s", at, x, p))
			}

			if n, ok := s["minLength"].(float64); ok && float64(len(x)) < n {
				problems = append(problems, fmt.Sprintf("%s: %q is shorter than %v", at, x, n))
			}
		case float64:
			if n, ok := s["minimum"].(float64); ok && x < n {
				problems = append(problems, fmt.Sprintf("%s: %v is below %v", at, x, n))
			}
		case []any:
			if items, ok := s["items"].(map[string]any); ok {
				for i, e := range x {
					walk(items, e, fmt.Sprintf("%s[%d]", at, i))
				}
			}
		case map[string]any:
			props, _ := s["properties"].(map[string]any)

			if req, ok := s["required"].([]any); ok {
				for _, r := range req {
					if _, present := x[r.(string)]; !present {
						problems = append(problems, fmt.Sprintf("%s: missing %q", at, r))
					}
				}
			}

			for k, e := range x {
				sub, known := props[k].(map[string]any)
				if !known {
					if s["additionalProperties"] == false {
						problems = append(problems, fmt.Sprintf("%s: unexpected field %q", at, k))
					}

					continue
				}

				walk(sub, e, at+"."+k)
			}
		}
	}

	walk(schema, doc, "$")

	return problems
}

func typeOK(t any, v any) bool {
	names, ok := t.([]any)
	if !ok {
		names = []any{t}
	}

	for _, n := range names {
		switch n {
		case "null":
			if v == nil {
				return true
			}
		case "string":
			if _, ok := v.(string); ok {
				return true
			}
		case "boolean":
			if _, ok := v.(bool); ok {
				return true
			}
		case "array":
			if _, ok := v.([]any); ok {
				return true
			}
		case "object":
			if _, ok := v.(map[string]any); ok {
				return true
			}
		case "integer":
			if f, ok := v.(float64); ok && f == math.Trunc(f) {
				return true
			}
		}
	}

	return false
}

// parseJSON decodes with numbers as float64; amounts in the tests stay far
// below 2^53, so nothing is lost.
func parseJSON(body string) (any, error) {
	var v any

	err := json.Unmarshal([]byte(body), &v)

	return v, err
}
