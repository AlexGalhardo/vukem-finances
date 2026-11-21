package account

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/ratelimit"
)

func TestCodeBudgetHoldsUnderConcurrency(t *testing.T) {
	t.Parallel()

	const limit = 10

	b := &codeBudget{limiter: ratelimit.NewMemory(limit, time.Hour)}

	var (
		checked, justLocked atomic.Int64
		wg                  sync.WaitGroup
	)

	for range 200 {
		wg.Go(func() {
			got, err := b.try(t.Context(), 7, "2fa-account:7", func() (bool, error) {
				checked.Add(1)
				time.Sleep(time.Millisecond)

				return false, nil
			})
			if err != nil {
				t.Error(err)
			}

			if got.JustLocked {
				justLocked.Add(1)
			}
		})
	}

	wg.Wait()

	if n := checked.Load(); n != limit {
		t.Errorf("%d codes were checked, want exactly %d", n, limit)
	}

	if n := justLocked.Load(); n != 1 {
		t.Errorf("the owner would get %d lock mails, want 1", n)
	}
}

func TestCodeBudgetSpendsOnlyWrongCodes(t *testing.T) {
	t.Parallel()

	b := &codeBudget{limiter: ratelimit.NewMemory(2, time.Hour)}

	for range 5 {
		got, err := b.try(t.Context(), 1, "k", func() (bool, error) { return true, nil })
		if err != nil || !got.OK || got.Locked {
			t.Fatalf("a right code = %+v, %v", got, err)
		}
	}
}
