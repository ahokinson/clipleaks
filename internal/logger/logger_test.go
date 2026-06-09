package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactSecret(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"short fully masked", "secret8x", "***"},
		{"exactly 8 masked", "12345678", "***"},
		{"long shows ends", "AKIAIOSFODNN7EXAMPLE", "AKIA***[12]***MPLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactSecret(tt.in); got != tt.want {
				t.Errorf("RedactSecret(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRedactSecret_NeverLeaksMiddle(t *testing.T) {
	secret := "AKIAIOSFODNN7EXAMPLE"
	out := RedactSecret(secret)
	if strings.Contains(out, "IOSFODNN7") {
		t.Errorf("redacted output leaks the middle of the secret: %q", out)
	}
}

func TestRedactSecretWithLength(t *testing.T) {
	if got := RedactSecretWithLength(""); got != "" {
		t.Errorf("empty: got %q", got)
	}
	if got := RedactSecretWithLength("short"); !strings.Contains(got, "5") {
		t.Errorf("short secret should report length, got %q", got)
	}
	long := RedactSecretWithLength("AKIAIOSFODNN7EXAMPLE")
	if !strings.HasPrefix(long, "AKIA") || !strings.HasSuffix(long, "MPLE") {
		t.Errorf("long secret should show ends, got %q", long)
	}
}

func TestIsSensitiveKey(t *testing.T) {
	for _, k := range []string{"secret", "password", "token", "match", "api_key"} {
		if !IsSensitiveKey(k) {
			t.Errorf("%q should be considered sensitive", k)
		}
	}
	for _, k := range []string{"pattern_id", "count", "level"} {
		if IsSensitiveKey(k) {
			t.Errorf("%q should not be considered sensitive", k)
		}
	}
}

func TestSetup_RedactsSensitiveAttributes(t *testing.T) {
	var buf bytes.Buffer
	Setup(&buf, slog.LevelInfo, false)

	secret := "AKIAIOSFODNN7EXAMPLE"
	slog.Info("finding", "match", secret, "pattern_id", "aws-key")

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Errorf("log output should not contain the raw secret: %q", out)
	}
	if !strings.Contains(out, "aws-key") {
		t.Errorf("non-sensitive attributes should pass through: %q", out)
	}
}
