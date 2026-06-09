package clipboard

import (
	"testing"
	"time"

	"github.com/ahokinson/clipleaks/internal/security"
)

func TestContent_Wipe(t *testing.T) {
	c := Content{Text: "secret"}
	c.Wipe()
	if c.Text != "" {
		t.Errorf("Text after Wipe = %q, want empty", c.Text)
	}
	// Wiping empty content must be a no-op, not a panic.
	c.Wipe()
}

func TestShouldProcessContent(t *testing.T) {
	debounce := 100 * time.Millisecond

	t.Run("empty content skipped", func(t *testing.T) {
		if shouldProcessContent("", nil, time.Time{}, debounce) {
			t.Error("empty content should not be processed")
		}
	})

	t.Run("first non-empty content processed", func(t *testing.T) {
		if !shouldProcessContent("hello", nil, time.Time{}, debounce) {
			t.Error("first content should be processed")
		}
	})

	t.Run("identical content skipped", func(t *testing.T) {
		last := security.New("hello")
		if shouldProcessContent("hello", last, time.Time{}, debounce) {
			t.Error("unchanged content should not be reprocessed")
		}
	})

	t.Run("changed content within debounce skipped", func(t *testing.T) {
		last := security.New("hello")
		if shouldProcessContent("world", last, time.Now(), debounce) {
			t.Error("changed content within debounce window should be skipped")
		}
	})

	t.Run("changed content after debounce processed", func(t *testing.T) {
		last := security.New("hello")
		old := time.Now().Add(-time.Second)
		if !shouldProcessContent("world", last, old, debounce) {
			t.Error("changed content after debounce window should be processed")
		}
	})
}

func TestProcessNewContent_EmitsAndUpdatesState(t *testing.T) {
	ch := make(chan Content, 1)
	var last *security.SecureString
	var lastChange time.Time

	processNewContent("hello", &last, &lastChange, ch, 1024)

	got := <-ch
	if got.Text != "hello" {
		t.Errorf("emitted Text = %q, want %q", got.Text, "hello")
	}
	if got.Truncated {
		t.Error("content within maxSize should not be marked truncated")
	}
	if last == nil || last.String() != "hello" {
		t.Error("lastContent should be updated to the new value")
	}
	if lastChange.IsZero() {
		t.Error("lastChangeTime should be updated")
	}
}

func TestProcessNewContent_TruncatesOversizedContent(t *testing.T) {
	ch := make(chan Content, 1)
	var last *security.SecureString
	var lastChange time.Time

	processNewContent("abcdefghij", &last, &lastChange, ch, 4)

	got := <-ch
	if got.Text != "abcd" {
		t.Errorf("truncated Text = %q, want %q", got.Text, "abcd")
	}
	if !got.Truncated {
		t.Error("oversized content should be marked truncated")
	}
}

func TestHandleReadError_ResetsAtThreshold(t *testing.T) {
	var lastErr time.Time
	count := 0
	for range MaxConsecutiveErrors {
		count = handleReadError(errTest, count, &lastErr)
	}
	// On reaching MaxConsecutiveErrors the counter resets to 0.
	if count != 0 {
		t.Errorf("counter should reset to 0 at threshold, got %d", count)
	}
}

func TestHandleReadError_IncrementsBelowThreshold(t *testing.T) {
	var lastErr time.Time
	count := handleReadError(errTest, 0, &lastErr)
	if count != 1 {
		t.Errorf("first error should yield count 1, got %d", count)
	}
	count = handleReadError(errTest, count, &lastErr)
	if count != 2 {
		t.Errorf("second error should yield count 2, got %d", count)
	}
}

var errTest = ErrClipboardReadFailed
