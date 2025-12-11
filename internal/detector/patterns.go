package detector

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
)

//go:embed patterns.json
var patternsJSON []byte

type Pattern struct {
	ID          string
	Name        string
	Description string
	Regex       *regexp.Regexp
}

type patternConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Regex       string `json:"regex"`
}

type patternsFile struct {
	Patterns []patternConfig `json:"patterns"`
}

func compilePatterns(configs []patternConfig, failFast bool, source string) ([]Pattern, error) {
	patterns := make([]Pattern, 0, len(configs))
	skippedCount := 0

	for _, pc := range configs {
		regex, err := regexp.Compile(pc.Regex)
		if err != nil {
			if failFast {
				return nil, fmt.Errorf("invalid regex for pattern %q in %s: %w", pc.ID, source, err)
			}
			slog.Warn("Skipping pattern with invalid regex",
				"pattern_id", pc.ID,
				"pattern_name", pc.Name,
				"regex", pc.Regex,
				"source", source,
				"error", err)
			skippedCount++
			continue
		}

		patterns = append(patterns, Pattern{
			ID:          pc.ID,
			Name:        pc.Name,
			Description: pc.Description,
			Regex:       regex,
		})
	}

	if !failFast && skippedCount > 0 {
		slog.Warn("Some patterns were skipped due to invalid regex",
			"skipped_count", skippedCount,
			"loaded_count", len(patterns),
			"total_count", len(configs),
			"source", source)
	}

	return patterns, nil
}

func GetDefaultPatterns() []Pattern {
	var config patternsFile
	if err := json.Unmarshal(patternsJSON, &config); err != nil {
		slog.Error("Failed to parse embedded patterns.json, no patterns will be loaded",
			"error", err)
		return []Pattern{}
	}

	patterns, _ := compilePatterns(config.Patterns, false, "embedded patterns.json")

	slog.Info("Default patterns loaded",
		"pattern_count", len(patterns))

	return patterns
}

func LoadPatternsFromFile(filename string) ([]Pattern, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns file %q: %w", filename, err)
	}

	var config patternsFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse patterns JSON from %q: %w", filename, err)
	}

	if len(config.Patterns) == 0 {
		slog.Warn("Patterns file contains no patterns", "file", filename)
		return []Pattern{}, nil
	}

	patterns, err := compilePatterns(config.Patterns, true, fmt.Sprintf("file %q", filename))
	if err != nil {
		return nil, err
	}

	slog.Info("Custom patterns loaded from file",
		"file", filename,
		"pattern_count", len(patterns))

	return patterns, nil
}

func GetDefaultPatternIDs() []string {
	var config patternsFile
	if err := json.Unmarshal(patternsJSON, &config); err != nil {
		slog.Error("Failed to parse embedded patterns.json",
			"error", err)
		return []string{}
	}

	ids := make([]string, 0, len(config.Patterns))
	for _, pc := range config.Patterns {
		ids = append(ids, pc.ID)
	}

	return ids
}
