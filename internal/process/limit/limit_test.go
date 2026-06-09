package limit

import (
	"testing"
	"time"
)

func TestLimiter_AllowsInitialBurst(t *testing.T) {
	// refillRate 0 so the bucket never refills during the test: exactly maxQuota
	// calls should succeed, then deny.
	l := New(3, 0)
	for i := range 3 {
		if !l.Allow() {
			t.Fatalf("call %d should be allowed within the burst of 3", i+1)
		}
	}
	if l.Allow() {
		t.Fatal("4th call should be denied after the burst is exhausted")
	}
}

func TestLimiter_RefillsOverTime(t *testing.T) {
	// maxQuota 1, refill 1000 tokens/sec. Drain the single token, then a short
	// sleep should refill enough to allow again.
	l := New(1, 1000)
	if !l.Allow() {
		t.Fatal("first call should be allowed")
	}
	if l.Allow() {
		t.Fatal("second immediate call should be denied")
	}

	time.Sleep(10 * time.Millisecond) // ~10 tokens refilled, capped at 1
	if !l.Allow() {
		t.Fatal("call after refill window should be allowed")
	}
}

func TestLimiter_QuotaCappedAtMax(t *testing.T) {
	// Wait long enough that an uncapped bucket would accumulate many tokens, then
	// confirm only maxQuota (2) calls succeed.
	l := New(2, 1000)
	time.Sleep(20 * time.Millisecond)

	allowed := 0
	for range 10 {
		if l.Allow() {
			allowed++
		}
	}
	if allowed != 2 {
		t.Fatalf("quota should be capped at 2, but %d calls were allowed", allowed)
	}
}
