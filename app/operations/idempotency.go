package operations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const (
	// idempotencyTTL keeps the answer of a finished call.
	idempotencyTTL = 24 * time.Hour
	// idempotencyLock bounds a call in flight: a crash frees the key after it.
	idempotencyLock = time.Minute
)

type idemRecord struct {
	Fingerprint string          `json:"f"`
	Done        bool            `json:"d"`
	Existing    bool            `json:"e,omitempty"`
	Value       json.RawMessage `json:"v,omitempty"`
}

func digest(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = io.WriteString(h, p)
		_, _ = io.WriteString(h, "\x00")
	}

	return hex.EncodeToString(h.Sum(nil))
}

func validKey(k string) bool {
	if k == "" || len(k) > 255 {
		return false
	}

	for i := range len(k) {
		if k[i] < 0x21 || k[i] > 0x7e {
			return false
		}
	}

	return true
}

// invokeKeyed runs op once per key of the actor. The first call claims the
// key, runs and stores its answer; a repeat while it runs is a conflict, a
// repeat with other input is refused, and a repeat after it finished gets the
// stored answer. A failed call frees the key (it changed nothing). When the call
// ran but its answer cannot be stored, the claim stays until idempotencyLock
// ends, so a retry in that time is a conflict and not a second effect.
func invokeKeyed[In, Out any](ctx context.Context, op *Op[In, Out], actor Actor, in In, key string, st *call) (Result[Out], error) {
	var zero Result[Out]

	if !validKey(key) {
		return zero, fail(http.StatusBadRequest, CodeValidation, "A valid Idempotency-Key header (1-255 printable characters) is required.")
	}

	store := op.reg.cfg.Store
	if store == nil {
		return zero, fail(http.StatusServiceUnavailable, CodeInternal, "Idempotency store unavailable.")
	}

	body, err := json.Marshal(in)
	if err != nil {
		return zero, fail(http.StatusUnprocessableEntity, CodeValidation, "The input is not valid.")
	}

	storeKey := "idempotency:" + digest(actor.idempotencyScope(), key)
	fingerprint := digest(op.meta.Name, string(body))

	pending, _ := json.Marshal(idemRecord{Fingerprint: fingerprint})

	added, err := store.Add(ctx, storeKey, pending, idempotencyLock)
	if err != nil {
		return zero, fail(http.StatusServiceUnavailable, CodeInternal, "Idempotency store unavailable.")
	}

	if !added {
		return replay[Out](ctx, op, storeKey, fingerprint)
	}

	// The client may be gone when the call ends: the outcome is stored anyway,
	// or its retry would run the operation a second time.
	settle := context.WithoutCancel(ctx)

	ran := false

	defer func() {
		if !ran {
			_ = store.Delete(settle, storeKey)
		}
	}()

	out, err := op.run(ctx, actor, in)
	if err != nil {
		return zero, asError(err)
	}

	ran = true

	value, err := json.Marshal(out)
	if err != nil {
		return zero, asError(err)
	}

	rec, _ := json.Marshal(idemRecord{Fingerprint: fingerprint, Done: true, Existing: st.existing, Value: value})
	if err := store.Set(settle, storeKey, rec, idempotencyTTL); err != nil {
		return zero, asError(err)
	}

	return Result[Out]{Value: out, Existing: st.existing}, nil
}

func replay[Out any, In any](ctx context.Context, op *Op[In, Out], storeKey, fingerprint string) (Result[Out], error) {
	var zero Result[Out]

	raw, ok, err := op.reg.cfg.Store.Get(ctx, storeKey)
	if err != nil {
		return zero, fail(http.StatusServiceUnavailable, CodeInternal, "Idempotency store unavailable.")
	}

	var rec idemRecord
	if !ok || json.Unmarshal(raw, &rec) != nil {
		return zero, fail(http.StatusConflict, CodeConflict, "A request with this Idempotency-Key is being processed.")
	}

	switch {
	case rec.Fingerprint != fingerprint:
		return zero, fail(http.StatusUnprocessableEntity, CodeValidation, "This Idempotency-Key was used for a different request.")
	case !rec.Done:
		return zero, fail(http.StatusConflict, CodeConflict, "A request with this Idempotency-Key is being processed.")
	}

	var out Out
	if err := json.Unmarshal(rec.Value, &out); err != nil {
		return zero, asError(err)
	}

	return Result[Out]{Value: out, Existing: rec.Existing, Replayed: true}, nil
}
