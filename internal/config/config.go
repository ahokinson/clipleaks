package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahokinson/clipleaks/internal/detector"
	"github.com/ahokinson/clipleaks/internal/util"
)

const (
	MinEntropyThreshold        = 0.0
	MaxEntropyThreshold        = 8.0
	MinPollInterval            = 100
	MaxPollInterval            = 60000
	MinClipboardSize           = 1024
	MaxClipboardSize           = 10 * 1024 * 1024
	DefaultClipboardSize       = 1024 * 1024
	MinPatternMatchSize        = 1024        // 1KB minimum
	MaxPatternMatchSizeLimit   = 1024 * 1024 // 1MB maximum
	DefaultPatternMatchSize    = 10 * 1024   // 10KB default
	MinPatternOverlapSize      = 0           // 0 bytes minimum (no overlap)
	MaxPatternOverlapSizeLimit = 1024        // 1KB maximum
	DefaultPatternOverlapSize  = 200         // 200 bytes default
	MinDetectionBurstLimit     = 1
	MaxDetectionBurstLimit     = 100
	DefaultDetectionBurstLimit = 10
	MinDetectionRateLimit      = 0.1
	MaxDetectionRateLimit      = 100.0
	DefaultDetectionRateLimit  = 2.0
)

var (
	ErrInvalidEntropyThreshold    = errors.New("entropy threshold must be between 0.0 and 8.0")
	ErrInvalidPollInterval        = errors.New("poll interval must be between 100ms and 60000ms")
	ErrInvalidPatternID           = errors.New("pattern ID cannot be empty")
	ErrInvalidClipboardSize       = errors.New("max clipboard size must be between 1KB and 10MB")
	ErrUnknownPatternID           = errors.New("unknown pattern ID")
	ErrInvalidPatternMatchSize    = errors.New("pattern match size must be between 1KB and 1MB")
	ErrInvalidPatternOverlapSize  = errors.New("pattern overlap size must be between 0 and 1KB")
	ErrInvalidPatternOverlap      = errors.New("pattern overlap size must be less than pattern match size")
	ErrInvalidDetectionBurstLimit = errors.New("detection burst limit must be between 1 and 100")
	ErrInvalidDetectionRateLimit  = errors.New("detection rate limit must be between 0.1 and 100.0")
)

