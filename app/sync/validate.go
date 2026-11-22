package sync

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Mutation error codes.
const (
	CodeUnknownEntity  = "unknown_entity"
	CodeUnknownField   = "unknown_field"
	CodeMissingField   = "missing_field"
	CodeInvalidField   = "invalid_field"
	CodeInvalidAmount  = "invalid_amount"
	CodeInvalidDate    = "invalid_date"
	CodeInvalidUID     = "invalid_uid"
	CodeInvalidOp      = "invalid_op"
	CodeForbidden      = "forbidden"
	CodeQuotaExceeded  = "quota_exceeded"
	CodeTransferBroken = "transfer_incomplete"
	CodeTransferBad    = "transfer_invalid"
	CodeDuplicateName  = "duplicate_name"
	CodeDuplicateKey   = "duplicate_key"
	CodeRestoreWindow  = "restore_window_expired"
	CodeEditedDeleted  = "edited_deleted_row"
	CodeNotRestorable  = "not_restorable"
	CodeInvalidState   = "invalid_state"
)

// mutationError is why one mutation is rejected; the rest of the batch goes on.
type mutationError struct {
	Code    string
	Message string
	Field   string
}

func (e *mutationError) Error() string { return e.Code + ": " + e.Message }

func reject(code, field, format string, args ...any) *mutationError {
	return &mutationError{Code: code, Field: field, Message: fmt.Sprintf(format, args...)}
}

// Years a date may name: wide enough for any real record, narrow enough that
// a typo cannot make a rule that never ends.
const (
	minYear = 1900
	maxYear = 2999
)

// number reads a JSON number (decoded as float64, json.Number or an integer)
// that is a whole number.
func number(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		if n != math.Trunc(n) || math.Abs(n) > float64(models.MaxCents) {
			return 0, false
		}

		return int64(n), true
	case json.Number:
		i, err := n.Int64()

		return i, err == nil
	case int64:
		return n, true
	case int:
		return int64(n), true
	}

	return 0, false
}

