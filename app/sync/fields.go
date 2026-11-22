// Package sync is the server side of the sync protocol (docs/specs/sync-protocol.md):
// devices, pull, snapshot and push over the entity registry of app/changes.
package sync

import (
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Field types on the wire.
const (
	TypeText      = "text"
	TypeEnum      = "enum"
	TypeCents     = "cents"
	TypeInt       = "int"
	TypeBool      = "bool"
	TypeDate      = "date"
	TypeTimestamp = "timestamp"
	TypeUID       = "uid"
	TypeUIDList   = "uid_list"
	TypeRef       = "ref"
)

// FieldSpec is how one wire field is validated and stored.
type FieldSpec struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Writable bool   `json:"writable"`
	Nullable bool   `json:"nullable"`
	Required bool   `json:"required"`
	// Min and Max bound a text length (in characters), an int, a list length
	// or a cents amount.
	Min int64 `json:"min,omitempty"`
	Max int64 `json:"max,omitempty"`
	// Enum lists the values of an enum; "" is also allowed when Blank is true.
	Enum  []string `json:"enum,omitempty"`
	Blank bool     `json:"blank,omitempty"`
	// Ref is the entity a ref points at.
	Ref string `json:"ref,omitempty"`
	// Column is the database column behind the wire name.
	Column string `json:"-"`
	// Default fills an optional field that a create leaves out.
	Default any `json:"-"`
}

func text(name string, minLen, maxLen int64) FieldSpec {
	return FieldSpec{Name: name, Type: TypeText, Min: minLen, Max: maxLen, Required: minLen > 0, Default: ""}
}

func enum(name string, values []string, blank bool) FieldSpec {
	f := FieldSpec{Name: name, Type: TypeEnum, Enum: values, Blank: blank, Required: !blank}
	if blank {
		f.Default = ""
	}

	return f
}

func cents(name string, minCents int64, required bool) FieldSpec {
	f := FieldSpec{Name: name, Type: TypeCents, Min: minCents, Max: models.MaxCents, Required: required}
	if !required {
		f.Default = int64(0)
	}

	return f
}

func integer(name string, minValue, maxValue int64, def any) FieldSpec {
	return FieldSpec{Name: name, Type: TypeInt, Min: minValue, Max: maxValue, Default: def, Required: def == nil}
}

func date(name string) FieldSpec { return FieldSpec{Name: name, Type: TypeDate, Required: true} }

func ref(name, entity string) FieldSpec {
	return FieldSpec{Name: name, Type: TypeRef, Ref: entity, Required: true}
}

// optional makes a field nullable and not required; an absent one is null.
func optional(f FieldSpec) FieldSpec {
	f.Nullable, f.Required, f.Default = true, false, nil

	return f
}

const (
	maxName        = 120
	maxDescription = 200
)

