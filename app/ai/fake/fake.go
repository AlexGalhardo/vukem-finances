// Package fake is a deterministic AI provider for tests: it never opens a
// socket, so no test can reach a real provider or need a real key.
package fake

import (
	"context"
	"sync"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

// Provider answers from memory and records what it was asked.
type Provider struct {
	// Reply, when set, builds the answer; the default echoes a fixed JSON text.
	Reply func(ai.Request) (ai.Response, error)
	// CheckErr is returned by Check.
	CheckErr error

	mu       sync.Mutex
	requests []ai.Request
	checks   int
}

// Complete implements ai.Provider.
func (p *Provider) Complete(_ context.Context, req ai.Request) (ai.Response, error) {
	p.mu.Lock()
	p.requests = append(p.requests, req)
	p.mu.Unlock()

	if p.Reply != nil {
		return p.Reply(req)
	}

	return ai.Response{Text: `{"ok":true}`, InputTokens: 10, OutputTokens: 5, Model: "fake-1", Provider: "fake"}, nil
}

// Check implements ai.Provider.
func (p *Provider) Check(context.Context) error {
	p.mu.Lock()
	p.checks++
	p.mu.Unlock()

	return p.CheckErr
}

// Calls is how many Complete calls arrived.
func (p *Provider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.requests)
}

// Checks is how many Check calls arrived.
func (p *Provider) Checks() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.checks
}

// Requests returns a copy of what Complete received.
func (p *Provider) Requests() []ai.Request {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]ai.Request(nil), p.requests...)
}

// Factory returns an ai.Factory that always yields p.
func (p *Provider) Factory() ai.Factory {
	return func(ai.ProviderConfig) (ai.Provider, error) { return p, nil }
}