// centsText writes cents the way the finance services read an amount.
func centsText(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// checkCents applies the money rules of the services to a wire amount: a whole
// number of cents within models.MaxCents that finance.ParseAmount reads back
// unchanged.
func checkCents(f FieldSpec, v any) (int64, *mutationError) {
	n, ok := number(v)
	if !ok {
		return 0, reject(CodeInvalidAmount, f.Name, "%s must be a whole number of cents.", f.Name)
	}

	if err := models.CheckCents(n); err != nil {
		return 0, reject(CodeInvalidAmount, f.Name, "%s is beyond %d cents.", f.Name, models.MaxCents)
	}

	if back, err := finance.ParseAmount(centsText(n)); err != nil || back != n {
		return 0, reject(CodeInvalidAmount, f.Name, "%s is not a valid amount.", f.Name)
	}

	if n < f.Min {
		return 0, reject(CodeInvalidAmount, f.Name, "%s must be at least %d.", f.Name, f.Min)
	}

	return n, nil
}

// ValidDate reports whether s is a strict ISO date (2026-11-30) in a sane year.
func ValidDate(s string) bool {
	t, err := time.Parse("2006-01-02", s)

	return err == nil && t.Format("2006-01-02") == s && t.Year() >= minYear && t.Year() <= maxYear
}

func cleanText(f FieldSpec, v any) (string, *mutationError) {
	s, ok := v.(string)
	if !ok || !utf8.ValidString(s) {
		return "", reject(CodeInvalidField, f.Name, "%s must be text.", f.Name)
	}

	for _, r := range s {
		if unicode.IsControl(r) {
			return "", reject(CodeInvalidField, f.Name, "%s must not hold control characters.", f.Name)
		}
	}

	if f.Name != "last4" {
		s = strings.TrimSpace(s)
	}

	if n := int64(utf8.RuneCountInString(s)); n < f.Min || n > f.Max {
		return "", reject(CodeInvalidField, f.Name, "%s must have %d to %d characters.", f.Name, f.Min, f.Max)
	}

	if f.Name == "last4" {
		for _, r := range s {
			if r < '0' || r > '9' {
				return "", reject(CodeInvalidField, f.Name, "last4 is four digits.")
			}
		}

		if s != "" && len(s) != 4 {
			return "", reject(CodeInvalidField, f.Name, "last4 is four digits.")
		}
	}

	return s, nil
}

// canonicalValue checks one wire value against its spec and returns it in the
// canonical Go type (the one StoredRow holds, so Merge compares like with like).
func (s *Service) canonicalValue(f FieldSpec, v any) (any, *mutationError) {
	if v == nil {
		if f.Nullable {
			return nil, nil //nolint:nilnil // null is the value
		}

		return nil, reject(CodeInvalidField, f.Name, "%s cannot be null.", f.Name)
	}

	switch f.Type {
	case TypeText:
		return cleanText(f, v)
	case TypeEnum:
		str, ok := v.(string)
		if !ok || !(slices.Contains(f.Enum, str) || (f.Blank && str == "")) {
			return nil, reject(CodeInvalidField, f.Name, "%s is not one of the allowed values.", f.Name)
		}

		return str, nil
	case TypeCents:
		return checkCents(f, v)
	case TypeInt:
		n, ok := number(v)
		if !ok || n < f.Min || n > f.Max {
			return nil, reject(CodeInvalidField, f.Name, "%s must be a whole number from %d to %d.", f.Name, f.Min, f.Max)
		}

		return n, nil
	case TypeBool:
		b, ok := v.(bool)
		if !ok {
			return nil, reject(CodeInvalidField, f.Name, "%s must be true or false.", f.Name)
		}

		return b, nil
	case TypeDate:
		str, ok := v.(string)
		if !ok || !ValidDate(str) {
			return nil, reject(CodeInvalidDate, f.Name, "%s must look like 2026-11-30.", f.Name)
		}

		return str, nil
	case TypeTimestamp:
		str, ok := v.(string)
		if !ok {
			return nil, reject(CodeInvalidField, f.Name, "%s must be an RFC 3339 time.", f.Name)
		}

		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return nil, reject(CodeInvalidField, f.Name, "%s must be an RFC 3339 time.", f.Name)
		}

		return t.UTC().Format(time.RFC3339), nil
	case TypeUID, TypeRef:
		str, ok := v.(string)
		if !ok || changes.ValidUID(str, s.now()) != nil {
			return nil, reject(CodeInvalidUID, f.Name, "%s must be a ULID.", f.Name)
		}

		return str, nil
	case TypeUIDList:
		list, ok := v.([]any)
		if !ok || int64(len(list)) > f.Max {
			return nil, reject(CodeInvalidField, f.Name, "%s must be a list of at most %d uids.", f.Name, f.Max)
		}

		seen := make([]string, 0, len(list))

		for _, item := range list {
			str, ok := item.(string)
			if !ok || changes.ValidUID(str, s.now()) != nil {
				return nil, reject(CodeInvalidUID, f.Name, "%s must hold ULIDs.", f.Name)
			}

			if !slices.Contains(seen, str) {
				seen = append(seen, str)
			}
		}

		slices.Sort(seen)

		out := make([]any, len(seen))
		for i, str := range seen {
			out[i] = str
		}

		return out, nil
	}

	return nil, reject(CodeInvalidField, f.Name, "%s has a type this server cannot check.", f.Name)
}

// validateFields checks the fields of a mutation against the registry: no
// field outside it (this is also what refuses server-generated fields such as
// version, workspace_id, created_at or deleted_at), every value of its type
// and, on a create, every required field present and every optional one
// defaulted.
func (s *Service) validateFields(e changes.Entity, in map[string]any, create bool) (map[string]any, *mutationError) {
	fields, missing := Fields(e)
	if len(missing) > 0 {
		return nil, reject(CodeInvalidState, "", "entity %s has fields this server cannot check: %v", e.Name, missing)
	}

	byName := make(map[string]FieldSpec, len(fields))
	for _, f := range fields {
		byName[f.Name] = f
	}

	for _, name := range slices.Sorted(maps.Keys(in)) {
		if _, ok := byName[name]; !ok {
			return nil, reject(CodeUnknownField, name, "%s is not a field of %s.", name, e.Name)
		}
	}

	out := make(map[string]any, len(fields))

	for _, f := range fields {
		v, present := in[f.Name]

		if !present {
			if !create {
				continue
			}

			if f.Required {
				return nil, reject(CodeMissingField, f.Name, "%s is required.", f.Name)
			}

			if f.Default != nil {
				out[f.Name] = f.Default
			}

			continue
		}

		cv, err := s.canonicalValue(f, v)
		if err != nil {
			return nil, err
		}

		out[f.Name] = cv
	}

	return out, nil
}
