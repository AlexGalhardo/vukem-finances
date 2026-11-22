package operations

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// Route is where a surface serves an operation; the REST adapter says it.
type Route struct {
	Method string
	Path   string
}

// InventoryField is one top-level field of an input or an output.
type InventoryField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// InventorySurface says whether a surface offers the operation, and where or why not.
type InventorySurface struct {
	Surface string `json:"surface"`
	Offered bool   `json:"offered"`
	Route   string `json:"route,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// InventoryEntry is one operation as the inventory lists it.
type InventoryEntry struct {
	Name       string             `json:"name"`
	Summary    string             `json:"summary"`
	Scope      string             `json:"scope"`
	Ability    string             `json:"ability"`
	Kind       string             `json:"kind"`
	Idempotent bool               `json:"idempotent"`
	Editions   []string           `json:"editions"`
	Surfaces   []InventorySurface `json:"surfaces"`
	Input      []InventoryField   `json:"input"`
	Output     string             `json:"output"`
	Fields     []InventoryField   `json:"output_fields"`
}

// Inventory is every operation, sorted by name: the committed
// resources/operations.json. Tests read it to know what must exist on every
// surface and what a call of each one must refuse.
type Inventory struct {
	Operations []InventoryEntry `json:"operations"`
}

// BuildInventory lists the operations of r. rest says where the REST adapter
// serves an operation; an operation it does not serve (and that does not opt
// out) is listed as offered with no route, which TestSurfaceParity refuses.
func BuildInventory(r *Registry, rest func(name string) (Route, bool)) Inventory {
	metas := r.Operations()
	slices.SortFunc(metas, func(a, b Meta) int { return strings.Compare(a.Name, b.Name) })

	out := Inventory{Operations: make([]InventoryEntry, 0, len(metas))}

	for _, m := range metas {
		editions := []string{string(config.EditionPersonal), string(config.EditionSaaS)}
		if len(m.Editions) > 0 {
			editions = nil
			for _, e := range m.Editions {
				editions = append(editions, string(e))
			}
		}

		e := InventoryEntry{
			Name: m.Name, Summary: m.Summary, Scope: m.Scope, Ability: m.Ability, Kind: string(m.Kind), Idempotent: m.Idempotent,
			Editions: editions, Input: fieldsOf(m.Input), Output: typeName(m.Output), Fields: fieldsOf(m.Output),
		}

		for _, s := range Surfaces() {
			is := InventorySurface{Surface: string(s), Offered: m.Offered(s)}
			if !is.Offered {
				is.Reason = m.OptOut[s]
			}

			if s == SurfaceREST && is.Offered && rest != nil {
				if route, ok := rest(m.Name); ok {
					is.Route = route.Method + " " + route.Path
				}
			}

			e.Surfaces = append(e.Surfaces, is)
		}

		out.Operations = append(out.Operations, e)
	}

	return out
}

// JSON renders the inventory: indented, sorted, with a final newline, so the
// same registry always writes the same bytes.
func (i Inventory) JSON() ([]byte, error) {
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(i); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func fieldsOf(t reflect.Type) []InventoryField {
	out := []InventoryField{}

	if t.Kind() != reflect.Struct {
		return out
	}

	for f := range t.Fields() {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" || !f.IsExported() {
			continue
		}

		if name == "" {
			name = f.Name
		}

		validate := f.Tag.Get("validate")
		required := strings.HasPrefix(validate, "required") || strings.Contains(validate, ",required")

		out = append(out, InventoryField{Name: name, Type: typeName(f.Type), Required: required})
	}

	return out
}

// typeName is a short, stable name of a type: the name of a struct, an
// element type for a slice, the name of the page for a list.
func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return typeName(t.Elem()) + "?"
	case reflect.Slice:
		return "[]" + typeName(t.Elem())
	case reflect.Struct:
		if data, ok := t.FieldByName("Data"); ok && data.Type.Kind() == reflect.Slice && strings.Contains(t.Name(), "[") {
			return "List<" + typeName(data.Type.Elem()) + ">"
		}

		return t.Name()
	}

	return t.Kind().String()
}
