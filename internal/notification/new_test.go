package notification

import (
	"testing"
	"time"
)

func TestNew_WithoutRateLimitReturnsPlatformNotifier(t *testing.T) {
	n := New()
	if n == nil {
		t.Fatal("New() should return a non-nil notifier")
	}
	if _, ok := n.(*rateLimitedNotifier); ok {
		t.Error("New() without a rate limit should not wrap in a rate limiter")
	}
}

func TestNew_WithRateLimitReturnsRateLimiter(t *testing.T) {
	n := New(
		WithRateLimit(30*time.Second),
		WithCleanupInterval(time.Minute),
		WithMaxAge(time.Hour),
	)
	rl, ok := n.(*rateLimitedNotifier)
	if !ok {
		t.Fatalf("New() with a rate limit should return *rateLimitedNotifier, got %T", n)
	}
	if rl.minInterval != 30*time.Second {
		t.Errorf("minInterval = %v, want 30s", rl.minInterval)
	}
	if rl.cleanupInterval != time.Minute {
		t.Errorf("cleanupInterval = %v, want 1m", rl.cleanupInterval)
	}
	if rl.maxAge != time.Hour {
		t.Errorf("maxAge = %v, want 1h", rl.maxAge)
	}
	if rl.lastNotified == nil {
		t.Error("lastNotified map should be initialized")
	}
}
