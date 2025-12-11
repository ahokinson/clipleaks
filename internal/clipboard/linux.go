//go:build linux

package clipboard

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/ahokinson/clipleaks/internal/reliability"
)

type linuxMonitor struct {
	pollInterval     time.Duration
	debounceInterval time.Duration
	maxSize          int
	channelBuffer    int
	useXclip         bool
}

func newPlatformMonitor(cfg *config) Monitor {
	_, err := exec.LookPath("xclip")
	useXclip := err == nil

	clipboardTool := "xsel"
	if useXclip {
		clipboardTool = "xclip"
	}

	slog.Info("Linux clipboard monitor initialized",
		"clipboard_tool", clipboardTool,
		"poll_interval", cfg.pollInterval.Milliseconds(),
		"debounce_interval", cfg.debounceInterval.Milliseconds(),
		"max_size_bytes", cfg.maxSize,
		"channel_buffer", cfg.channelBuffer)

	return &linuxMonitor{
		pollInterval:     cfg.pollInterval,
		debounceInterval: cfg.debounceInterval,
		maxSize:          cfg.maxSize,
		channelBuffer:    cfg.channelBuffer,
		useXclip:         useXclip,
	}
}

func (m *linuxMonitor) Start(ctx context.Context) (<-chan Content, error) {
	contentChan := make(chan Content, m.channelBuffer)

	go reliability.WithRecover("linux-clipboard-monitor", func() {
		runMonitorLoop(ctx, m, monitorConfig{
			maxSize:          m.maxSize,
			debounceInterval: m.debounceInterval,
			pollInterval:     m.pollInterval,
		}, contentChan)
	})

	return contentChan, nil
}

func (m *linuxMonitor) GetContent() (Content, error) {
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

func (m *linuxMonitor) readClipboard(ctx context.Context) (string, error) {
	var result string
	var tool string

	if m.useXclip {
		tool = "xclip"
	} else {
		tool = "xsel"
	}

	err := reliability.Retry(func() error {
		var cmd *exec.Cmd

		if m.useXclip {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
		} else {
			cmd = exec.CommandContext(ctx, "xsel", "--clipboard", "--output")
		}

		output, err := cmd.Output()
		if err != nil {
			slog.Debug("Failed to read clipboard", "error", err, "tool", tool)
			return fmt.Errorf("failed to execute %s: %w", tool, err)
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
