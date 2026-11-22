// Package desktopgate protects the embedded server of the desktop app. Any
// program on the machine can reach a loopback port, so every request must come
// from a loopback peer with a loopback Host and no proxy headers, and must
// carry the cookie minted by the one-time exchange of the launch token. The
// gate knows nothing about Wails: the desktop shell and the tests both wrap an
// http.Handler with it.
package desktopgate

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// CookieName is the cookie that proves a request comes from the app window.
	CookieName = "vf_desktop"
	// EnterPrefix is the path the window opens first; the launch token follows.
	EnterPrefix = "/desktop/enter/"

	// minTokenLen is the base64 length of 32 random bytes (256 bits).
	minTokenLen = 43
)

// proxyHeaders mark a request that went through something other than a direct
// local connection; the gate refuses them all.
var proxyHeaders = []string{
	"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto",
	"X-Forwarded-Port", "X-Forwarded-Server", "X-Real-Ip", "X-Client-Ip",
	"True-Client-Ip", "Cf-Connecting-Ip", "Via",
}

type verifiedKey struct{}

// Verified reports whether the request that carries ctx passed the gate with
// a valid cookie. It is false when no gate wraps the handler, so code that
// trusts it (the setup form) is inert outside the desktop app.
func Verified(ctx context.Context) bool {
	ok, _ := ctx.Value(verifiedKey{}).(bool)

	return ok
}

// Gate is the per-run guard. Build it with New, wrap the handler with Wrap.
type Gate struct {
	tokenHash [sha256.Size]byte
	spent     atomic.Bool

	mu         sync.RWMutex
	cookieHash [sha256.Size]byte
	hasCookie  bool
}

// NewToken returns a fresh launch token with 256 bits of entropy.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// New returns a gate that accepts token once. A token shorter than 256 bits of
// base64 is refused so a weak secret cannot be mounted by mistake.
func New(token string) (*Gate, error) {
	if len(token) < minTokenLen || strings.ContainsAny(token, "/?#% ") {
		return nil, errors.New("desktopgate: the launch token must be at least 43 URL-safe characters")
	}

	return &Gate{tokenHash: sha256.Sum256([]byte(token))}, nil
}

// EntryPath is the path (with the token) the window opens first.
func EntryPath(token string) string { return EnterPrefix + token }

// Wrap returns next behind the gate. It is the outermost handler, so nothing
// of the app (health checks, static files, webhooks) answers without it.
func (g *Gate) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = &headerWriter{ResponseWriter: w}

		if !localRequest(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, EnterPrefix) {
			g.exchange(w, r, strings.TrimPrefix(r.URL.Path, EnterPrefix))
			return
		}

		if !g.validCookie(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), verifiedKey{}, true)))
	})
}

// headerWriter adds the gate's response headers at the moment the header is
// sent, so a handler that sets its own Referrer-Policy cannot weaken it.
type headerWriter struct {
	http.ResponseWriter
	done bool
}

func (w *headerWriter) seal() {
	if w.done {
		return
	}

	w.done = true

	h := w.Header()
	// Not no-referrer: a form that fails validation returns to its page through the
	// Referer, and without it the error is lost. No other site ever receives it.
	h.Set("Referrer-Policy", "same-origin")
	// A second policy narrows the first: both must allow what a page does.
	h.Add("Content-Security-Policy", "frame-ancestors 'none'")
}

func (w *headerWriter) WriteHeader(status int) {
	w.seal()
	w.ResponseWriter.WriteHeader(status)
}

func (w *headerWriter) Write(p []byte) (int, error) {
	w.seal()
	return w.ResponseWriter.Write(p)
}

// Flush keeps streaming responses (server-sent events) working.
func (w *headerWriter) Flush() {
	w.seal()

	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real writer.
func (w *headerWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// exchange spends the launch token and mints the cookie. A wrong or spent token
// answers 401 and never spends the real one.
func (g *Gate) exchange(w http.ResponseWriter, r *http.Request, token string) {
	got := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(got[:], g.tokenHash[:]) != 1 || !g.spent.CompareAndSwap(false, true) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	value := base64.RawURLEncoding.EncodeToString(raw)

	g.mu.Lock()
	g.cookieHash, g.hasCookie = sha256.Sum256([]byte(value)), true
	g.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (g *Gate) validCookie(r *http.Request) bool {
	ck, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}

	got := sha256.Sum256([]byte(ck.Value))

	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.hasCookie && subtle.ConstantTimeCompare(got[:], g.cookieHash[:]) == 1
}

// localRequest is true for a direct loopback connection to a loopback Host that
// no other web page tries to reach from a different origin.
func localRequest(r *http.Request) bool {
	ap, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil || !ap.Addr().Unmap().IsLoopback() {
		return false
	}

	for _, name := range proxyHeaders {
		if len(r.Header.Values(name)) > 0 {
			return false
		}
	}

	if !loopbackHost(r.Host) {
		return false
	}

	// A page served by another local program is same-site, and cookies ignore
	// ports, so the browser would still send the cookie: refuse it by origin.
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
	default:
		return false
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || !strings.EqualFold(u.Host, r.Host) {
			return false
		}
	}

	return true
}

func loopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}

	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")

	switch strings.ToLower(host) {
	case "localhost":
		return true
	}

	ip, err := netip.ParseAddr(host)

	return err == nil && ip.Unmap().IsLoopback()
}
