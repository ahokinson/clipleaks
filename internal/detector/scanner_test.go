package detector

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

func TestNew_DisablesPatterns(t *testing.T) {
	all := GetDefaultPatternIDs()
	if len(all) == 0 {
		t.Skip("no default patterns to disable")
	}
	disabled := all[0]

	d := New(WithDisabledPatterns(disabled))
	for _, id := range d.GetActivePatterns() {
		if id == disabled {
			t.Fatalf("pattern %q should have been disabled", disabled)
		}
	}
	if len(d.GetActivePatterns()) != len(all)-1 {
		t.Fatalf("expected %d active patterns, got %d", len(all)-1, len(d.GetActivePatterns()))
	}
}

func TestAddPattern(t *testing.T) {
	d := New()
	before := len(d.GetActivePatterns())

	d.AddPattern(Pattern{ID: "custom", Name: "Custom", Regex: regexp.MustCompile(`zzz`)})
	if len(d.GetActivePatterns()) != before+1 {
		t.Fatalf("AddPattern should add a pattern: before=%d after=%d", before, len(d.GetActivePatterns()))
	}

	findings := d.Scan("zzz")
	found := false
	for _, f := range findings {
		if f.PatternID == "custom" {
			found = true
		}
	}
	if !found {
		t.Error("added pattern should produce findings")
	}
}

func TestAddPattern_RespectsDisabledList(t *testing.T) {
	d := New(WithDisabledPatterns("custom"))
	before := len(d.GetActivePatterns())

	d.AddPattern(Pattern{ID: "custom", Name: "Custom", Regex: regexp.MustCompile(`zzz`)})
	if len(d.GetActivePatterns()) != before {
		t.Error("AddPattern should skip patterns present in the disabled list")
	}
}

func TestScanWithContext_CancelledReturnsEarly(t *testing.T) {
	d := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before scanning

	findings := d.ScanWithContext(ctx, strings.Repeat("CANARY1234 ", 100))
	if len(findings) != 0 {
		t.Fatalf("cancelled scan should return no findings, got %d", len(findings))
	}
}

func TestScan_NoMatchReturnsEmpty(t *testing.T) {
	d := New()
	if findings := d.Scan("just some plain harmless text"); len(findings) != 0 {
		t.Fatalf("expected no findings in benign text, got %d: %+v", len(findings), findings)
	}
}

func TestScan_EmptyInput(t *testing.T) {
	d := New()
	if findings := d.Scan(""); len(findings) != 0 {
		t.Fatalf("empty input should yield no findings, got %d", len(findings))
	}
}
