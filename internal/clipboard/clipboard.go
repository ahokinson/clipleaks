package clipboard

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ahokinson/clipleaks/internal/security"
)

const (
	DefaultChannelBuffer        = 10                     // Buffer clipboard events to prevent blocking during processing
	MaxConsecutiveErrors        = 10                     // Tolerate temporary clipboard lock issues before resetting counter
	ErrorLogThrottle            = 10 * time.Second       // Prevent log spam during persistent failures
	ClipboardReadMaxRetries     = 2                      // 3 total attempts balances reliability vs delay
	ClipboardReadInitialBackoff = 100 * time.Millisecond // Short enough for good UX, long enough for transient issues
	ClipboardReadMaxBackoff     = 500 * time.Millisecond // Cap retry delay to avoid excessive waiting
)

var (
	ErrClipboardTooLarge    = errors.New("clipboard content exceeds maximum size")
	ErrClipboardUnavailable = errors.New("clipboard is not available")
	ErrClipboardReadFailed  = errors.New("failed to read clipboard content")
	ErrInvalidConfig        = errors.New("invalid clipboard configuration")
)

type Monitor interface {
	Start(ctx context.Context) (<-chan Content, error)
	GetContent() (Content, error)
}

type Content struct {
	Text      string
	Timestamp time.Time
	Truncated bool
}

func (c *Content) Wipe() {
	if c.Text != "" {
		textBytes := []byte(c.Text)
		security.WipeBytes(textBytes)
		c.Text = ""
	}
}

type Option func(*config)

type config struct {
	maxSize          int
	pollInterval     time.Duration
	debounceInterval time.Duration
	channelBuffer    int
}

func defaultConfig() *config {
	return &config{
		maxSize:          1024 * 1024,
		pollInterval:     500 * time.Millisecond,
		debounceInterval: 100 * time.Millisecond,
		channelBuffer:    DefaultChannelBuffer,
	}
}

func WithMaxSize(size int) Option {
	return func(c *config) {
		c.maxSize = size
	}
}

func WithPollInterval(interval time.Duration) Option {
	return func(c *config) {
		c.pollInterval = interval
	}
}

func WithDebounceInterval(interval time.Duration) Option {
	return func(c *config) {
		c.debounceInterval = interval
	}
}

func WithChannelBuffer(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.channelBuffer = size
		}
	}
}

func New(opts ...Option) Monitor {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return newPlatformMonitor(cfg)
}

type clipboardReader interface {
	readClipboard(ctx context.Context) (string, error)
}

type monitorConfig struct {
	maxSize          int
	debounceInterval time.Duration
	pollInterval     time.Duration
}

func runMonitorLoop(ctx context.Context, reader clipboardReader, cfg monitorConfig, contentChan chan Content) {
	defer close(contentChan)

	var lastContent *security.SecureString
	var lastChangeTime time.Time
	var consecutiveErrors int
	var lastErrorTime time.Time
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()
	defer func() {
		if lastContent != nil {
			lastContent.Wipe()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Clipboard monitor stopping")
			return
		case <-ticker.C:
			content, err := reader.readClipboard(ctx)
			if err != nil {
				consecutiveErrors = handleReadError(err, consecutiveErrors, &lastErrorTime)
				continue
			}
			consecutiveErrors = 0

			if shouldProcessContent(content, lastContent, lastChangeTime, cfg.debounceInterval) {
				processNewContent(content, &lastContent, &lastChangeTime, contentChan, cfg.maxSize)
			}
		}
	}
}

func handleReadError(err error, consecutiveErrors int, lastErrorTime *time.Time) int {
	consecutiveErrors++
	now := time.Now()

	if consecutiveErrors == 1 || now.Sub(*lastErrorTime) > ErrorLogThrottle {
		slog.Warn("Failed to read clipboard",
			"error", err,
			"consecutive_errors", consecutiveErrors)
		*lastErrorTime = now
	}

	if consecutiveErrors >= MaxConsecutiveErrors {
		slog.Error("Clipboard read failures exceeded threshold",
			"consecutive_errors", consecutiveErrors,
			"error", err)
		return 0
	}

	return consecutiveErrors
}

func shouldProcessContent(content string, lastContent *security.SecureString, lastChangeTime time.Time, debounceInterval time.Duration) bool {
	if content == "" {
		return false
	}

	if lastContent != nil && bytes.Equal(lastContent.Bytes(), []byte(content)) {
		return false
	}

	if !lastChangeTime.IsZero() && time.Since(lastChangeTime) < debounceInterval {
		return false
	}

	return true
}

func processNewContent(content string, lastContent **security.SecureString, lastChangeTime *time.Time, contentChan chan Content, maxSize int) {
	now := time.Now()

	if *lastContent != nil {
		(*lastContent).Wipe()
	}
	*lastContent = security.New(content)
	*lastChangeTime = now

	truncated := false
	if len(content) > maxSize {
		content = content[:maxSize]
		truncated = true
	}

	contentChan <- Content{
		Text:      content,
		Timestamp: now,
		Truncated: truncated,
	}
}
