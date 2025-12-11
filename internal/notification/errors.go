package notification

import "errors"

var (
	ErrNotificationFailed      = errors.New("failed to send notification")
	ErrNotificationUnavailable = errors.New("notification system is unavailable")
	ErrRateLimited             = errors.New("notification rate limited")
)
