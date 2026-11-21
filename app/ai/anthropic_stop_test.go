package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func anthropicStub(t *testing.T, stop string, got *map[string]any) *ai.Anthropic {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got != nil {
			_ = json.NewDecoder(r.Body).Decode(got)
		}

		_, _ = io.WriteString(w, fmt.Sprintf(`{"model":"m","content":[{"type":"text","text":"partial"}],`+
			`"stop_reason":%q,"usage":{"input_tokens":1,"output_tokens":1}}`, stop))
	}))
	t.Cleanup(srv.Close)

	p := ai.NewAnthropic(ai.NewKey(testKey), "", localHTTP())
	p.Endpoint = srv.URL

	return p
}

func TestAnthropicRequestDefaults(t *testing.T) {
	t.Parallel()

	var got map[string]any

	_, err := anthropicStub(t, "end_turn", &got).Complete(t.Context(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "oi"}}, Temperature: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got["model"] != "claude-opus-5-5" {
		t.Errorf("model = %v", got["model"])
	}

	if _, sent := got["temperature"]; sent {
		t.Error("temperature was sent: current models answer 400 to it")
	}

	if got["max_tokens"] != float64(16000) {
		t.Errorf("max_tokens = %v", got["max_tokens"])
	}
}

func TestAnthropicStopReasons(t *testing.T) {
	t.Parallel()

	for _, stop := range []string{"refusal", "max_tokens"} {
		_, err := anthropicStub(t, stop, nil).Complete(t.Context(), ai.Request{
			Messages: []ai.Message{{Role: "user", Content: "oi"}},
		})
		if !errors.Is(err, ai.ErrBadRequest) {
			t.Errorf("stop_reason %s: err = %v, want ErrBadRequest", stop, err)
		}
	}

	if err := anthropicStub(t, "max_tokens", nil).Check(t.Context()); err != nil {
		t.Errorf("a check that hits its 16 tokens proves the key: %v", err)
	}

	if err := anthropicStub(t, "refusal", nil).Check(t.Context()); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("a refused check = %v", err)
	}
}

func TestCanceledCallReleasesItsReservation(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	saveTestKey(t, r.client.Vault, r.site)
	r.enable(t, 5)

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	r.provider.Reply = func(ai.Request) (ai.Response, error) {
		cancel()

		return ai.Response{}, ctx.Err()
	}

	_, err := r.client.Call(ctx, r.ws.ID, ai.Request{
		Feature: ai.FeatureCategorize, Messages: []ai.Message{{Role: "user", Content: "Mercado"}}, MaxTokens: 50,
	})
	if err == nil {
		t.Fatal("the canceled call must fail")
	}

	row, err := orm.QueryOf[models.AIUsage](mustDB(t, r.site)).First(r.ctx)
	if err != nil {
		t.Fatal(err)
	}

	if row.Status != ai.UsageAborted {
		t.Errorf("status = %q, want %q: the provider may have billed a canceled call", row.Status, ai.UsageAborted)
	}

	usage, err := r.client.Usage(r.ctx)
	if err != nil || usage.Calls != 1 {
		t.Errorf("counted calls = %d, %v, want 1", usage.Calls, err)
	}
}

func TestBilledRefusalStaysCounted(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	saveTestKey(t, r.client.Vault, r.site)
	r.enable(t, 5)

	r.provider.Reply = func(ai.Request) (ai.Response, error) {
		return ai.Response{InputTokens: 7, OutputTokens: 3, Model: "m"},
			&ai.ProviderError{Status: http.StatusOK, Type: "refusal", Class: ai.ErrBadRequest}
	}

	if err := r.call(); !errors.Is(err, ai.ErrBadRequest) {
		t.Fatalf("err = %v", err)
	}

	usage, err := r.client.Usage(r.ctx)
	if err != nil || usage.Calls != 1 || usage.InputTokens != 7 || usage.OutputTokens != 3 {
		t.Errorf("usage = %+v, %v, want one call with its real tokens", usage, err)
	}
}
