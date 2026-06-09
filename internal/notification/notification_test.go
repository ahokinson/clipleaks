package notification

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeNotifier records calls and optionally returns an error, letting us test
// the rate-limiting wrapper without firing real OS notifications.
type fakeNotifier struct {
	calls int
	err   error
}

func (f *fakeNotifier) Notify(title, message string) error {
	return f.NotifyWithContext(context.Background(), title, message)
}

func (f *fakeNotifier) NotifyWithContext(_ context.Context, title, message string) error {
	f.calls++
	return f.err
}

// newTestRateLimiter builds a rateLimitedNotifier wrapping a fake, bypassing the
// platform notifier that New() would otherwise construct.
func newTestRateLimiter(fake Notifier, minInterval time.Duration) *rateLimitedNotifier {
	return &rateLimitedNotifier{
		notifier:        fake,
		minInterval:     minInterval,
		cleanupInterval: DefaultCleanupInterval,
		maxAge:          DefaultMaxAge,
		lastNotified:    make(map[notificationKey]time.Time),
		lastCleanup:     time.Now(),
	}
}

func TestRateLimiter_SuppressesDuplicateWithinInterval(t *testing.T) {
	fake := &fakeNotifier{}
	r := newTestRateLimiter(fake, time.Hour)

	if err := r.Notify("Alert", "secret found"); err != nil {
		t.Fatalf("first notify: %v", err)
	}
	if err := r.Notify("Alert", "secret found"); err != nil {
		t.Fatalf("second notify should be suppressed silently: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected duplicate to be rate-limited (1 underlying call), got %d", fake.calls)
	}
}

func TestRateLimiter_DistinctKeysNotSuppressed(t *testing.T) {
	fake := &fakeNotifier{}
	r := newTestRateLimiter(fake, time.Hour)

	_ = r.Notify("Alert", "secret A")
	_ = r.Notify("Alert", "secret B")
	_ = r.Notify("Other", "secret A")

	if fake.calls != 3 {
		t.Fatalf("distinct title/message keys should each pass through, got %d calls", fake.calls)
	}
}

func TestRateLimiter_AllowsAfterIntervalElapses(t *testing.T) {
	fake := &fakeNotifier{}
	r := newTestRateLimiter(fake, time.Millisecond)

	_ = r.Notify("Alert", "x")
	time.Sleep(5 * time.Millisecond)
	_ = r.Notify("Alert", "x")

	if fake.calls != 2 {
		t.Fatalf("after interval elapses the notification should resend, got %d calls", fake.calls)
	}
}

func TestRateLimiter_WrapsUnderlyingError(t *testing.T) {
	fake := &fakeNotifier{err: errors.New("boom")}
	r := newTestRateLimiter(fake, time.Hour)

	err := r.Notify("Alert", "x")
	if err == nil {
		t.Fatal("expected error from underlying notifier")
	}
	if !errors.Is(err, ErrNotificationFailed) {
		t.Fatalf("error should wrap ErrNotificationFailed, got: %v", err)
	}
}

func TestRateLimiter_CleanupRemovesStaleEntries(t *testing.T) {
	fake := &fakeNotifier{}
	r := newTestRateLimiter(fake, time.Hour)
	r.maxAge = time.Millisecond

	r.lastNotified[notificationKey{"old", "old"}] = time.Now().Add(-time.Hour)
	r.lastNotified[notificationKey{"fresh", "fresh"}] = time.Now()

	r.cleanupOldEntries()

	if _, ok := r.lastNotified[notificationKey{"old", "old"}]; ok {
		t.Error("stale entry should have been removed")
	}
	if _, ok := r.lastNotified[notificationKey{"fresh", "fresh"}]; !ok {
		t.Error("fresh entry should have been retained")
	}
}
