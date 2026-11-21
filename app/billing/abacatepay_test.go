package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testKey = "abc_dev_test_key"

func TestAbacatePayClient(t *testing.T) {
	t.Parallel()

	var created map[string]any

	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, status string) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": map[string]any{
			"id": "pix_char_1", "amount": 900, "status": status, "brCode": "0002010102",
			"brCodeBase64": "data:image/png;base64,AAAA", "expiresAt": "2026-03-04T15:48:59.876Z",
		}})
	}

	mux.HandleFunc("POST /transparents/create", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&created)
		reply(w, "PENDING")
	})
	mux.HandleFunc("GET /transparents/check", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("id") {
		case "pix_char_1":
			reply(w, "PAID")
		case "boom":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"leaks-the-key-abc_dev_test_key"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"Pix QR Code not found"}`))
		}
	})
	mux.HandleFunc("POST /transparents/simulate-payment", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "pix_char_1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		reply(w, "PAID")
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	c := newAbacatePay(testKey, srv.URL, srv.Client())
	ctx := context.Background()

	ch, err := c.CreatePix(ctx, CreatePixRequest{
		AmountCents: 900, Description: "Pro 1 month", ExpiresInSeconds: 1800, Metadata: map[string]string{"ref": "r1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := created["data"].(map[string]any)
	if created["method"] != "PIX" || data["amount"] != float64(900) || data["expiresIn"] != float64(1800) {
		t.Errorf("create body = %v", created)
	}

	want := time.Date(2026, 3, 4, 15, 48, 59, 876_000_000, time.UTC)
	if ch.ID != "pix_char_1" || ch.Status != ProviderPending || ch.BRCode == "" || !ch.ExpiresAt.Equal(want) {
		t.Errorf("charge = %+v", ch)
	}

	if got, err := c.CheckPix(ctx, "pix_char_1"); err != nil || got.Status != ProviderPaid {
		t.Errorf("check = %+v, %v", got, err)
	}

	if err := c.SimulatePix(ctx, "pix_char_1"); err != nil {
		t.Errorf("simulate: %v", err)
	}

	if _, err := c.CheckPix(ctx, "pix_char_missing"); !errors.Is(err, ErrCharge) {
		t.Errorf("unknown charge = %v, want ErrCharge", err)
	}

	// A provider failure keeps a status and a kind, never the body or the key.
	_, err = c.CheckPix(ctx, "boom")
	if err == nil || strings.Contains(err.Error(), "leaks") || strings.Contains(err.Error(), testKey) {
		t.Errorf("error = %v", err)
	}

	// A wrong key is refused and the key never prints.
	bad := newAbacatePay("other", srv.URL, srv.Client())
	if _, err := bad.CheckPix(ctx, "pix_char_1"); err == nil {
		t.Error("a wrong key was accepted")
	}

	if s := c.String(); strings.Contains(s, testKey) {
		t.Errorf("String() = %q leaks the key", s)
	}
}

func TestAbacatePayWebhookHelpers(t *testing.T) {
	t.Parallel()

	body := []byte(`{"event":"transparent.completed"}`)
	sig := SignForTest(body)

	if !VerifySignature(body, sig) || VerifySignature(append(body, ' '), sig) || VerifySignature(body, "") {
		t.Error("VerifySignature accepted a wrong body or refused a right one")
	}

	if !SecretEqual("s3cret", "s3cret") || SecretEqual("s3cret", "s3cre") || SecretEqual("", "") {
		t.Error("SecretEqual is wrong")
	}
}

func TestNormalizeQR(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"data:image/png;base64,iVBORw0KGgo=": "data:image/png;base64,iVBORw0KGgo=",
		"iVBORw0KGgo=":                       "data:image/png;base64,iVBORw0KGgo=",
		"":                                   "",
		"javascript:alert(1)":                "",
		"data:text/html;base64,PHNjcmlwdD4=": "",
		"https://evil.example/x.png":         "",
		"data:image/png;base64,AA\"onerror=": "",
	} {
		if got := normalizeQR(in); got != want {
			t.Errorf("normalizeQR(%q) = %q, want %q", in, got, want)
		}
	}
}
