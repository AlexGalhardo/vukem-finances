package operations

import (
	"context"
	"log/slog"
)

// audit writes the entry of a write or destructive call: who did what, from
// which surface, and how it ended. It holds no input and no output: never an
// amount, a description, a name or a uid. The entry is a structured log line
// (message "operation"), one per call that got as far as the pipeline,
// refusals included.
func (r *Registry) audit(ctx context.Context, m Meta, a Actor, existing, replayed bool, err error) {
	attrs := []any{
		"operation", m.Name, "kind", string(m.Kind), "surface", string(a.surface),
		"workspace", a.workspace, "user", a.user.ID,
	}

	if err != nil {
		code := CodeInternal
		if e := asError(err); e != nil {
			code = e.Code
		}

		r.cfg.Log.Log(ctx, slog.LevelWarn, "operation", append(attrs, "outcome", "refused", "code", code)...)

		return
	}

	r.cfg.Log.Log(ctx, slog.LevelInfo, "operation", append(attrs, "outcome", "ok", "existing", existing, "replayed", replayed)...)
}
