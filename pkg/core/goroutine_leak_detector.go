package core

import (
	"runtime"
	"testing"
	"time"
)

// GoroutineLeakDetector provides utilities for detecting goroutine leaks in tests
type GoroutineLeakDetector struct {
	initialCount int
	finalCount   int
	leaked       int
}

// CountGoroutines returns the current number of goroutines
func CountGoroutines() int {
	return runtime.NumGoroutine()
}

// NewGoroutineLeakDetector creates a new goroutine leak detector
func NewGoroutineLeakDetector() *GoroutineLeakDetector {
	return &GoroutineLeakDetector{
		initialCount: CountGoroutines(),
	}
}

// Finish checks for goroutine leaks and returns the count of leaked goroutines
func (d *GoroutineLeakDetector) Finish() int {
	// Allow time for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	d.finalCount = CountGoroutines()
	d.leaked = d.finalCount - d.initialCount

	return d.leaked
}

// AssertNoLeaks asserts that no goroutines were leaked
func (d *GoroutineLeakDetector) AssertNoLeaks(t *testing.T) {
	leaked := d.Finish()
	if leaked > 0 {
		t.Errorf("goroutine leak detected: started with %d, ended with %d, leaked %d",
			d.initialCount, d.finalCount, leaked)
	}
}

// GetLeakCount returns the number of leaked goroutines
func (d *GoroutineLeakDetector) GetLeakCount() int {
	return d.leaked
}

// GetInitialCount returns the initial goroutine count
func (d *GoroutineLeakDetector) GetInitialCount() int {
	return d.initialCount
}

// GetFinalCount returns the final goroutine count
func (d *GoroutineLeakDetector) GetFinalCount() int {
	return d.finalCount
}

// AssertNoGoroutineLeaks is a helper function that asserts no goroutines were leaked
// It should be called at the end of a test
func AssertNoGoroutineLeaks(t *testing.T, initialCount int) {
	// Allow time for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	finalCount := CountGoroutines()
	leaked := finalCount - initialCount

	if leaked > 0 {
		t.Errorf("goroutine leak detected: started with %d, ended with %d, leaked %d",
			initialCount, finalCount, leaked)
	}
}

// WithGoroutineLeakDetection wraps a test function with goroutine leak detection
func WithGoroutineLeakDetection(t *testing.T, testFunc func()) {
	detector := NewGoroutineLeakDetector()
	testFunc()
	detector.AssertNoLeaks(t)
}

// WithGoroutineLeakDetectionAndTimeout wraps a test function with goroutine leak detection and timeout
func WithGoroutineLeakDetectionAndTimeout(t *testing.T, timeout time.Duration, testFunc func()) {
	detector := NewGoroutineLeakDetector()

	// Run test with timeout
	done := make(chan bool, 1)
	go func() {
		testFunc()
		done <- true
	}()

	select {
	case <-done:
		// Test completed
		detector.AssertNoLeaks(t)
	case <-time.After(timeout):
		t.Errorf("test timeout after %v", timeout)
	}
}

// GoroutineLeakReport provides a detailed report of goroutine leaks
type GoroutineLeakReport struct {
	InitialCount int
	FinalCount   int
	LeakedCount  int
	LeakPercent  float64
}

// GenerateReport generates a detailed report of goroutine leaks
func (d *GoroutineLeakDetector) GenerateReport() GoroutineLeakReport {
	leakPercent := 0.0
	if d.initialCount > 0 {
		leakPercent = (float64(d.leaked) / float64(d.initialCount)) * 100
	}

	return GoroutineLeakReport{
		InitialCount: d.initialCount,
		FinalCount:   d.finalCount,
		LeakedCount:  d.leaked,
		LeakPercent:  leakPercent,
	}
}
