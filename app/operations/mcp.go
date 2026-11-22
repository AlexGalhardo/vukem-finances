package operations

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/mcp"
)

// ToolGate is what an MCP adapter runs before a tool call: it names the actor
// of the call and applies the gates the adapter owns (toolsets, the destructive
// switch, the confirmation). Whatever it returns, Invoke checks the edition,
// scope, role and input again.
type ToolGate func(ctx context.Context, m Meta, in any) (Actor, error)

// AddTool registers the operation as an MCP tool on s. The input and output
// schemas come from the operation types, so the tool and the REST endpoint
// cannot disagree; the call goes through Invoke like every other surface.
func (o *Op[In, Out]) AddTool(s *mcp.Server, t mcp.Tool, gate ToolGate) {
	mcp.AddTool(s, t, func(ctx context.Context, in In) (Out, error) {
		var zero Out

		actor, err := gate(ctx, o.meta, in)
		if err != nil {
			return zero, err
		}

		// Sent only when the client asked for progress with a token.
		mcp.Progress(ctx, 0, 1, o.meta.Summary)

		res, err := Invoke(ctx, o, actor, in)
		if err != nil {
			return zero, ToolError(err)
		}

		mcp.Progress(ctx, 1, 1, "done")

		return res.Value, nil
	})
}

// ToolError turns the error of an operation into what an agent may read: a
// tool error with the stable code and the safe message, or, for an internal
// failure, the error itself, which the MCP server logs and answers generically.
func ToolError(err error) error {
	e, ok := errors.AsType[*Error](err)
	if !ok || e.Code == CodeInternal {
		return err
	}

	text := e.Code + ": " + e.Message

	if v, ok := errors.AsType[*errs.ValidationError](e.Err); ok {
		fields := make([]string, 0, len(v.Fields))

		for f, msgs := range v.Fields {
			fields = append(fields, fmt.Sprintf("%s %s", f, strings.Join(msgs, ", ")))
		}

		slices.Sort(fields)

		if len(fields) > 0 {
			text += " (" + strings.Join(fields, "; ") + ")"
		}
	}

	return mcp.ToolErrorf("%s", text)
}
