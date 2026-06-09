package process

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ahokinson/clipleaks/internal/config"
	"github.com/ahokinson/clipleaks/internal/detector"
)

func TestNew_BuildsProcessFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if err := cfg.Validate(); err != nil { // populates default pointer fields
		t.Fatalf("default config invalid: %v", err)
	}

	p := New(cfg, &bytes.Buffer{})
	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.detector == nil || p.notifier == nil || p.monitor == nil || p.limiter == nil {
		t.Error("New() should fully initialize detector, notifier, monitor, and limiter")
	}
	if p.errCh == nil {
		t.Error("New() should initialize the error channel")
	}
}

func TestNew_HonorsDisabledPatterns(t *testing.T) {
	ids := detector.GetDefaultPatternIDs()
	if len(ids) == 0 {
		t.Skip("no default patterns available to disable")
	}
	disabled := ids[0]

	cfg := config.DefaultConfig()
	cfg.DisabledPatterns = []string{disabled}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config invalid: %v", err)
	}

	p := New(cfg, &bytes.Buffer{})
	for _, id := range p.detector.GetActivePatterns() {
		if id == disabled {
			t.Fatalf("disabled pattern %q should not be active", disabled)
		}
	}
}

func TestFormatSecretDetectedMessage(t *testing.T) {
	msg := FormatSecretDetectedMessage("AWS Access Key")
	if !strings.Contains(msg, "AWS Access Key") {
		t.Errorf("message should name the pattern, got: %q", msg)
	}
	if !strings.Contains(msg, "clipboard") {
		t.Errorf("message should mention the clipboard, got: %q", msg)
	}
}
