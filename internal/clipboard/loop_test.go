package clipboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// scriptedReader returns a sequence of (content, err) results, repeating the
// last one once the script is exhausted.
type scriptedReader struct {
	mu      sync.Mutex
	results []readResult
	idx     int
}

type readResult struct {
	content string
	err     error
}

func (s *scriptedReader) readClipboard(_ context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.results[s.idx]
	if s.idx < len(s.results)-1 {
		s.idx++
	}
	return r.content, r.err
}

func drainUntilClosed(ch <-chan Content) []Content {
	var got []Content
	for c := range ch {
		got = append(got, c)
	}
	return got
}

func TestRunMonitorLoop_EmitsChangedContentOnce(t *testing.T) {
	reader := &scriptedReader{results: []readResult{{content: "hello"}}}
	ch := make(chan Content, 16)
	ctx, cancel := context.WithCancel(context.Background())

	cfg := monitorConfig{maxSize: 1024, debounceInterval: 0, pollInterval: time.Millisecond}

	done := make(chan struct{})
	go func() {
		runMonitorLoop(ctx, reader, cfg, ch)
		close(done)
	}()

	// Wait for the first emission, then stop.
	select {
	case first := <-ch:
		if first.Text != "hello" {
			t.Errorf("emitted Text = %q, want %q", first.Text, "hello")
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("timed out waiting for first emission")
	}
	cancel()
	<-done

	// Identical content should not have produced more than the first emission.
	rest := drainUntilClosed(ch)
	if len(rest) != 0 {
		t.Errorf("expected no further emissions for unchanged content, got %d", len(rest))
	}
}

func TestRunMonitorLoop_StopsOnContextCancel(t *testing.T) {
	reader := &scriptedReader{results: []readResult{{err: errors.New("clipboard busy")}}}
	ch := make(chan Content, 4)
	ctx, cancel := context.WithCancel(context.Background())

	cfg := monitorConfig{maxSize: 1024, debounceInterval: 0, pollInterval: time.Millisecond}

	done := make(chan struct{})
	go func() {
		runMonitorLoop(ctx, reader, cfg, ch)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond) // let it spin through some read errors
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runMonitorLoop did not stop after context cancellation")
	}

	if got := drainUntilClosed(ch); len(got) != 0 {
		t.Errorf("erroring reader should emit nothing, got %d", len(got))
	}
}

func TestRunMonitorLoop_TruncatesLargeContent(t *testing.T) {
	reader := &scriptedReader{results: []readResult{{content: "abcdefghij"}}}
	ch := make(chan Content, 16)
	ctx, cancel := context.WithCancel(context.Background())

	cfg := monitorConfig{maxSize: 4, debounceInterval: 0, pollInterval: time.Millisecond}

	done := make(chan struct{})
	go func() {
		runMonitorLoop(ctx, reader, cfg, ch)
		close(done)
	}()

	select {
	case c := <-ch:
		if c.Text != "abcd" || !c.Truncated {
			t.Errorf("expected truncated %q, got Text=%q Truncated=%v", "abcd", c.Text, c.Truncated)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("timed out waiting for emission")
	}
	cancel()
	<-done
}