// specs is the wire description of every writable field of every entity. A
// registered field without an entry fails TestSyncEveryRegisteredEntity.
var specs = map[string][]FieldSpec{ //nolint:gochecknoglobals // constant table
	"account": {
		text("name", 1, 80),
		enum("kind", append(models.AccountKinds(), models.AccountCreditCard), false),
		cents("opening_cents", -models.MaxCents, false),
		optional(FieldSpec{Name: "archived_at", Type: TypeTimestamp}),
		integer("position", 0, 1_000_000, int64(0)),
	},
	"category": {
		text("name", 1, 80),
		enum("kind", []string{models.KindIncome, models.KindExpense}, false),
		optional(ref("parent_uid", "category")),
		enum("color", models.CategoryColors, true),
		enum("icon", models.CategoryIcons, true),
		optional(FieldSpec{Name: "archived_at", Type: TypeTimestamp}),
	},
	"tag": {
		text("name", 1, models.MaxTagNameLength),
		enum("color", models.CategoryColors, true),
	},
	"transaction": {
		ref("account_uid", "account"),
		optional(ref("category_uid", "category")),
		enum("kind", []string{models.KindIncome, models.KindExpense}, false),
		cents("amount_cents", 1, true),
		text("description", 0, maxDescription),
		date("occurred_on"),
		optional(FieldSpec{Name: "transfer_uid", Type: TypeUID}),
		{Name: "tag_uids", Type: TypeUIDList, Max: models.MaxTagsPerTransaction, Default: []any{}},
		optional(FieldSpec{Name: "statement_ref", Type: TypeDate}),
		optional(FieldSpec{Name: "recurrence_uid", Type: TypeUID}),
		optional(FieldSpec{Name: "occurrence_on", Type: TypeDate}),
		optional(FieldSpec{Name: "installment_group_uid", Type: TypeUID}),
		optional(FieldSpec{Name: "installment_no", Type: TypeInt, Min: 1, Max: 9999}),
		optional(FieldSpec{Name: "installment_total", Type: TypeInt, Min: 1, Max: 9999}),
	},
	"budget": {
		optional(ref("category_uid", "category")),
		cents("limit_cents", 1, true),
		{Name: "rollover", Type: TypeBool, Default: false},
	},
	"credit_card": {
		ref("account_uid", "account"),
		cents("limit_cents", 0, false),
		integer("closing_day", 1, 31, nil),
		integer("due_day", 1, 31, nil),
		optional(ref("payment_account_uid", "account")),
		enum("brand", finance.CardBrands, true),
		{Name: "last4", Type: TypeText, Min: 0, Max: 4, Default: ""},
	},
	"recurrence": {
		ref("account_uid", "account"),
		optional(ref("category_uid", "category")),
		enum("kind", []string{models.KindIncome, models.KindExpense}, false),
		cents("amount_cents", 1, true),
		text("description", 0, maxDescription),
		enum("frequency", []string{"daily", "weekly", "monthly", "yearly"}, false),
		integer("interval", 1, 999, int64(1)),
		date("start_date"),
		optional(FieldSpec{Name: "end_date", Type: TypeDate}),
		optional(FieldSpec{Name: "max_count", Type: TypeInt, Min: 1, Max: 9999}),
		optional(FieldSpec{Name: "generated_through", Type: TypeDate}),
		{Name: "active", Type: TypeBool, Default: true},
	},
	"goal": {
		text("name", 1, maxName),
		cents("target_cents", 1, true),
		optional(FieldSpec{Name: "target_date", Type: TypeDate}),
		optional(FieldSpec{Name: "archived_at", Type: TypeTimestamp}),
	},
	"goal_contribution": {
		ref("goal_uid", "goal"),
		cents("amount_cents", -models.MaxCents, true),
		date("on_date"),
		text("note", 0, 200),
	},
}

// wireName is the name a column travels under: a reference column ends in
// "_uid" instead of "_id" (account_id is account_uid, parent_id is parent_uid).
func wireName(column string, isRef bool) string {
	if isRef {
		return strings.TrimSuffix(column, "_id") + "_uid"
	}

	return column
}

// Fields returns the specs of an entity in registry order with the database
// column of each. A registered column without a spec comes back in missing, so
// a test fails when someone adds a column and forgets its type.
func Fields(e changes.Entity) (fields []FieldSpec, missing []string) {
	refs := map[string]bool{}
	for _, r := range e.Refs {
		refs[r.Column] = true
	}

	byName := map[string]FieldSpec{}
	for _, f := range specs[e.Name] {
		byName[f.Name] = f
	}

	for _, col := range e.Fields {
		f, ok := byName[wireName(col, refs[col])]
		if !ok {
			missing = append(missing, col)
			continue
		}

		f.Column, f.Writable = col, true
		fields = append(fields, f)
	}

	return fields, missing
}

// depth is how many parents deep an entity sits: parents sync first.
func depth(e changes.Entity, seen map[string]bool) int {
	if seen[e.Name] {
		return 0
	}

	seen[e.Name] = true
	defer delete(seen, e.Name)

	d := 0

	for _, r := range e.Refs {
		if r.Entity == e.Name {
			continue
		}

		if p, ok := changes.Lookup(r.Entity); ok {
			d = max(d, depth(p, seen)+1)
		}
	}

	return d
}

// Order lists the registered entities with parents first, registry order
// breaking ties.
func Order() []changes.Entity {
	all := changes.Entities()
	rank := map[string]int{}

	for _, e := range all {
		rank[e.Name] = depth(e, map[string]bool{})
	}

	slices.SortStableFunc(all, func(a, b changes.Entity) int { return rank[a.Name] - rank[b.Name] })

	return all
}
