package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/AlexGalhardo/vukem/mcp"
)

// Dynamic is an operation called by name with its input as JSON text, the way
// an adapter that does not know the types (the CLI, later MCP) reaches it. It
// goes through Invoke, so the gate, the validation, the idempotency key and the
// audit are the same ones the typed handle has.
type Dynamic interface {
	Meta() Meta
	// CallJSON decodes body (the whole input as a JSON object; empty means the
	// zero input) and invokes the operation. An unknown field is refused.
	CallJSON(ctx context.Context, actor Actor, body []byte, opts ...CallOption) (Result[any], error)
	// AddTool registers the operation as an MCP tool.
	AddTool(s *mcp.Server, t mcp.Tool, gate ToolGate, done ToolDone)
}

// CallJSON implements Dynamic.
func (o *Op[In, Out]) CallJSON(ctx context.Context, actor Actor, body []byte, opts ...CallOption) (Result[any], error) {
	var in In

	if len(bytes.TrimSpace(body)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.DisallowUnknownFields()

		err := dec.Decode(&in)
		if err == nil {
			if _, more := dec.Token(); !errors.Is(more, io.EOF) {
				err = errors.New("trailing data after the input")
			}
		}

		if err != nil {
			return Result[any]{}, fail(422, CodeValidation, "The input is not valid: "+err.Error()+".")
		}
	}

	res, err := Invoke(ctx, o, actor, in, opts...)

	return Result[any]{Value: res.Value, Existing: res.Existing, Replayed: res.Replayed}, err
}

// Dynamic finds the dynamic handle of an operation.
func (r *Registry) Dynamic(name string) (Dynamic, bool) {
	d, ok := r.dynamics[name]

	return d, ok
}
