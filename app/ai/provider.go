package ai

import (
	"context"
	"errors"
	"slices"
)

// Provider kinds a credential may name.
const (
	KindAnthropic        = "anthropic"
	KindOpenAI           = "openai"
	KindOpenAICompatible = "openai_compatible"
)

// Kinds lists the providers the application can talk to.
func Kinds() []string { return []string{KindAnthropic, KindOpenAI, KindOpenAICompatible} }

// ValidKind reports whether kind names a provider.
func ValidKind(kind string) bool { return slices.Contains(Kinds(), kind) }

// Message is one turn of a conversation: Role is "user" or "assistant".
type Message struct {
	Role    string
	Content string
}

// Request is what a feature asks of a provider.
type Request struct {
	System   string
	Messages []Message
	// JSONSchema, when set, asks for structured output.
	JSONSchema  []byte
	MaxTokens   int
	Temperature float64
	// Feature names the caller for the usage meter ("categorize", "assistant").
	Feature string
}

// Response is what a provider answered.
type Response struct {
	Text         string
	InputTokens  int
	OutputTokens int
	Model        string
	Provider     string
}

// Provider is one AI service. Implementations live in one file each.
type Provider interface {
	// Complete sends the request and returns the answer.
	Complete(ctx context.Context, req Request) (Response, error)
	// Check makes the cheapest call that proves the key works.
	Check(ctx context.Context) error
}

// Errors a provider returns. They are generic on purpose: their text is the
// only thing that may reach a browser. The upstream body never travels in them.
var (
	// ErrRejected means the provider refused the key (401, 403).
	ErrRejected = errors.New("the provider rejected the key")
	// ErrUnavailable means the provider failed, timed out or was unreachable.
	ErrUnavailable = errors.New("the provider is unavailable")
	// ErrRateLimited means the provider asked to slow down (429) even after the retry.
	ErrRateLimited = errors.New("the provider is rate limiting this key")
	// ErrBadRequest means the provider refused the request itself (other 4xx).
	ErrBadRequest = errors.New("the provider could not handle the request")
)
