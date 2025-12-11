package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	DefaultCleanupInterval = 5 * time.Minute // Clean up old notification timestamps periodically
	DefaultMaxAge          = 1 * time.Hour   // Remove notification records older than 1 hour to prevent memory growth
)

type Notifier interface {
	Notify(title, message string) error
	NotifyWithContext(ctx context.Context, title, message string) error
}

type Option func(*config)

type config struct {
	minInterval     time.Duration
	cleanupInterval time.Duration
	maxAge          time.Duration
}

type notificationKey struct {
	title   string
	message string
}

type rateLimitedNotifier struct {
	notifier        Notifier
	minInterval     time.Duration
	cleanupInterval time.Duration
	maxAge          time.Duration
	lastNotified    map[notificationKey]time.Time
	mu              sync.Mutex
	lastCleanup     time.Time
}

func WithRateLimit(interval time.Duration) Option {
	return func(c *config) {
		c.minInterval = interval
	}
}

func WithCleanupInterval(interval time.Duration) Option {
	return func(c *config) {
		c.cleanupInterval = interval
	}
}

func WithMaxAge(age time.Duration) Option {
	return func(c *config) {
		c.maxAge = age
	}
}

func New(opts ...Option) Notifier {
	cfg := &config{
		minInterval:     0,
		cleanupInterval: DefaultCleanupInterval,
		maxAge:          DefaultMaxAge,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.minInterval == 0 {
		return newPlatformNotifier()
	}

	slog.Info("Rate-limited notifier initialized",
		"min_interval_seconds", cfg.minInterval.Seconds(),
		"cleanup_interval_seconds", cfg.cleanupInterval.Seconds(),
		"max_age_seconds", cfg.maxAge.Seconds())

	return &rateLimitedNotifier{
		notifier:        newPlatformNotifier(),
		minInterval:     cfg.minInterval,
		cleanupInterval: cfg.cleanupInterval,
		maxAge:          cfg.maxAge,
		lastNotified:    make(map[notificationKey]time.Time),
		lastCleanup:     time.Now(),
	}
}

func (r *rateLimitedNotifier) Notify(title, message string) error {
	return r.NotifyWithContext(context.Background(), title, message)
}

func (r *rateLimitedNotifier) NotifyWithContext(ctx context.Context, title, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := notificationKey{title: title, message: message}

	if lastTime, exists := r.lastNotified[key]; exists {
		timeSince := time.Since(lastTime)
		if timeSince < r.minInterval {
			slog.Debug("Notification rate limited",
				"title", title,
				"time_since_last_seconds", timeSince.Seconds(),
				"min_interval_seconds", r.minInterval.Seconds())
			return nil
		}
	}

	if err := r.notifier.Notify(title, message); err != nil {
		slog.Error("Failed to send notification", "title", title, "error", err)
		return fmt.Errorf("%w: %w", ErrNotificationFailed, err)
	}

	slog.Info("Notification sent", "title", title, "message", message)

	r.lastNotified[key] = time.Now()

	if time.Since(r.lastCleanup) >= r.cleanupInterval {
		r.cleanupOldEntries()
		r.lastCleanup = time.Now()
	}

	return nil
}

func (r *rateLimitedNotifier) cleanupOldEntries() {
	now := time.Now()
	keysToDelete := make([]notificationKey, 0)

	for k, t := range r.lastNotified {
		if now.Sub(t) > r.maxAge {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, k := range keysToDelete {
		delete(r.lastNotified, k)
	}

	if len(keysToDelete) > 0 {
		slog.Debug("Cleaned up old notification entries", "count", len(keysToDelete))
	}
}
