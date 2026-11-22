package telegram

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// counterFake counts per key, and lets a test keep the minute budget fresh
// while the day budget runs down.
type counterFake struct {
	mu       sync.Mutex
	counts   map[string]int64
	freshMin bool
}

func (c *counterFake) Increment(_ context.Context, key string, delta int64, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.freshMin && strings.Contains(key, "tg-minute") {
		return 1, nil
	}

	c.counts[key] += delta

	return c.counts[key], nil
}

func TestDayLimit(t *testing.T) {
	t.Parallel()

	l := NewLimits(&counterFake{counts: map[string]int64{}, freshMin: true})

	for i := range config.TelegramMessagesPerDay {
		if v := l.Message(t.Context(), 1); v != Allow {
			t.Fatalf("message %d of the day: verdict %v, want Allow", i+1, v)
		}
	}

	if v := l.Message(t.Context(), 1); v == Allow {
		t.Fatal("the 201st message of the day was allowed")
	}

	// Another chat is untouched.
	if v := l.Message(t.Context(), 2); v != Allow {
		t.Fatalf("another chat: %v", v)
	}
}
