// Package cli is the command line adapter over the operation registry: every
// operation offered on the CLI surface becomes `<resource> <verb>`, with its
// flags read from the input type, and runs through operations.Invoke in local
// mode or through the REST API in remote mode.
package cli

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// FieldKind is how a flag carries the value of an input field.
type FieldKind int

// The kinds of field an input has.
const (
	KindString FieldKind = iota
	KindBool
	KindInt
	KindStrings
)

// Field is one input field of an operation and the flag that sets it.
type Field struct {
	// Name is the JSON name of the field.
	Name string
	// Flag is the name of the flag: the JSON name with dashes.
	Flag string
	Kind FieldKind
	// Cents marks an amount in cents: its flag is --x-cents, and --x takes a
	// decimal amount ("12.50", "1.234,56").
	Cents    bool
	Required bool
	// Positional is the place of the field among the positional arguments, or
	// -1: a field named in the REST path is given as an argument.
	Positional int
}

// Command is one operation as a command of the CLI.
type Command struct {
	Resource string
	// Verb is the rest of the operation name, with dashes: goals.contributions.add
	// is `goals contributions-add`.
	Verb   string
	Op     operations.Meta
	Method string
	Path   string
	Fields []Field
	// Params are the fields named in the REST path, in order.
	Params []string
}

// Name is the command as typed, without the program name.
func (c Command) Name() string { return c.Resource + " " + c.Verb }

// Destructive reports whether the command removes or archives data.
func (c Command) Destructive() bool { return c.Op.Kind == operations.KindDestructive }

// Tree is every command of the CLI.
type Tree struct {
	Commands []Command
}

// Resources lists the resources in order, once each.
func (t *Tree) Resources() []string {
	var out []string

	for _, c := range t.Commands {
		if !slices.Contains(out, c.Resource) {
			out = append(out, c.Resource)
		}
	}

	return out
}

// Of lists the commands of a resource.
func (t *Tree) Of(resource string) []Command {
	var out []Command

	for _, c := range t.Commands {
		if c.Resource == resource {
			out = append(out, c)
		}
	}

	return out
}

// Find returns the command for an operation name.
func (t *Tree) Find(op string) (Command, bool) {
	for _, c := range t.Commands {
		if c.Op.Name == op {
			return c, true
		}
	}

	return Command{}, false
}

// Route says where the REST adapter serves an operation: the method and the
// path with {placeholders}.
type Route func(op string) (method, path string, ok bool)

// reservedFlags are the flags every command has; an input field cannot use one.
var reservedFlags = []string{"output", "o", "input", "yes", "server", "local", "if-match", "idempotency-key", "out", "help", "h"}

// BuildTree makes a command of every operation the registry offers on the CLI.
func BuildTree(reg *operations.Registry, route Route) (*Tree, error) {
	t := &Tree{}

	seen := map[string]string{}

	for _, m := range reg.Operations() {
		if !m.Offered(operations.SurfaceCLI) {
			continue
		}

		resource, rest, ok := strings.Cut(m.Name, ".")
		if !ok || resource == "" || rest == "" {
			return nil, fmt.Errorf("cli: operation %q is not resource.verb", m.Name)
		}

		c := Command{Resource: resource, Verb: strings.ReplaceAll(rest, ".", "-"), Op: m}

		if prev, dup := seen[c.Name()]; dup {
			return nil, fmt.Errorf("cli: %s and %s are both %q", prev, m.Name, c.Name())
		}

		seen[c.Name()] = m.Name

		if route != nil {
			c.Method, c.Path, _ = route(m.Name)
			c.Params = pathParams(c.Path)
		}

		fields, err := fieldsOf(m, c.Params)
		if err != nil {
			return nil, err
		}

		c.Fields = fields
		t.Commands = append(t.Commands, c)
	}

	return t, nil
}

func pathParams(path string) []string {
	var out []string

	for {
		i := strings.IndexByte(path, '{')
		if i < 0 {
			return out
		}

		j := strings.IndexByte(path[i:], '}')
		if j < 0 {
			return out
		}

		out = append(out, path[i+1:i+j])
		path = path[i+j+1:]
	}
}

func fieldsOf(m operations.Meta, params []string) ([]Field, error) {
	t := m.Input
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	var (
		out   []Field
		flags = map[string]string{}
	)

	claim := func(flag, field string) error {
		if slices.Contains(reservedFlags, flag) {
			return fmt.Errorf("cli: %s: field %s needs the flag --%s, which every command has", m.Name, field, flag)
		}

		if prev, dup := flags[flag]; dup {
			return fmt.Errorf("cli: %s: fields %s and %s share the flag --%s", m.Name, prev, field, flag)
		}

		flags[flag] = field

		return nil
	}

	for f := range t.Fields() {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" || !f.IsExported() {
			continue
		}

		if name == "" {
			name = f.Name
		}

		kind, err := kindOf(f.Type)
		if err != nil {
			return nil, fmt.Errorf("cli: %s: field %s: %w", m.Name, name, err)
		}

		validate := f.Tag.Get("validate")
		fld := Field{
			Name: name, Flag: strings.ReplaceAll(name, "_", "-"), Kind: kind, Positional: slices.Index(params, name),
			Required: strings.HasPrefix(validate, "required") || strings.Contains(validate, ",required"),
		}

		fld.Cents = kind == KindInt && strings.HasSuffix(name, "_cents")

		if fld.Positional < 0 {
			if err := claim(fld.Flag, name); err != nil {
				return nil, err
			}

			if fld.Cents {
				if err := claim(strings.TrimSuffix(fld.Flag, "-cents"), name); err != nil {
					return nil, err
				}
			}
		}

		out = append(out, fld)
	}

	for _, p := range params {
		if !slices.ContainsFunc(out, func(f Field) bool { return f.Name == p }) {
			return nil, fmt.Errorf("cli: %s: the path names {%s}, which the input does not have", m.Name, p)
		}
	}

	return out, nil
}

func kindOf(t reflect.Type) (FieldKind, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return KindString, nil
	case reflect.Bool:
		return KindBool, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return KindInt, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return KindStrings, nil
		}
	}

	return 0, fmt.Errorf("the type %s has no flag", t)
}
