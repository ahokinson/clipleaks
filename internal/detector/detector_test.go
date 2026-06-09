package detector

import (
	"regexp"
	"strings"
	"testing"
)

const testPatternID = "test-canary"

// newCanaryDetector returns a detector whose only relevant pattern matches
// "CANARY" followed by four digits, using small chunk/overlap sizes so test
// inputs can exercise multi-chunk scanning without large strings.
func newCanaryDetector(t *testing.T, matchSize, overlapSize int) Scanner {
	t.Helper()
	d := New(WithPatternMatchSize(matchSize), WithPatternOverlapSize(overlapSize))
	d.AddPattern(Pattern{
		ID:          testPatternID,
		Name:        "Test Canary",
		Description: "synthetic test pattern",
		Regex:       regexp.MustCompile(`CANARY[0-9]{4}`),
	})
	return d
}

// canaryFindings filters out any incidental matches from the bundled default
// patterns so tests assert only on the synthetic pattern.
func canaryFindings(findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.PatternID == testPatternID {
			out = append(out, f)
		}
	}
	return out
}

// A secret that lands entirely inside a chunk-overlap region is contained in two
// adjacent chunks; it must be reported exactly once. Regression for the
// duplicate-findings bug in the chunked scanner.
func TestScan_OverlapRegionReportedOnce(t *testing.T) {
	matchSize, overlapSize := 100, 20
	d := newCanaryDetector(t, matchSize, overlapSize)

	// step = 80, so chunk0=[0,100), chunk1=[80,180). The overlap region is
	// [80,100). Place the token at index 85 so it sits fully inside the overlap.
	const token = "CANARY1234"
	start := 85
	text := strings.Repeat("a", start) + token + strings.Repeat("b", 150)

	findings := canaryFindings(d.Scan(text))
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding for overlap-region secret, got %d: %+v", len(findings), findings)
	}
	if findings[0].StartIndex != start || findings[0].EndIndex != start+len(token) {
		t.Errorf("expected absolute indices [%d,%d), got [%d,%d)",
			start, start+len(token), findings[0].StartIndex, findings[0].EndIndex)
	}
	if findings[0].Match != token {
		t.Errorf("expected match %q, got %q", token, findings[0].Match)
	}
}

// A secret straddling a chunk boundary (truncated in the first chunk) must still
// be detected via the overlap in the next chunk — exactly once.
func TestScan_StraddlingBoundaryDetectedOnce(t *testing.T) {
	matchSize, overlapSize := 100, 20
	d := newCanaryDetector(t, matchSize, overlapSize)

	const token = "CANARY1234" // length 10
	start := 95                // [95,105) crosses the chunk0 boundary at 100
	text := strings.Repeat("a", start) + token + strings.Repeat("b", 150)

	findings := canaryFindings(d.Scan(text))
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding for boundary-straddling secret, got %d: %+v", len(findings), findings)
	}
	if findings[0].StartIndex != start {
		t.Errorf("expected absolute start %d, got %d", start, findings[0].StartIndex)
	}
}

// The same secret literal at two distinct positions should yield two findings:
// dedup keys on absolute position, not on the matched value.
func TestScan_RepeatedLiteralAtDistinctPositions(t *testing.T) {
	matchSize, overlapSize := 100, 20
	d := newCanaryDetector(t, matchSize, overlapSize)

	const token = "CANARY1234"
	text := strings.Repeat("a", 10) + token + strings.Repeat("a", 180) + token + strings.Repeat("a", 50)
	first := 10
	second := 10 + len(token) + 180

	findings := canaryFindings(d.Scan(text))
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings for the same literal at distinct positions, got %d: %+v", len(findings), findings)
	}

	starts := map[int]bool{}
	for _, f := range findings {
		starts[f.StartIndex] = true
	}
	if !starts[first] || !starts[second] {
		t.Errorf("expected findings at absolute starts %d and %d, got %+v", first, second, findings)
	}
}

// A secret within a single chunk (no overlap involved) is detected once with
// correct absolute offsets.
func TestScan_SingleChunkBasic(t *testing.T) {
	d := newCanaryDetector(t, 1024, 200)

	const token = "CANARY9999"
	start := 42
	text := strings.Repeat("x", start) + token

	findings := canaryFindings(d.Scan(text))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].StartIndex != start || findings[0].EndIndex != start+len(token) {
		t.Errorf("expected indices [%d,%d), got [%d,%d)",
			start, start+len(token), findings[0].StartIndex, findings[0].EndIndex)
	}
}
