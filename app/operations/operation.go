package operations

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem/cache"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/z"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Kind says what an operation does to the data.
type Kind string

// The kinds. A destructive operation removes or archives data.
const (
	KindRead        Kind = "read"
	KindWrite       Kind = "write"
	KindDestructive Kind = "destructive"
)

// Operation is the declaration of one operation: what it is called, what it
// needs and what it does. In and Out are the wire types: a `json` tag names a
// field and a `validate` tag is a rule Invoke enforces.
type Operation[In, Out any] struct {
	// Name is `resource.verb`, the operation id of the REST document too.
	Name    string
	Summary string
	// Scope is the scope a credential needs; every operation has one.
	Scope string
	// Ability overrides the role ability the scope implies, for an operation
	// that asks more or less of the role than its scope (the JSON export of a
	// workspace needs the data ability under the read scope of the exports).
	Ability string
	Kind    Kind
	// Idempotent makes the operation honor an idempotency key.
	Idempotent bool
	// Editions lists the editions that have the operation; empty means all.
	Editions []config.Edition
	// OptOut names the surfaces that do not offer the operation, each with the
	// reason. An operation is offered everywhere else.
	OptOut map[Surface]string
	// Run does the work for the actor, in the workspace of the actor.
	Run func(ctx context.Context, actor Actor, in In) (Out, error)
}

// Meta describes an operation without its types parameters: what the
// inventory, the parity test and the adapters read.
type Meta struct {
	Name       string
	Summary    string
	Scope      string
	Ability    string
	Kind       Kind
	Idempotent bool
	Editions   []config.Edition
	OptOut     map[Surface]string
	Input      reflect.Type
	Output     reflect.Type
}

// Offered reports whether the operation is offered on surface.
func (m Meta) Offered(s Surface) bool {
	_, out := m.OptOut[s]

	return !out
}

