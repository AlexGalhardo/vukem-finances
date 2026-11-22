package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/ratelimit"
)

// allowOnly is a limiter that cannot look without spending: not a Peeker.
type allowOnly struct{}

func (allowOnly) Allow(context.Context, string) (ratelimit.Result, error) {
	return ratelimit.Result{Allowed: true}, nil
}

func TestPairFailureGate(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Not configured: nothing to hold pairing to.
	if err := pairFailureGate(ctx, nil, http.Header{}); err != nil {
		t.Fatalf("a nil limiter = %v, want nil", err)
	}

	// A limiter that cannot peek must not turn the cap off silently: it fails closed.
	err := pairFailureGate(ctx, allowOnly{}, http.Header{})
	if got := errs.Status(err); got != http.StatusServiceUnavailable {
		t.Fatalf("a limiter that is not a Peeker = %d (%v), want 503", got, err)
	}

	// A Peeker with credit left lets the attempt through; spent, it answers 429 with Retry-After.
	l := ratelimit.NewMemory(2, time.Hour)
	if err := pairFailureGate(ctx, l, http.Header{}); err != nil {
		t.Fatalf("a limiter with credit = %v", err)
	}

	for range 2 {
		if _, err := l.Allow(ctx, "pair-failed"); err != nil {
			t.Fatal(err)
		}
	}

	h := http.Header{}
	if got := errs.Status(pairFailureGate(ctx, l, h)); got != http.StatusTooManyRequests {
		t.Fatalf("a spent limiter = %d, want 429", got)
	}

	if h.Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}
