package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDefaultPatterns_AllCompileAndAreNonEmpty(t *testing.T) {
	patterns := GetDefaultPatterns()
	if len(patterns) == 0 {
		t.Fatal("expected embedded patterns.json to yield at least one pattern")
	}
	for _, p := range patterns {
		if p.ID == "" {
			t.Errorf("pattern has empty ID: %+v", p)
		}
		if p.Regex == nil {
			t.Errorf("pattern %q has nil compiled regex", p.ID)
		}
	}
}

func TestGetDefaultPatternIDs_MatchesPatternCount(t *testing.T) {
	ids := GetDefaultPatternIDs()
	patterns := GetDefaultPatterns()
	if len(ids) != len(patterns) {
		t.Fatalf("ID count (%d) should match pattern count (%d)", len(ids), len(patterns))
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		if id == "" {
			t.Error("found empty pattern ID")
		}
		if seen[id] {
			t.Errorf("duplicate pattern ID: %q", id)
		}
		seen[id] = true
	}
}

func TestLoadPatternsFromFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(dir, "valid.json")
		writeFile(t, path, `{"patterns":[{"id":"x","name":"X","description":"d","regex":"foo[0-9]+"}]}`)
		patterns, err := LoadPatternsFromFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(patterns) != 1 || patterns[0].ID != "x" {
			t.Fatalf("unexpected patterns: %+v", patterns)
		}
		if !patterns[0].Regex.MatchString("foo123") {
			t.Error("compiled regex should match foo123")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadPatternsFromFile(filepath.Join(dir, "nope.json")); err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		writeFile(t, path, `{not json`)
		if _, err := LoadPatternsFromFile(path); err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("invalid regex fails fast", func(t *testing.T) {
		path := filepath.Join(dir, "badregex.json")
		writeFile(t, path, `{"patterns":[{"id":"x","name":"X","description":"d","regex":"("}]}`)
		if _, err := LoadPatternsFromFile(path); err == nil {
			t.Fatal("expected error for uncompilable regex (fail-fast mode)")
		}
	})

	t.Run("empty patterns", func(t *testing.T) {
		path := filepath.Join(dir, "empty.json")
		writeFile(t, path, `{"patterns":[]}`)
		patterns, err := LoadPatternsFromFile(path)
		if err != nil {
			t.Fatalf("empty patterns list should not error: %v", err)
		}
		if len(patterns) != 0 {
			t.Fatalf("expected no patterns, got %d", len(patterns))
		}
	})
}

func TestCompilePatterns_SkipsInvalidWhenNotFailFast(t *testing.T) {
	configs := []patternConfig{
		{ID: "good", Name: "G", Regex: "abc"},
		{ID: "bad", Name: "B", Regex: "("},
	}
	patterns, err := compilePatterns(configs, false, "test")
	if err != nil {
		t.Fatalf("non-fail-fast mode should not return an error, got: %v", err)
	}
	if len(patterns) != 1 || patterns[0].ID != "good" {
		t.Fatalf("expected only the good pattern to survive, got: %+v", patterns)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