type Config struct {
	EnableEntropy       bool     `json:"enable_entropy"`
	EntropyThreshold    float64  `json:"entropy_threshold"`
	PollInterval        int      `json:"poll_interval"`
	DisabledPatterns    []string `json:"disabled_patterns"`
	MaxClipboardSize    *int     `json:"max_clipboard_size,omitempty"`
	PatternMatchSize    *int     `json:"pattern_match_size,omitempty"`
	PatternOverlapSize  *int     `json:"pattern_overlap_size,omitempty"`
	DetectionBurstLimit *int     `json:"detection_burst_limit,omitempty"`
	DetectionRateLimit  *float64 `json:"detection_rate_limit,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		EnableEntropy:       true,
		EntropyThreshold:    3.5,
		PollInterval:        500,
		DisabledPatterns:    []string{},
		MaxClipboardSize:    util.Pointer(DefaultClipboardSize),
		PatternMatchSize:    util.Pointer(DefaultPatternMatchSize),
		PatternOverlapSize:  util.Pointer(DefaultPatternOverlapSize),
		DetectionBurstLimit: util.Pointer(DefaultDetectionBurstLimit),
		DetectionRateLimit:  util.Pointer(DefaultDetectionRateLimit),
	}
}

func (c *Config) Validate() error {
	if c.EntropyThreshold < MinEntropyThreshold || c.EntropyThreshold > MaxEntropyThreshold {
		return fmt.Errorf("%w: got %.2f, must be between %.1f and %.1f",
			ErrInvalidEntropyThreshold, c.EntropyThreshold, MinEntropyThreshold, MaxEntropyThreshold)
	}

	if c.PollInterval < MinPollInterval || c.PollInterval > MaxPollInterval {
		return fmt.Errorf("%w: got %dms, must be between %dms and %dms",
			ErrInvalidPollInterval, c.PollInterval, MinPollInterval, MaxPollInterval)
	}

	for i, id := range c.DisabledPatterns {
		if id == "" {
			return fmt.Errorf("%w: disabled_patterns[%d] is empty", ErrInvalidPatternID, i)
		}
	}

	availablePatterns := detector.GetDefaultPatternIDs()
	available := make(map[string]bool)
	for _, id := range availablePatterns {
		available[id] = true
	}

	for _, id := range c.DisabledPatterns {
		if !available[id] {
			return fmt.Errorf("%w in disabled_patterns: %q", ErrUnknownPatternID, id)
		}
	}

	if c.MaxClipboardSize == nil {
		c.MaxClipboardSize = util.Pointer(DefaultClipboardSize)
	}

	if *c.MaxClipboardSize < MinClipboardSize || *c.MaxClipboardSize > MaxClipboardSize {
		return fmt.Errorf("%w: got %d bytes, must be between %d bytes (1KB) and %d bytes (10MB)",
			ErrInvalidClipboardSize, *c.MaxClipboardSize, MinClipboardSize, MaxClipboardSize)
	}

	if c.PatternMatchSize == nil {
		c.PatternMatchSize = util.Pointer(DefaultPatternMatchSize)
	}

	if *c.PatternMatchSize < MinPatternMatchSize || *c.PatternMatchSize > MaxPatternMatchSizeLimit {
		return fmt.Errorf("%w: got %d bytes, must be between %d bytes (1KB) and %d bytes (1MB)",
			ErrInvalidPatternMatchSize, *c.PatternMatchSize, MinPatternMatchSize, MaxPatternMatchSizeLimit)
	}

	if c.PatternOverlapSize == nil {
		c.PatternOverlapSize = util.Pointer(DefaultPatternOverlapSize)
	}

	if *c.PatternOverlapSize < MinPatternOverlapSize || *c.PatternOverlapSize > MaxPatternOverlapSizeLimit {
		return fmt.Errorf("%w: got %d bytes, must be between %d and %d bytes",
			ErrInvalidPatternOverlapSize, *c.PatternOverlapSize, MinPatternOverlapSize, MaxPatternOverlapSizeLimit)
	}

	if *c.PatternOverlapSize >= *c.PatternMatchSize {
		return fmt.Errorf("%w: overlap size (%d bytes) must be less than match size (%d bytes)",
			ErrInvalidPatternOverlap, *c.PatternOverlapSize, *c.PatternMatchSize)
	}

	if c.DetectionBurstLimit == nil {
		c.DetectionBurstLimit = util.Pointer(DefaultDetectionBurstLimit)
	}

	if *c.DetectionBurstLimit < MinDetectionBurstLimit || *c.DetectionBurstLimit > MaxDetectionBurstLimit {
		return fmt.Errorf("%w: got %d, must be between %d and %d",
			ErrInvalidDetectionBurstLimit, *c.DetectionBurstLimit, MinDetectionBurstLimit, MaxDetectionBurstLimit)
	}

	if c.DetectionRateLimit == nil {
		c.DetectionRateLimit = util.Pointer(DefaultDetectionRateLimit)
	}

	if *c.DetectionRateLimit < MinDetectionRateLimit || *c.DetectionRateLimit > MaxDetectionRateLimit {
		return fmt.Errorf("%w: got %.1f, must be between %.1f and %.1f",
			ErrInvalidDetectionRateLimit, *c.DetectionRateLimit, MinDetectionRateLimit, MaxDetectionRateLimit)
	}

	return nil
}

func Load() (*Config, error) {
	path := GetConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Create default config file
			cfg := DefaultConfig()
			if err := cfg.Save(path); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %q: %w", path, err)
	}

	return nil
}

func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "clipleaks", "config.json")
}
