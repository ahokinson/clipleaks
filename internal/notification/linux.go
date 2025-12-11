//go:build linux

package notification

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ahokinson/clipleaks/internal/reliability"
)

type linuxNotifier struct{}

func newPlatformNotifier() Notifier {
	return &linuxNotifier{}
}

func sanitizeNotificationText(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	if len(s) > 1000 {
		s = s[:1000] + "..."
	}
	return s
}

func (n *linuxNotifier) Notify(title, message string) error {
	return n.NotifyWithContext(context.Background(), title, message)
}

func (n *linuxNotifier) NotifyWithContext(ctx context.Context, title, message string) error {
	err := reliability.Retry(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		sanitizedTitle := sanitizeNotificationText(title)
		sanitizedMessage := sanitizeNotificationText(message)

		cmd := exec.CommandContext(ctx, "notify-send", sanitizedTitle, sanitizedMessage)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to send Linux notification (is notify-send installed?): %w", err)
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
