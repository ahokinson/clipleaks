package detector

import (
	"math"
)

func CalculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	frequencies := make(map[rune]int)
	for _, char := range s {
		frequencies[char]++
	}

	var entropy float64
	length := float64(len(s))

	for _, count := range frequencies {
		frequency := float64(count) / length
		entropy -= frequency * math.Log2(frequency)
	}

	return entropy
}

func IsHighEntropy(s string, threshold float64) bool {
	return CalculateEntropy(s) >= threshold
}
