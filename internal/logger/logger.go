package logger

import (
	"fmt"
	"io"
	"log/slog"
)

var sensitiveKeys = map[string]bool{
	"secret":         true,
	"password":       true,
	"token":          true,
	"match":          true,
	"matched_text":   true,
	"matched_secret": true,
	"api_key":        true,
	"access_key":     true,
	"private_key":    true,
	"clipboard_text": true,
	"content":        true,
}

func Setup(writer io.Writer, level slog.Level, useJSON bool) {
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactSensitiveAttr,
	}

	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func IsSensitiveKey(key string) bool {
	return sensitiveKeys[key]
}

func redactSensitiveAttr(groups []string, a slog.Attr) slog.Attr {
	if sensitiveKeys[a.Key] {
		return slog.String(a.Key, RedactSecret(a.Value.String()))
	}
	return a
}

func RedactSecret(secret string) string {
	if len(secret) == 0 {
		return ""
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***[" + fmt.Sprintf("%d", len(secret)-8) + "]***" + secret[len(secret)-4:]
}

func RedactSecretWithLength(secret string) string {
	if len(secret) == 0 {
		return ""
	}
	if len(secret) <= 8 {
		return fmt.Sprintf("***[%d chars]", len(secret))
	}
	return fmt.Sprintf("%s***[%d chars]***%s", secret[:4], len(secret), secret[len(secret)-4:])
}
