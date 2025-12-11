package detector

import "errors"

var (
	ErrNoPatterns         = errors.New("no patterns available for detection")
	ErrPatternTimeout     = errors.New("pattern matching timed out")
	ErrPatternPanic       = errors.New("pattern matching panicked")
	ErrInvalidPattern     = errors.New("invalid pattern configuration")
	ErrPatternNotFound    = errors.New("pattern not found")
	ErrPatternCompileFail = errors.New("failed to compile pattern regex")
)
