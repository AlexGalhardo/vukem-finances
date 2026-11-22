package operations_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/cache"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/config"
)

type rowIn struct {
	Name string `json:"name" validate:"required,max=20"`
}

type rowOut struct {
	Row  int64  `json:"row"`
	Name string `json:"name"`
}

// rows counts the rows an operation made: the thing a replay must not repeat.
type rows struct {
	n    atomic.Int64
	fail atomic.Bool
	hold chan struct{}
}

func (r *rows) op(reg *operations.Registry) *operations.Op[rowIn, rowOut] {
	return operations.Define(reg, operations.Operation[rowIn, rowOut]{
		Name: "rows.create", Scope: operations.ScopeAccountsWrite, Kind: operations.KindWrite, Idempotent: true,
		Run: func(_ context.Context, _ operations.Actor, in rowIn) (rowOut, error) {
			if r.hold != nil {
				<-r.hold
			}

			if r.fail.Load() {
				return rowOut{}, errors.New("boom")
			}

			return rowOut{Row: r.n.Add(1), Name: in.Name}, nil
		},
	})
}

// setFails is a store that takes the claim but cannot keep the answer.
type setFails struct{ cache.Store }

func (setFails) Set(context.Context, string, []byte, time.Duration) error {
	return errors.New("store down")
}

// The operation already ran: a retry must not run it again.
func TestIdempotencyKeepsTheClaimWhenTheAnswerCannotBeStored(t *testing.T) {
	t.Parallel()

	var r rows

	reg := operations.NewRegistry(operations.Config{Edition: config.EditionSaaS, Store: setFails{cache.NewMemory(1000)}})
	op := r.op(reg)
	owner := actor(models.RoleOwner, operations.ScopeAccountsWrite)
	key := operations.WithIdempotencyKey("key-store-down")

	if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key); err == nil {
		t.Fatal("a call whose answer was not stored reported success")
	}

	if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key); err == nil {
		t.Fatal("the retry was not refused")
	}

	if got := r.n.Load(); got != 1 {
		t.Fatalf("the operation ran %d times, want 1", got)
	}
}

func TestIdempotency(t *testing.T) {
	t.Parallel()

	newReg := func() *operations.Registry {
		return operations.NewRegistry(operations.Config{Edition: config.EditionSaaS, Store: cache.NewMemory(1000)})
	}
	owner := actor(models.RoleOwner, operations.ScopeAccountsWrite)
	other := operations.NewActor(operations.SurfaceREST, user(7), 3, models.RoleOwner, []string{operations.ScopeAccountsWrite}, "another-token")

	t.Run("a replayed key creates one row", func(t *testing.T) {
		t.Parallel()

		var r rows

		op := r.op(newReg())
		key := operations.WithIdempotencyKey("key-1")

		first, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key)
		if err != nil || first.Replayed || first.Value.Row != 1 {
			t.Fatalf("first = %+v, %v", first, err)
		}

		again, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key)
		if err != nil || !again.Replayed || again.Value != first.Value {
			t.Fatalf("again = %+v, %v, want the stored answer %+v", again, err, first.Value)
		}

		if n := r.n.Load(); n != 1 {
			t.Fatalf("rows = %d, want 1", n)
		}
	})

	t.Run("the key is tied to its input and its caller", func(t *testing.T) {
		t.Parallel()

		var r rows

		op := r.op(newReg())
		key := operations.WithIdempotencyKey("key-1")

		if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key); err != nil {
			t.Fatal(err)
		}

		if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "b"}, key); !operations.IsCode(err, operations.CodeValidation) {
			t.Errorf("another input under the key: %v, want validation_failed", err)
		}

		if res, err := operations.Invoke(t.Context(), op, other, rowIn{Name: "a"}, key); err != nil || res.Replayed {
			t.Errorf("another caller shared the key: %+v, %v", res, err)
		}

		if n := r.n.Load(); n != 2 {
			t.Errorf("rows = %d, want 2", n)
		}
	})

	t.Run("no key runs every time and a bad key is refused", func(t *testing.T) {
		t.Parallel()

		var r rows

		op := r.op(newReg())

		for range 2 {
			if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}); err != nil {
				t.Fatal(err)
			}
		}

		if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, operations.WithIdempotencyKey("has space")); !operations.IsCode(err, operations.CodeValidation) {
			t.Errorf("bad key: %v", err)
		}

		if n := r.n.Load(); n != 2 {
			t.Errorf("rows = %d, want 2", n)
		}
	})

	t.Run("a failed call frees the key", func(t *testing.T) {
		t.Parallel()

		var r rows

		op := r.op(newReg())
		key := operations.WithIdempotencyKey("key-1")

		r.fail.Store(true)

		if _, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key); err == nil {
			t.Fatal("the failing call succeeded")
		}

		r.fail.Store(false)

		if res, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, key); err != nil || res.Replayed || r.n.Load() != 1 {
			t.Errorf("retry = %+v, %v, rows %d", res, err, r.n.Load())
		}
	})

	t.Run("ten parallel calls with one key make one row", func(t *testing.T) {
		t.Parallel()

		r := rows{hold: make(chan struct{})}
		op := r.op(newReg())

		var (
			wg       sync.WaitGroup
			conflict atomic.Int64
			ok       atomic.Int64
		)

		for range 10 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				_, err := operations.Invoke(t.Context(), op, owner, rowIn{Name: "a"}, operations.WithIdempotencyKey("key-1"))

				switch {
				case err == nil:
					ok.Add(1)
				case operations.IsCode(err, operations.CodeConflict):
					conflict.Add(1)
				default:
					t.Errorf("unexpected error: %v", err)
				}
			}()
		}

		time.Sleep(50 * time.Millisecond)
		close(r.hold)
		wg.Wait()

		if r.n.Load() != 1 || ok.Load() < 1 || ok.Load()+conflict.Load() != 10 {
			t.Fatalf("rows %d, ok %d, conflict %d, want 1 row and every call an answer or a conflict", r.n.Load(), ok.Load(), conflict.Load())
		}
	})
}
