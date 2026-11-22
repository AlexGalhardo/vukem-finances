package cli

import (
	"context"
	"io"
)

// Request is one call of an operation, as the same for both modes: Body is the
// whole input, path parameters included.
type Request struct {
	Cmd     Command
	Body    map[string]any
	IfMatch string
	Key     string
}

// Response is the answer of an operation. Value is the JSON of the output
// decoded with numbers kept as text, so no amount passes through a float.
type Response struct {
	Value any
	// Stream is set for a file (an export): it writes the bytes to w.
	Stream   func(w io.Writer) error
	Filename string
}

// Backend runs requests: in this process (local) or against a server (remote).
type Backend interface {
	Call(ctx context.Context, r Request) (Response, error)
	Close() error
}
