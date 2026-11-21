package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/httpclient"
)

// The AbacatePay API (https://docs.abacatepay.com, pages transparents/create,
// transparents/check and transparents/simulate-payment). Amounts are cents.
const (
	abacatePayBase    = "https://api.abacatepay.com/v2"
	abacatePayTimeout = 15 * time.Second
	abacatePayMaxBody = 1 << 20

	// pixPublicKey is the public key AbacatePay documents for the body
	// signature of its webhooks (page notificacoes/seguranca). It is the same
	// for every account, so the signature proves the integrity of a body, not
	// where it came from.
	pixPublicKey = "t9dXRhHHo3yDEj5pVDYz0frf7q6bMKyMRmxxCPIPp3RCplBfXRxqlC6ZpiWmOqj4L63qEaeUOtrCI8P0VMUgo6iIga2ri9ogaHFs0WIIywSMg0q7RmBfybe1E5XJcfC4IW3alNqym0tXoAKkzvfEjZxV6bE0oG2zJrNNYmUCKZyV0KZ3JS8Votf9EAWWYdiDkMkpbMdPggfh1EqHlVkMiTady6jOR3hyzGEHrIz2Ret0xHKMbiqkr9HS1JhNHDX9"
)

// Status of a charge at AbacatePay, as the API spells them.
const (
	ProviderPending      = "PENDING"
	ProviderPaid         = "PAID"
	ProviderExpired      = "EXPIRED"
	ProviderCancelled    = "CANCELLED"
	ProviderRefunded     = "REFUNDED"
	ProviderUnderDispute = "UNDER_DISPUTE"
)

// PixCharge is a PIX charge as AbacatePay reports it.
type PixCharge struct {
	ID          string    `json:"id"`
	AmountCents int64     `json:"amount"`
	Status      string    `json:"status"`
	BRCode      string    `json:"brCode"`
	BRCodeImage string    `json:"brCodeBase64"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// CreatePixRequest is what the server asks AbacatePay to charge.
type CreatePixRequest struct {
	AmountCents      int64
	Description      string
	ExpiresInSeconds int
	Metadata         map[string]string
}

// AbacatePay is the client of the PIX provider. The key is never logged,
// returned or put in an error.
type AbacatePay struct {
	key  string
	base string
	http *http.Client
}

// NewAbacatePay builds the client with the safe HTTP client (no private
// hosts, no redirects, capped body). base is empty outside tests.
func NewAbacatePay(key, base string) *AbacatePay {
	c := httpclient.Safe(httpclient.Options{Timeout: abacatePayTimeout, MaxBody: abacatePayMaxBody})
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return newAbacatePay(key, base, c)
}

func newAbacatePay(key, base string, c *http.Client) *AbacatePay {
	if base == "" {
		base = abacatePayBase
	}

	return &AbacatePay{key: key, base: strings.TrimSuffix(base, "/"), http: c}
}

// String keeps the key out of a log line or a %v.
func (a *AbacatePay) String() string { return "billing.AbacatePay{key: [redacted]}" }

// GoString keeps the key out of a %#v.
func (a *AbacatePay) GoString() string { return a.String() }

// ErrCharge is a charge AbacatePay does not know.
var ErrCharge = errors.New("pix charge not found")

type envelope struct {
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
}

// CreatePix creates a PIX charge (POST /transparents/create).
func (a *AbacatePay) CreatePix(ctx context.Context, req CreatePixRequest) (PixCharge, error) {
	data := map[string]any{"amount": req.AmountCents, "expiresIn": req.ExpiresInSeconds}
	if req.Description != "" {
		data["description"] = req.Description
	}

	if len(req.Metadata) > 0 {
		data["metadata"] = req.Metadata
	}

	return a.charge(ctx, http.MethodPost, "/transparents/create", map[string]any{"method": "PIX", "data": data})
}

// CheckPix reads the current state of a PIX charge (GET /transparents/check).
func (a *AbacatePay) CheckPix(ctx context.Context, id string) (PixCharge, error) {
	return a.charge(ctx, http.MethodGet, "/transparents/check?id="+url.QueryEscape(id), nil)
}

// SimulatePix pays a charge in Test Mode (POST /transparents/simulate-payment);
// a production key makes the provider answer an error.
func (a *AbacatePay) SimulatePix(ctx context.Context, id string) error {
	_, err := a.charge(ctx, http.MethodPost, "/transparents/simulate-payment?id="+url.QueryEscape(id), nil)
	return err
}

func (a *AbacatePay) charge(ctx context.Context, method, path string, body any) (PixCharge, error) {
	var payload io.Reader

	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return PixCharge{}, &providerError{Provider: "abacatepay", Kind: "encode"}
		}

		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.base+path, payload)
	if err != nil {
		return PixCharge{}, &providerError{Provider: "abacatepay", Kind: "request"}
	}

	req.Header.Set("Authorization", "Bearer "+a.key)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := a.http.Do(req)
	if err != nil {
		return PixCharge{}, &providerError{Provider: "abacatepay", Kind: "transport"}
	}

	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, abacatePayMaxBody))
	if err != nil {
		return PixCharge{}, &providerError{Provider: "abacatepay", Status: res.StatusCode, Kind: "transport"}
	}

	if res.StatusCode == http.StatusNotFound {
		return PixCharge{}, ErrCharge
	}

	var env envelope
	if res.StatusCode < 200 || res.StatusCode > 299 || json.Unmarshal(raw, &env) != nil || !env.Success {
		return PixCharge{}, &providerError{Provider: "abacatepay", Status: res.StatusCode, Kind: "api_error"}
	}

	var ch PixCharge
	if err := json.Unmarshal(env.Data, &ch); err != nil || ch.ID == "" {
		return PixCharge{}, &providerError{Provider: "abacatepay", Status: res.StatusCode, Kind: "bad_response"}
	}

	ch.BRCodeImage = normalizeQR(ch.BRCodeImage)

	return ch, nil
}

const pngDataURL = "data:image/png;base64,"

var base64Body = regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)

// normalizeQR returns the QR image as a PNG data URL, or "" when the provider's
// value is anything else: the value goes into an <img src>, so only a PNG made
// of base64 characters is passed on.
func normalizeQR(v string) string {
	body := strings.TrimPrefix(v, pngDataURL)
	if body == "" || !base64Body.MatchString(body) {
		return ""
	}

	return pngDataURL + body
}

// signBody is the X-Webhook-Signature of body: HMAC-SHA256 with the public
// key, base64.
func signBody(body []byte) string {
	mac := hmac.New(sha256.New, []byte(pixPublicKey))
	_, _ = mac.Write(body)

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks the X-Webhook-Signature header against the raw body
// in constant time.
func VerifySignature(body []byte, header string) bool {
	return subtle.ConstantTimeCompare([]byte(signBody(body)), []byte(header)) == 1
}

// SignForTest returns the signature AbacatePay would send for body.
func SignForTest(body []byte) string { return signBody(body) }

// SecretEqual compares a shared secret in constant time. Both sides are
// hashed first so the comparison takes the same time whatever the lengths.
func SecretEqual(got, want string) bool {
	if want == "" {
		return false
	}

	a, b := sha256.Sum256([]byte(got)), sha256.Sum256([]byte(want))

	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
