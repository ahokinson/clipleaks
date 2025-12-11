//go:build darwin

package notification

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ahokinson/clipleaks/internal/reliability"
)

type darwinNotifier struct{}

func newPlatformNotifier() Notifier {
	return &darwinNotifier{}
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (n *darwinNotifier) Notify(title, message string) error {
	return n.NotifyWithContext(context.Background(), title, message)
}

func (n *darwinNotifier) NotifyWithContext(ctx context.Context, title, message string) error {
	err := reliability.Retry(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		escapedTitle := escapeAppleScript(title)
		escapedMessage := escapeAppleScript(message)

		script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapedMessage, escapedTitle)
		cmd := exec.CommandContext(ctx, "osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to send macOS notification: %w", err)
		}
		return nil
	}, &reliability.RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	})

	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotificationFailed, err)
	}
	return nil
}