// Config is what the pipeline needs to run.
type Config struct {
	// Edition is the edition the app runs in.
	Edition config.Edition
	// Store keeps the idempotency keys; without one a key is refused.
	Store cache.Store
	// Log receives the audit entries; nil discards them.
	Log *slog.Logger
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Frozen reports whether the workspace refuses writes (it waits to be
	// purged, epic 34). NewCatalog fills it from the workspace service; nil
	// means no workspace is frozen. Invoke refuses every operation that is not a
	// read, on every surface, before it validates or runs anything.
	Frozen func(ctx context.Context, workspaceID int64) (bool, error)
}

// Registry holds the operations. It is filled at boot and only read after.
type Registry struct {
	cfg      Config
	metas    []Meta
	dynamics map[string]Dynamic
}

// NewRegistry returns an empty registry.
func NewRegistry(cfg Config) *Registry {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	if cfg.Log == nil {
		cfg.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Registry{cfg: cfg}
}

// Config returns the configuration of the pipeline.
func (r *Registry) Config() Config { return r.cfg }

// Operations lists the metadata of every operation, in the order declared.
func (r *Registry) Operations() []Meta { return slices.Clone(r.metas) }

// Named finds the metadata of an operation.
func (r *Registry) Named(name string) (Meta, bool) {
	for _, m := range r.metas {
		if m.Name == name {
			return m, true
		}
	}

	return Meta{}, false
}

// Op is a declared operation, the handle an adapter invokes.
type Op[In, Out any] struct {
	reg   *Registry
	run   func(context.Context, Actor, In) (Out, error)
	meta  Meta
	check func(In) error
}

// Meta returns the metadata of the operation.
func (o *Op[In, Out]) Meta() Meta { return o.meta }

// Define adds an operation to r and returns its handle. A malformed
// declaration panics, so the mistake surfaces at boot.
func Define[In, Out any](r *Registry, d Operation[In, Out]) *Op[In, Out] {
	switch {
	case d.Name == "" || d.Run == nil:
		panic("operations: an operation needs a name and a Run")
	case !ValidScope(d.Scope):
		panic(fmt.Sprintf("operations: %s has no known scope (%q)", d.Name, d.Scope))
	case d.Kind != KindRead && d.Kind != KindWrite && d.Kind != KindDestructive:
		panic(fmt.Sprintf("operations: %s has no kind", d.Name))
	case d.Idempotent && d.Kind == KindRead:
		panic(fmt.Sprintf("operations: %s reads and cannot be idempotent", d.Name))
	}

	if _, dup := r.Named(d.Name); dup {
		panic("operations: " + d.Name + " is declared twice")
	}

	for s, why := range d.OptOut {
		if !slices.Contains(Surfaces(), s) || why == "" {
			panic(fmt.Sprintf("operations: %s opts out of %q without a surface or a reason", d.Name, s))
		}
	}

	ability := d.Ability
	if ability == "" {
		ability = AbilityOf(d.Scope)
	}

	in := reflect.TypeFor[In]()

	m := Meta{
		Name: d.Name, Summary: d.Summary, Scope: d.Scope, Ability: ability, Kind: d.Kind, Idempotent: d.Idempotent,
		Editions: slices.Clone(d.Editions), OptOut: d.OptOut, Input: in, Output: reflect.TypeFor[Out](),
	}

	op := &Op[In, Out]{reg: r, run: d.Run, meta: m}

	// z.For panics at boot on a bad tag; an input with no field has no rules.
	if in.Kind() == reflect.Struct && in.NumField() > 0 {
		schema := z.For[In]()
		op.check = schema.Check
	}

	r.metas = append(r.metas, m)

	if r.dynamics == nil {
		r.dynamics = map[string]Dynamic{}
	}

	r.dynamics[d.Name] = op

	return op
}

// Authorize is the gate every call passes first: the actor is real, the
// operation exists in the edition, the credential holds the scope and the role
// holds the ability. The scope is a ceiling, never a grant. An adapter may call
// it early (the REST adapter does, to refuse before it reads a body); Invoke
// calls it again and cannot be skipped.
func (r *Registry) Authorize(m Meta, a Actor) error {
	switch {
	case !a.valid():
		return fail(403, CodeForbidden, "A valid credential is required.")
	case len(m.Editions) > 0 && !slices.Contains(m.Editions, r.cfg.Edition):
		return fail(404, CodeNotFound, "Not found.")
	case !a.HasScope(m.Scope):
		return fail(403, CodeMissingScope, "The token lacks the scope "+m.Scope+".")
	case !workspaces.RoleCan(a.role, m.Ability):
		return fail(403, CodeForbidden, "Your role in this workspace cannot do that.")
	}

	return nil
}

// Result is what an invocation returns.
type Result[Out any] struct {
	Value Out
	// Existing is set when the operation answered with a row that was already
	// there (a create that repeated a client uid): an adapter says "ok", not
	// "created".
	Existing bool
	// Replayed is set when an idempotency key already ran: Value is the stored
	// answer and Run did not run again.
	Replayed bool
}

// CallOption tunes one invocation.
type CallOption func(*callOptions)

type callOptions struct {
	key     string
	ifMatch string
}

// WithIdempotencyKey names the call: an operation declared Idempotent runs once
// per key and replays its answer after.
func WithIdempotencyKey(key string) CallOption { return func(o *callOptions) { o.key = key } }

// WithIfMatch states the version of the row the caller saw, as text (the
// If-Match header, a CLI flag). A write that finds another version is a
// version_conflict; text that is not a whole number is a validation error.
func WithIfMatch(version string) CallOption { return func(o *callOptions) { o.ifMatch = version } }

// call is what Run can read and set about the invocation in progress.
type call struct {
	existing bool
	ifMatch  string
}

type callKey struct{}

// MarkExisting is how Run says that it answered with a row that was already
// there.
func MarkExisting(ctx context.Context) {
	if c, ok := ctx.Value(callKey{}).(*call); ok {
		c.existing = true
	}
}

// Invoke runs op for actor with in. It is the only way to run an operation:
// authorize, validate, apply the idempotency key, run, audit.
func Invoke[In, Out any](ctx context.Context, op *Op[In, Out], actor Actor, in In, opts ...CallOption) (Result[Out], error) {
	res, err := invoke(ctx, op, actor, in, opts)

	if op.meta.Kind != KindRead {
		op.reg.audit(ctx, op.meta, actor, res.Existing, res.Replayed, err)
	}

	return res, err
}

func invoke[In, Out any](ctx context.Context, op *Op[In, Out], actor Actor, in In, opts []CallOption) (Result[Out], error) {
	var zero Result[Out]

	var o callOptions
	for _, opt := range opts {
		opt(&o)
	}

	if err := op.reg.Authorize(op.meta, actor); err != nil {
		return zero, err
	}

	if op.meta.Kind != KindRead && op.reg.cfg.Frozen != nil {
		frozen, err := op.reg.cfg.Frozen(ctx, actor.workspace)
		if err != nil {
			return zero, asError(err)
		}

		if frozen {
			return zero, fail(http.StatusLocked, CodePendingDeletion, workspaces.ErrPendingDeletion.Text())
		}
	}

	if op.check != nil {
		if err := op.check(in); err != nil {
			return zero, invalid(err)
		}
	}

	st := &call{ifMatch: o.ifMatch}
	ctx = context.WithValue(ctx, callKey{}, st)
	ctx = workspaces.WithTenant(ctx, actor.workspace)
	ctx = workspaces.WithRole(ctx, actor.role)

	if !op.meta.Idempotent || o.key == "" {
		out, err := op.run(ctx, actor, in)
		if err != nil {
			return zero, asError(err)
		}

		return Result[Out]{Value: out, Existing: st.existing}, nil
	}

	return invokeKeyed(ctx, op, actor, in, o.key, st)
}

// invalid turns the issues of a failed check into a validation error.
func invalid(err error) *Error {
	is, ok := z.AsIssues(err)
	if !ok {
		return fail(422, CodeValidation, "The input is not valid.")
	}

	verr := &errs.ValidationError{Fields: is.Fields()}

	return &Error{Code: CodeValidation, Message: "The input is not valid.", Err: verr}
}
