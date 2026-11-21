// Package ai is the one safe layer between the application and an AI
// provider: the user's key (bring your own key, D-007), its encryption, the
// redaction of what is sent, the limits and the providers themselves. Every
// AI feature goes through Client.Call.
package ai

import (
	"fmt"
	"log/slog"
)

const redacted = "[redacted]"

// Key is the API key of a user. It prints as [redacted] under every verb,
// in JSON, text and logs, so it cannot reach a log line, a page prop or an
// error message by accident. Reveal is the only way to read it, and is
// called where one outbound request is built.
type Key struct{ v string }

// NewKey wraps a key.
func NewKey(s string) Key { return Key{v: s} }

// Reveal returns the key in clear.
func (k Key) Reveal() string { return k.v }

// IsZero reports whether there is no key.
func (k Key) IsZero() bool { return k.v == "" }

// Hint is the last four characters of the key, the only part the interface
// may show. A key too short to hide anything behind its tail has no hint.
func (k Key) Hint() string {
	if len(k.v) < 12 {
		return ""
	}

	return k.v[len(k.v)-4:]
}

// String implements fmt.Stringer.
func (Key) String() string { return redacted }

// GoString implements fmt.GoStringer.
func (Key) GoString() string { return redacted }

// Format implements fmt.Formatter, so no verb (%v, %+v, %#v, %s, %q, %x)
// prints the key.
func (Key) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(redacted)) }

// MarshalJSON implements json.Marshaler.
func (Key) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// MarshalText implements encoding.TextMarshaler.
func (Key) MarshalText() ([]byte, error) { return []byte(redacted), nil }

// LogValue implements slog.LogValuer.
func (Key) LogValue() slog.Value { return slog.StringValue(redacted) }
