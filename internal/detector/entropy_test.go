package detector

import (
	"math"
	"testing"
)

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want float64
	}{
		{"empty", "", 0.0},
		{"single char", "a", 0.0},
		{"all same", "aaaaaa", 0.0},
		{"two equal symbols", "ab", 1.0},
		{"four equal symbols", "abcd", 2.0},
		{"balanced binary", "aabb", 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEntropy(tt.in)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("CalculateEntropy(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCalculateEntropy_RandomHigherThanRepeated(t *testing.T) {
	low := CalculateEntropy("aaaaaaaaaaaaaaaa")
	high := CalculateEntropy("aB3$xZ9!qW2#mK7&")
	if high <= low {
		t.Errorf("random string entropy (%v) should exceed repeated string entropy (%v)", high, low)
	}
}

func TestIsHighEntropy(t *testing.T) {
	// "abcd" has entropy 2.0.
	if !IsHighEntropy("abcd", 2.0) {
		t.Error("entropy 2.0 should satisfy threshold 2.0 (>=)")
	}
	if IsHighEntropy("abcd", 2.5) {
		t.Error("entropy 2.0 should not satisfy threshold 2.5")
	}
	if IsHighEntropy("aaaa", 0.5) {
		t.Error("zero-entropy string should not satisfy threshold 0.5")
	}
}
