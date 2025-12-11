package reliability

import (
	"bytes"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"
)

const (
	MaxStackFrames = 20 // Limit stack trace depth to reduce potential information leakage
)

func WithRecover(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			stack := limitStackDepth(debug.Stack(), MaxStackFrames)
			slog.Error("Panic recovered",
				"component", name,
				"panic", r,
				"stack", stack)
		}
	}()
	fn()
}

func limitStackDepth(stack []byte, maxFrames int) string {
	lines := bytes.Split(stack, []byte("\n"))

	if len(lines) == 0 {
		return string(stack)
	}

	maxLines := 1 + (maxFrames * 2)

	if len(lines) <= maxLines {
		return string(stack)
	}

	truncated := lines[:maxLines]
	omittedFrames := (len(lines) - maxLines) / 2
	truncated = append(truncated, []byte(fmt.Sprintf("... (%d additional frames omitted for security)", omittedFrames)))

	return string(bytes.Join(truncated, []byte("\n")))
}

type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		InitialBackoff: time.Second,
		MaxBackoff:     10 * time.Second,
	}
}

func Retry(operation func() error, cfg *RetryConfig) error {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	var lastErr error
	backoff := cfg.InitialBackoff
	startTime := time.Now()

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("Retrying operation",
				"attempt", attempt,
				"max_retries", cfg.MaxRetries,
				"backoff_seconds", backoff.Seconds())
			time.Sleep(backoff)

			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}

		err := operation()
		if err == nil {
			if attempt > 0 {
				elapsed := time.Since(startTime)
				slog.Info("Operation succeeded after retry",
					"attempts", attempt+1,
					"elapsed_seconds", elapsed.Seconds())
			}
			return nil
		}

		lastErr = err
		slog.Debug("Operation failed",
			"attempt", attempt+1,
			"error", err)
	}

	elapsed := time.Since(startTime)
	return fmt.Errorf("operation failed after %d attempts (took %v): %w", cfg.MaxRetries+1, elapsed, lastErr)
}
