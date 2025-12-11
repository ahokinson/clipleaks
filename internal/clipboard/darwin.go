//go:build darwin

package clipboard

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/ahokinson/clipleaks/internal/reliability"
)

type darwinMonitor struct {
	pollInterval     time.Duration
	debounceInterval time.Duration
	maxSize          int
	channelBuffer    int
}

func newPlatformMonitor(cfg *config) Monitor {
	slog.Info("Darwin clipboard monitor initialized",
		"poll_interval", cfg.pollInterval.Milliseconds(),
		"debounce_interval", cfg.debounceInterval.Milliseconds(),
		"max_size_bytes", cfg.maxSize,
		"channel_buffer", cfg.channelBuffer)
	return &darwinMonitor{
		pollInterval:     cfg.pollInterval,
		debounceInterval: cfg.debounceInterval,
		maxSize:          cfg.maxSize,
		channelBuffer:    cfg.channelBuffer,
	}
}

func (m *darwinMonitor) Start(ctx context.Context) (<-chan Content, error) {
	contentChan := make(chan Content, m.channelBuffer)

	go reliability.WithRecover("darwin-clipboard-monitor", func() {
		runMonitorLoop(ctx, m, monitorConfig{
			maxSize:          m.maxSize,
			debounceInterval: m.debounceInterval,
			pollInterval:     m.pollInterval,
		}, contentChan)
	})

	return contentChan, nil
}

func (m *darwinMonitor) GetContent() (Content, error) {
	text, err := m.readClipboard(context.Background())
	if err != nil {
		return Content{}, err
	}

	truncated := false
	if len(text) > m.maxSize {
		text = text[:m.maxSize]
		truncated = true
	}

	return Content{
		Text:      text,
		Timestamp: time.Now(),
		Truncated: truncated,
	}, nil
}

func (m *darwinMonitor) readClipboard(ctx context.Context) (string, error) {
	var result string

	err := reliability.Retry(func() error {
		cmd := exec.CommandContext(ctx, "pbpaste")
		output, err := cmd.Output()
		if err != nil {
			slog.Debug("Failed to read clipboard", "error", err)
			return fmt.Errorf("failed to execute pbpaste: %w", err)
		}
		result = string(output)
		return nil
	}, &reliability.RetryConfig{
		MaxRetries:     ClipboardReadMaxRetries,
		InitialBackoff: ClipboardReadInitialBackoff,
		MaxBackoff:     ClipboardReadMaxBackoff,
	})

	return result, err
}
