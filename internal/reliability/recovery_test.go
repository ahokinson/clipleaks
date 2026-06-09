package reliability

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWithRecover_SwallowsPanic(t *testing.T) {
	ran := false
	// Should not propagate the panic past WithRecover.
	WithRecover("test", func() {
		ran = true
		panic("boom")
	})
	if !ran {
		t.Error("function body should have executed")
	}
}

func TestWithRecover_NoPanic(t *testing.T) {
	called := false
	WithRecover("test", func() { called = true })
	if !called {
		t.Error("function should have been called")
	}
}

func TestRetry_SucceedsAfterTransientFailures(t *testing.T) {
	attempts := 0
	cfg := &RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}

	err := Retry(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	}, cfg)

	if err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_ExhaustsAndReturnsLastError(t *testing.T) {
	attempts := 0
	sentinel := errors.New("always fails")
	cfg := &RetryConfig{MaxRetries: 2, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}

	err := Retry(func() error {
		attempts++
		return sentinel
	}, cfg)

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("returned error should wrap the last failure, got: %v", err)
	}
	if attempts != cfg.MaxRetries+1 {
		t.Fatalf("expected %d attempts, got %d", cfg.MaxRetries+1, attempts)
	}
}

func TestRetry_NilConfigUsesDefaults(t *testing.T) {
	called := false
	if err := Retry(func() error { called = true; return nil }, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("operation should have been invoked with default config")
	}
}

func TestLimitStackDepth_Truncates(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("goroutine 1 [running]:\n")
	for range 100 {
		sb.WriteString("some.Func(...)\n\t/path/to/file.go:42\n")
	}

	out := limitStackDepth([]byte(sb.String()), 5)
	if !strings.Contains(out, "additional frames omitted") {
		t.Error("expected truncation marker in oversized stack trace")
	}
	if len(out) >= sb.Len() {
		t.Error("truncated output should be shorter than the original stack")
	}
}
