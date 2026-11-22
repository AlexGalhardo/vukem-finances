package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Local runs operations in this process, as the owner of a personal
// installation. It reaches them only through operations.Invoke (here, the
// dynamic handle of the registry), so the edition, scope, role, validation,
// idempotency key and audit are the pipeline's.
type Local struct {
	cat   *operations.Catalog
	actor operations.Actor
	close func() error
}

// OpenLocal opens the application of this installation: the booted app and
// what closes it.
type OpenLocal func(ctx context.Context) (*vukem.App, func() error, error)

// LocalFrom returns the function the CLI calls to get its local backend.
func LocalFrom(open OpenLocal) func(ctx context.Context) (Backend, error) {
	return func(ctx context.Context) (Backend, error) {
		app, closeApp, err := open(ctx)
		if err != nil {
			return nil, err
		}

		l, err := NewLocal(ctx, app)
		if err != nil {
			_ = closeApp()

			return nil, err
		}

		l.close = closeApp

		return l, nil
	}
}

// NewLocal builds the local backend over a booted app. It fails when the
// edition is not personal (there is no local owner then) or the installation
// has no owner yet.
func NewLocal(ctx context.Context, app *vukem.App) (*Local, error) {
	ws, err := workspaces.New(app)
	if err != nil {
		return nil, fmt.Errorf("opening the workspaces: %w", err)
	}

	var scopes []string
	for _, s := range operations.Scopes() {
		scopes = append(scopes, s.Name)
	}

	actor, err := operations.LocalOwner(ctx, ws, operations.SurfaceCLI, scopes)
	if err != nil {
		return nil, failureOf(err)
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return nil, fmt.Errorf("opening the finances: %w", err)
	}

	store, err := app.Cache()
	if err != nil {
		return nil, fmt.Errorf("opening the cache: %w", err)
	}

	cat := operations.NewCatalog(
		operations.Config{Edition: ws.Edition, Store: store.Store(), Log: app.Log, Now: app.Now},
		operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)},
	)

	return &Local{cat: cat, actor: actor}, nil
}

// NewLocalWith builds a local backend from parts, for tests.
func NewLocalWith(cat *operations.Catalog, actor operations.Actor) *Local {
	return &Local{cat: cat, actor: actor}
}

// Close releases the application.
func (l *Local) Close() error {
	if l.close == nil {
		return nil
	}

	return l.close()
}

// Call implements Backend.
func (l *Local) Call(ctx context.Context, r Request) (Response, error) {
	dyn, ok := l.cat.Dynamic(r.Cmd.Op.Name)
	if !ok {
		return Response{}, fmt.Errorf("cli: %s is not in the registry", r.Cmd.Op.Name)
	}

	body, err := json.Marshal(r.Body)
	if err != nil {
		return Response{}, err
	}

	var opts []operations.CallOption
	if r.IfMatch != "" {
		opts = append(opts, operations.WithIfMatch(r.IfMatch))
	}

	if r.Key != "" {
		opts = append(opts, operations.WithIdempotencyKey(r.Key))
	}

	res, err := dyn.CallJSON(ctx, l.actor, body, opts...)
	if err != nil {
		return Response{}, failureOf(err)
	}

	if file, ok := res.Value.(operations.Download); ok {
		return Response{Stream: file.Write, Filename: file.Filename}, nil
	}

	raw, err := json.Marshal(res.Value)
	if err != nil {
		return Response{}, err
	}

	v, err := decodeValue(raw)
	if err != nil {
		return Response{}, err
	}

	return Response{Value: v}, nil
}

// decodeValue reads JSON keeping numbers as json.Number.
func decodeValue(raw []byte) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, errors.New("the answer is not JSON")
	}

	return v, nil
}
