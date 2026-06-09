package detector

import (
	"context"
	"log/slog"
)

const (
	MaxPatternMatchSize = 10 * 1024
	PatternOverlapSize  = 200
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

	// Adjacent chunks overlap by patternOverlapSize bytes so secrets straddling a
	// chunk boundary are still matched. seenStarts is shared across all chunks so a
	// secret that falls inside an overlap region (present in two chunks) is reported
	// once, keyed on its absolute start index.
	step := d.patternMatchSize - d.patternOverlapSize
	if step < 1 {
		step = 1
	}
	seenStarts := make(map[int]bool)

	offset := 0
	for offset < len(text) {
		select {
		case <-ctx.Done():
			return findings
		default:
		}

		chunkEnd := offset + d.patternMatchSize
		if chunkEnd > len(text) {
			chunkEnd = len(text)
		}

		scanText := text[offset:chunkEnd]
		matches := pattern.Regex.FindAllStringIndex(scanText, -1)

		if len(matches) > 0 {
			slog.Debug("Pattern matched in chunk",
				"pattern_id", pattern.ID,
				"match_count", len(matches),
				"chunk_offset", offset,
				"chunk_size", len(scanText))
		}

		chunkFindings := d.processMatchesWithOffset(scanText, matches, pattern, offset, seenStarts)
		findings = append(findings, chunkFindings...)

		if chunkEnd == len(text) {
			break
		}

		offset += step
		if offset >= len(text) {
			break
		}
	}

	if len(findings) > 0 {
		slog.Debug("Pattern scan completed",
			"pattern_id", pattern.ID,
			"total_findings", len(findings))
	}

	return findings
}

func (d *detector) processMatchesWithOffset(scanText string, matches [][]int, pattern Pattern, offset int, seenStarts map[int]bool) []Finding {
	var findings []Finding

	for _, match := range matches {
		absoluteStart := offset + match[0]

		if seenStarts[absoluteStart] {
			continue
		}
		seenStarts[absoluteStart] = true

		matchedText := scanText[match[0]:match[1]]
		entropy := CalculateEntropy(matchedText)

		findings = append(findings, Finding{
			PatternID:   pattern.ID,
			PatternName: pattern.Name,
			Description: pattern.Description,
			Match:       matchedText,
			Entropy:     entropy,
			StartIndex:  absoluteStart,
			EndIndex:    offset + match[1],
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
