package account

import (
	"context"
	"sync"

	"github.com/AlexGalhardo/vukem/ratelimit"
)

// codeBudget is the per-account budget of wrong two-factor codes. Peek and
// Allow are separate calls, so concurrent requests of one account take turns:
// otherwise a burst would all pass Peek before the first wrong code is spent,
// and try far more codes than the limit.
type codeBudget struct {
	limiter ratelimit.Peeker
	turns   [64]sync.Mutex
}

// codeAttempt is the outcome of one try. Wrong is true when a code was checked
// and failed; JustLocked when that failure used up the budget.
type codeAttempt struct {
	OK, Wrong, Locked, JustLocked bool
}

// try runs check unless the budget of key is already spent, and spends one
// credit when check says the code is wrong.
func (b *codeBudget) try(ctx context.Context, id int64, key string, check func() (bool, error)) (codeAttempt, error) {
	turn := &b.turns[uint64(id)%uint64(len(b.turns))]

	turn.Lock()
	defer turn.Unlock()

	left, err := b.limiter.Peek(ctx, key)
	if err != nil {
		return codeAttempt{}, err
	}

	if !left.Allowed {
		return codeAttempt{Locked: true}, nil
	}

	ok, err := check()
	if err != nil {
		return codeAttempt{}, err
	}

	if ok {
		return codeAttempt{OK: true}, nil
	}

	spent, err := b.limiter.Allow(ctx, key)
	if err != nil {
		return codeAttempt{}, err
	}

	locked := !spent.Allowed || spent.Remaining == 0

	return codeAttempt{Wrong: true, Locked: locked, JustLocked: locked}, nil
}
