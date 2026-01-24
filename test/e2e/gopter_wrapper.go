package e2e

import (
	"testing"

	"github.com/leanovate/gopter"
)

// TestReporter wraps testing.T to implement gopter.Reporter
type TestReporter struct {
	t *testing.T
}

// NewTestReporter creates a new test reporter
func NewTestReporter(t *testing.T) gopter.Reporter {
	return &TestReporter{t: t}
}

// ReportTestResult implements gopter.Reporter
func (tr *TestReporter) ReportTestResult(name string, result *gopter.TestResult) {
	if !result.Passed() {
		tr.t.Logf("Property test %s failed: %s", name, result.Error)
		tr.t.Fail()
	}
}
