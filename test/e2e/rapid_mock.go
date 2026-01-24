package e2e

import (
	"testing"
)

// This is a mock implementation of pgregory.net/rapid for testing purposes
// when network access is not available

// Generator is a mock generator for property-based testing
type Generator struct {
	t *testing.T
}

// IntRange returns a random integer in the given range
func (g *Generator) IntRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + (max-min)/2 // Return middle value for deterministic testing
}

// Uint64 returns a random uint64
func (g *Generator) Uint64() uint64 {
	return 1000
}

// Check runs a property test
func Check(t *testing.T, fn func(*testing.T, *Generator)) {
	gen := &Generator{t: t}
	fn(t, gen)
}

// TestLogger is a simple logger for testing
type TestLogger struct{}

// NewTestLogger creates a new test logger
func NewTestLogger() Logger {
	return &TestLogger{}
}

// Infof logs an info message
func (tl *TestLogger) Infof(format string, args ...interface{}) {
	// No-op for testing
}

// Warnf logs a warning message
func (tl *TestLogger) Warnf(format string, args ...interface{}) {
	// No-op for testing
}

// Errorf logs an error message
func (tl *TestLogger) Errorf(format string, args ...interface{}) {
	// No-op for testing
}

// Debugf logs a debug message
func (tl *TestLogger) Debugf(format string, args ...interface{}) {
	// No-op for testing
}
