package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ahokinson/clipleaks/internal/util"
)

func TestDefaultConfigIsValid(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("DefaultConfig() should be valid, got: %v", err)
	}
}

func TestValidate_RejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr error
	}{
		{"entropy too high", func(c *Config) { c.EntropyThreshold = MaxEntropyThreshold + 1 }, ErrInvalidEntropyThreshold},
		{"entropy negative", func(c *Config) { c.EntropyThreshold = -1 }, ErrInvalidEntropyThreshold},
		{"poll too low", func(c *Config) { c.PollInterval = MinPollInterval - 1 }, ErrInvalidPollInterval},
		{"poll too high", func(c *Config) { c.PollInterval = MaxPollInterval + 1 }, ErrInvalidPollInterval},
		{"empty disabled pattern id", func(c *Config) { c.DisabledPatterns = []string{""} }, ErrInvalidPatternID},
		{"unknown disabled pattern", func(c *Config) { c.DisabledPatterns = []string{"does-not-exist"} }, ErrUnknownPatternID},
		{"clipboard too small", func(c *Config) { c.MaxClipboardSize = util.Pointer(MinClipboardSize - 1) }, ErrInvalidClipboardSize},
		{"clipboard too large", func(c *Config) { c.MaxClipboardSize = util.Pointer(MaxClipboardSize + 1) }, ErrInvalidClipboardSize},
		{"match size too small", func(c *Config) { c.PatternMatchSize = util.Pointer(MinPatternMatchSize - 1) }, ErrInvalidPatternMatchSize},
		{"match size too large", func(c *Config) { c.PatternMatchSize = util.Pointer(MaxPatternMatchSizeLimit + 1) }, ErrInvalidPatternMatchSize},
		{"overlap too large", func(c *Config) { c.PatternOverlapSize = util.Pointer(MaxPatternOverlapSizeLimit + 1) }, ErrInvalidPatternOverlapSize},
		{"overlap negative", func(c *Config) { c.PatternOverlapSize = util.Pointer(-1) }, ErrInvalidPatternOverlapSize},
		{"overlap >= match", func(c *Config) {
			// Both within their individual ranges, but overlap not < match.
			c.PatternMatchSize = util.Pointer(MinPatternMatchSize)
			c.PatternOverlapSize = util.Pointer(MaxPatternOverlapSizeLimit) // == MinPatternMatchSize (1024)
		}, ErrInvalidPatternOverlap},
		{"burst too low", func(c *Config) { c.DetectionBurstLimit = util.Pointer(MinDetectionBurstLimit - 1) }, ErrInvalidDetectionBurstLimit},
		{"burst too high", func(c *Config) { c.DetectionBurstLimit = util.Pointer(MaxDetectionBurstLimit + 1) }, ErrInvalidDetectionBurstLimit},
		{"rate too low", func(c *Config) { c.DetectionRateLimit = util.Pointer(MinDetectionRateLimit - 0.01) }, ErrInvalidDetectionRateLimit},
		{"rate too high", func(c *Config) { c.DetectionRateLimit = util.Pointer(MaxDetectionRateLimit + 1) }, ErrInvalidDetectionRateLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultConfig()
			tt.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidate_DefaultsNilPointers(t *testing.T) {
	c := &Config{
		EntropyThreshold: 3.5,
		PollInterval:     500,
		DisabledPatterns: []string{},
		// all *T fields intentionally nil
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() should default nil pointers and pass, got: %v", err)
	}

	for name, got := range map[string]bool{
		"MaxClipboardSize":    c.MaxClipboardSize == nil,
		"PatternMatchSize":    c.PatternMatchSize == nil,
		"PatternOverlapSize":  c.PatternOverlapSize == nil,
		"DetectionBurstLimit": c.DetectionBurstLimit == nil,
		"DetectionRateLimit":  c.DetectionRateLimit == nil,
	} {
		if got {
			t.Errorf("%s should have been defaulted to non-nil", name)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// First Load on a fresh HOME writes a default config file.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() on fresh HOME: %v", err)
	}
	if _, err := os.Stat(GetConfigPath()); err != nil {
		t.Fatalf("expected config file to be created at %s: %v", GetConfigPath(), err)
	}

	// Mutate, save, and reload to confirm values persist.
	cfg.PollInterval = 1234
	cfg.DisabledPatterns = []string{} // keep valid
	if err := cfg.Save(GetConfigPath()); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.PollInterval != 1234 {
		t.Errorf("PollInterval did not persist: got %d", reloaded.PollInterval)
	}
}

func TestLoad_RejectsInvalidFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"poll_interval": 1}`), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected Load() to reject an out-of-range poll_interval")
	}
}

func TestSave_RejectsInvalidConfig(t *testing.T) {
	c := DefaultConfig()
	c.PollInterval = -1
	if err := c.Save(filepath.Join(t.TempDir(), "config.json")); err == nil {
		t.Fatal("expected Save() to reject an invalid config")
	}
}
