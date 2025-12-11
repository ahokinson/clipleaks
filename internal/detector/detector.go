package detector

import (
	"context"
	"log/slog"
)

const (
	MaxPatternMatchSize = 10 * 1024 // 10KB - prevents ReDoS while allowing any reasonable secret
	PatternOverlapSize  = 200       // Extra bytes to scan beyond limit to catch patterns at boundary
)

type Finding struct {
	PatternID   string
	PatternName string
	Description string
	Match       string
	Entropy     float64
	StartIndex  int
	EndIndex    int
}

type Scanner interface {
	Scan(text string) []Finding
	ScanWithContext(ctx context.Context, text string) []Finding
	AddPattern(pattern Pattern)
	GetActivePatterns() []string
}

type Option func(*config)

type config struct {
	disabledPatterns   []string
	patternMatchSize   int
	patternOverlapSize int
}

type detector struct {
	patterns           []Pattern
	disabledPatterns   map[string]bool
	patternMatchSize   int
	patternOverlapSize int
}

func WithDisabledPatterns(patterns ...string) Option {
	return func(c *config) {
		c.disabledPatterns = patterns
	}
}

func WithPatternMatchSize(size int) Option {
	return func(c *config) {
		c.patternMatchSize = size
	}
}

func WithPatternOverlapSize(size int) Option {
	return func(c *config) {
		c.patternOverlapSize = size
	}
}

func New(opts ...Option) Scanner {
	cfg := &config{
		patternMatchSize:   MaxPatternMatchSize,
		patternOverlapSize: PatternOverlapSize,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	disabledPatterns := cfg.disabledPatterns
	allPatterns := GetDefaultPatterns()

	disabled := make(map[string]bool)
	for _, id := range disabledPatterns {
		disabled[id] = true
	}

	var activePatterns []Pattern
	for _, pattern := range allPatterns {
		if !disabled[pattern.ID] {
			activePatterns = append(activePatterns, pattern)
		}
	}

	slog.Info("Detector initialized",
		"total_patterns", len(allPatterns),
		"active_patterns", len(activePatterns),
		"disabled_count", len(disabled),
		"pattern_match_size", cfg.patternMatchSize,
		"pattern_overlap_size", cfg.patternOverlapSize)

	return &detector{
		patterns:           activePatterns,
		disabledPatterns:   disabled,
		patternMatchSize:   cfg.patternMatchSize,
		patternOverlapSize: cfg.patternOverlapSize,
	}
}

func (d *detector) Scan(text string) []Finding {
	return d.ScanWithContext(context.Background(), text)
}

func (d *detector) ScanWithContext(ctx context.Context, text string) []Finding {
	var findings []Finding

	slog.Debug("Scanning text", "text_length", len(text), "pattern_count", len(d.patterns))

	for _, pattern := range d.patterns {
		select {
		case <-ctx.Done():
			slog.Warn("Scan cancelled by context", "patterns_scanned", len(findings))
			return findings
		default:
		}

		patternFindings := d.scanPatternWithContext(ctx, text, pattern)
		findings = append(findings, patternFindings...)
	}

	if len(findings) > 0 {
		slog.Info("Scan completed", "findings_count", len(findings))
	}

	return findings
}

func (d *detector) scanPatternWithContext(ctx context.Context, text string, pattern Pattern) (findings []Finding) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Pattern matching panicked, skipping pattern",
				"pattern_id", pattern.ID,
				"pattern_name", pattern.Name,
				"panic", r)
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	scanText := d.prepareScanText(text, pattern.ID)
	matches := pattern.Regex.FindAllStringIndex(scanText, -1)

	if len(matches) > 0 {
		slog.Debug("Pattern matched",
			"pattern_id", pattern.ID,
			"match_count", len(matches))
	}

	return d.processMatches(scanText, matches, pattern)
}

func (d *detector) prepareScanText(text string, patternID string) string {
	if len(text) <= d.patternMatchSize {
		return text
	}

	scanLimit := d.patternMatchSize + d.patternOverlapSize
	if scanLimit > len(text) {
		scanLimit = len(text)
	}

	slog.Debug("Text limited for pattern matching",
		"pattern_id", patternID,
		"original_size", len(text),
		"scan_size", scanLimit,
		"overlap_bytes", d.patternOverlapSize)

	return text[:scanLimit]
}

func (d *detector) processMatches(scanText string, matches [][]int, pattern Pattern) []Finding {
	var findings []Finding

	for _, match := range matches {
		if match[0] >= d.patternMatchSize {
			slog.Debug("Skipping match in overlap region",
				"pattern_id", pattern.ID,
				"match_start", match[0],
				"pattern_match_size", d.patternMatchSize)
			continue
		}

		matchedText := scanText[match[0]:match[1]]
		entropy := CalculateEntropy(matchedText)

		// TODO: Entropy threshold filtering is not yet implemented.
		// The entropy value is calculated and logged, but not used to filter findings.
		// This feature requires better tuning and understanding of appropriate thresholds
		// for different pattern types before it can be reliably used to reduce false positives.
		// See config.EntropyThreshold and config.EnableEntropy for the intended configuration.

		findings = append(findings, Finding{
			PatternID:   pattern.ID,
			PatternName: pattern.Name,
			Description: pattern.Description,
			Match:       matchedText,
			Entropy:     entropy,
			StartIndex:  match[0],
			EndIndex:    match[1],
		})
	}

	return findings
}

func (d *detector) AddPattern(pattern Pattern) {
	if d.disabledPatterns[pattern.ID] {
		slog.Debug("Skipping pattern (in disabled list)", "pattern_id", pattern.ID)
		return
	}

	d.patterns = append(d.patterns, pattern)
	slog.Info("Custom pattern added", "pattern_id", pattern.ID, "pattern_name", pattern.Name)
}

func (d *detector) GetActivePatterns() []string {
	ids := make([]string, 0, len(d.patterns))
	for _, p := range d.patterns {
		ids = append(ids, p.ID)
	}
	return ids
}
