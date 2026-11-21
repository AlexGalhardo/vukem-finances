package ai_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/fake"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// syncBuffer is a log sink that is safe for the concurrent tests.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.b.String()
}

type rig struct {
	site
	client   *ai.Client
	settings *ai.SettingsStore
	provider *fake.Provider
	factory  int
	log      *syncBuffer
	clock    *time.Time
}

func newRig(t *testing.T) *rig {
	t.Helper()

	s := newSite(t, newApp(t, nil), "ana@example.com")
	db := mustDB(t, s)
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	r := &rig{site: s, provider: &fake.Provider{}, log: &syncBuffer{}, clock: &at}

	r.settings = ai.NewSettingsStore(db, func() time.Time { return *r.clock })

	factory := func(ai.ProviderConfig) (ai.Provider, error) {
		r.factory++
		return r.provider, nil
	}

	r.client = ai.NewClient(db, newVault(t, s), r.settings, factory, slog.New(slog.NewTextHandler(r.log, nil)),
		func() time.Time { return *r.clock })

	return r
}

func (r *rig) enable(t *testing.T, limit int) {
	t.Helper()

	_, err := r.settings.Update(r.ctx, r.user.ID, ai.UpdateInput{
		Enabled: true, Consent: true, MonthlyCallLimit: limit,
		Features: map[string]bool{ai.FeatureCategorize: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (r *rig) call() error {
	_, err := r.client.Call(r.ctx, r.ws.ID, ai.Request{
		Feature: ai.FeatureCategorize, Messages: []ai.Message{{Role: "user", Content: "Mercado"}}, MaxTokens: 50,
	})

	return err
}

func TestDisabledMakesNoCall(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	saveTestKey(t, r.client.Vault, r.site)

	// Never enabled, then enabled and disabled again: no call either way.
	if err := r.call(); !errors.Is(err, ai.ErrDisabled) {
		t.Fatalf("fresh workspace: %v", err)
	}

	r.enable(t, 100)

	if err := r.call(); err != nil {
		t.Fatal(err)
	}

	if _, err := r.settings.Update(r.ctx, r.user.ID, ai.UpdateInput{MonthlyCallLimit: 100}); err != nil {
		t.Fatal(err)
	}

	before := r.provider.Calls()

	if err := r.call(); !errors.Is(err, ai.ErrDisabled) {
		t.Fatalf("after disabling: %v", err)
	}

	if err := r.client.Check(r.ctx, r.ws.ID); err != nil {
		t.Fatalf("Check is the owner's explicit test and is allowed while off: %v", err)
	}

	if r.provider.Calls() != before {
		t.Errorf("a disabled workspace made %d outbound calls", r.provider.Calls()-before)
	}

	// Nothing was counted for the refused call.
	if u, _ := r.client.Usage(r.ctx); u.Calls != 1 {
		t.Errorf("usage = %d calls, want 1", u.Calls)
	}
}

func TestCallPipeline(t *testing.T) {
	t.Parallel()

	t.Run("checks run in order", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)

		if err := r.call(); !errors.Is(err, ai.ErrDisabled) {
			t.Fatalf("off: %v", err)
		}

		r.enable(t, 100)

		_, err := r.client.Call(r.ctx, r.ws.ID, ai.Request{Feature: ai.FeatureAssistant})
		if !errors.Is(err, ai.ErrFeatureOff) {
			t.Fatalf("feature off: %v", err)
		}

		if err := r.call(); !errors.Is(err, ai.ErrNoCredential) {
			t.Fatalf("no key: %v", err)
		}

		saveTestKey(t, r.client.Vault, r.site)

		if err := r.call(); err != nil {
			t.Fatalf("all checks pass: %v", err)
		}

		if r.provider.Calls() != 1 || r.factory != 1 {
			t.Errorf("calls = %d, factory = %d, want 1 each", r.provider.Calls(), r.factory)
		}
	})

	t.Run("consent is needed to turn on", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)

		_, err := r.settings.Update(r.ctx, r.user.ID, ai.UpdateInput{Enabled: true, MonthlyCallLimit: 10})
		if err == nil {
			t.Fatal("AI was enabled without consent")
		}

		if s, _ := r.settings.Get(r.ctx); s.Enabled {
			t.Error("the settings say enabled")
		}
	})

	t.Run("usage is recorded with tokens", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 100)

		if err := r.call(); err != nil {
			t.Fatal(err)
		}

		u, err := r.client.Usage(r.ctx)
		if err != nil || u.Calls != 1 || u.InputTokens != 10 || u.OutputTokens != 5 || u.Month != "2026-11" {
			t.Errorf("usage = %+v, %v", u, err)
		}

		cred, _, _ := r.client.Vault.Active(r.ctx)
		if !cred.LastUsedAt.Valid {
			t.Error("last_used_at was not set")
		}
	})

	t.Run("rate limit is per minute", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 1000)

		for i := range ai.CallsPerMinute {
			if err := r.call(); err != nil {
				t.Fatalf("call %d: %v", i, err)
			}
		}

		if err := r.call(); !errors.Is(err, ai.ErrTooManyCalls) {
			t.Fatalf("call 21: %v", err)
		}

		*r.clock = r.clock.Add(61 * time.Second)

		if err := r.call(); err != nil {
			t.Fatalf("after the window: %v", err)
		}
	})

	t.Run("monthly limit, and a failed call is given back", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 2)

		r.provider.Reply = func(ai.Request) (ai.Response, error) { return ai.Response{}, ai.ErrUnavailable }
		if err := r.call(); !errors.Is(err, ai.ErrUnavailable) {
			t.Fatalf("failing provider: %v", err)
		}

		r.provider.Reply = nil

		for i := range 2 {
			if err := r.call(); err != nil {
				t.Fatalf("call %d: %v", i, err)
			}
		}

		if err := r.call(); !errors.Is(err, ai.ErrMonthlyLimit) {
			t.Fatalf("third call: %v", err)
		}

		// A new month starts the count again.
		*r.clock = time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC)

		if err := r.call(); err != nil {
			t.Fatalf("next month: %v", err)
		}
	})

	t.Run("the month follows the workspace timezone", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 100)

		_, err := orm.QueryOf[models.Workspace](mustDB(t, r.site)).Where("id", r.ws.ID).
			Update(t.Context(), orm.Set{"timezone": "Asia/Tokyo"})
		if err != nil {
			t.Fatal(err)
		}

		*r.clock = time.Date(2026, 11, 30, 20, 0, 0, 0, time.UTC) // 05:00 on 1 December in Tokyo

		if err := r.call(); err != nil {
			t.Fatal(err)
		}

		if u, _ := r.client.Usage(r.ctx); u.Month != "2026-12" || u.Calls != 1 {
			t.Errorf("usage = %+v, want the call in 2026-12", u)
		}
	})

	t.Run("a provider error text reaches neither the caller nor the log", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 100)

		r.provider.Reply = func(req ai.Request) (ai.Response, error) {
			return ai.Response{}, errors.New("bad request for " + testKey + ": " + req.Messages[0].Content)
		}

		err := r.call()
		if !errors.Is(err, ai.ErrUnavailable) {
			t.Fatalf("err = %v", err)
		}

		for _, secret := range []string{testKey, "0123456789", "Mercado"} {
			if strings.Contains(err.Error(), secret) || strings.Contains(r.log.String(), secret) {
				t.Errorf("%q leaked to the caller or the log: %v / %s", secret, err, r.log.String())
			}
		}
	})

	t.Run("another workspace has its own switch", func(t *testing.T) {
		t.Parallel()

		r := newRig(t)
		saveTestKey(t, r.client.Vault, r.site)
		r.enable(t, 100)

		other := newSite(t, r.app, "bob@example.com")
		saveTestKey(t, r.client.Vault, other)

		_, err := r.client.Call(other.ctx, other.ws.ID, ai.Request{Feature: ai.FeatureCategorize})
		if !errors.Is(err, ai.ErrDisabled) {
			t.Fatalf("a workspace that never enabled AI got %v", err)
		}
	})

}

func TestMonthlyLimitIsAtomic(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	saveTestKey(t, r.client.Vault, r.site)
	r.enable(t, 5)

	// Four calls are already counted: one is left.
	for range 4 {
		row := models.AIUsage{Month: "2026-11", Feature: ai.FeatureCategorize, Provider: ai.KindAnthropic, Status: ai.UsageOK}
		if err := orm.Create(r.ctx, mustDB(t, r.site), &row); err != nil {
			t.Fatal(err)
		}
	}

	const workers = ai.CallsPerMinute

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
		limited int
		others  []error
	)

	start := make(chan struct{})

	for range workers {
		wg.Go(func() {
			<-start

			err := r.call()

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				allowed++
			case errors.Is(err, ai.ErrMonthlyLimit):
				limited++
			default:
				others = append(others, err)
			}
		})
	}

	close(start)
	wg.Wait()

	if allowed != 1 || limited != workers-1 || len(others) != 0 {
		t.Fatalf("allowed = %d, limited = %d, others = %v; want 1, %d, none", allowed, limited, others, workers-1)
	}

	if r.provider.Calls() != 1 {
		t.Errorf("the provider received %d calls, want 1", r.provider.Calls())
	}
}
