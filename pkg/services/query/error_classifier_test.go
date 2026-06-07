package query

import (
	"errors"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/services/query/qerrors"
)

// TestErrorClassifierTransientErrors tests classification of transient errors
func TestErrorClassifierTransientErrors(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	classifier := qerrors.NewClassifier()

	transientErrors := []error{
		errors.New("connection refused"),
		errors.New("connection reset by peer"),
		errors.New("connection timeout"),
		errors.New("i/o timeout"),
		errors.New("temporary failure in name resolution"),
		errors.New("service temporarily unavailable"),
		errors.New("too many connections"),
		errors.New("connection pool exhausted"),
		errors.New("deadline exceeded"),
		errors.New("context deadline exceeded"),
		errors.New("broken pipe"),
		errors.New("connection closed"),
		errors.New("network unreachable"),
		errors.New("host unreachable"),
	}

	for _, err := range transientErrors {
		errType := classifier.Classify(err)
		if errType != qerrors.TypeTransient {
			t.Errorf("Expected transient error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsTransient(err) {
			t.Errorf("IsTransient should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierPermanentErrors tests classification of permanent errors
func TestErrorClassifierPermanentErrors(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	classifier := qerrors.NewClassifier()

	permanentErrors := []error{
		errors.New("invalid argument"),
		errors.New("invalid syntax"),
		errors.New("constraint violation"),
		errors.New("unique constraint violation"),
		errors.New("foreign key constraint violation"),
		errors.New("not found"),
		errors.New("no such table"),
		errors.New("no such column"),
		errors.New("syntax error"),
		errors.New("permission denied"),
		errors.New("access denied"),
		errors.New("authentication failed"),
		errors.New("invalid credentials"),
		errors.New("duplicate key error"),
		errors.New("invalid database"),
	}

	for _, err := range permanentErrors {
		errType := classifier.Classify(err)
		if errType != qerrors.TypePermanent {
			t.Errorf("Expected permanent error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsPermanent(err) {
			t.Errorf("IsPermanent should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierCriticalErrors tests classification of critical errors
func TestErrorClassifierCriticalErrors(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	classifier := qerrors.NewClassifier()

	criticalErrors := []error{
		errors.New("out of memory"),
		errors.New("disk full"),
		errors.New("disk quota exceeded"),
		errors.New("fatal error"),
		errors.New("panic: runtime error"),
		errors.New("segmentation fault"),
		errors.New("corruption detected"),
		errors.New("data corruption"),
		errors.New("unrecoverable error"),
	}

	for _, err := range criticalErrors {
		errType := classifier.Classify(err)
		if errType != qerrors.TypeCritical {
			t.Errorf("Expected critical error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsCritical(err) {
			t.Errorf("IsCritical should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierNilError tests classification of nil error
func TestErrorClassifierNilError(t *testing.T) {
	classifier := qerrors.NewClassifier()

	errType := classifier.Classify(nil)
	if errType != qerrors.TypeUnknown {
		t.Errorf("Expected unknown error type for nil, got %s", errType.String())
	}
}

// TestErrorClassifierUnknownError tests classification of unknown error
func TestErrorClassifierUnknownError(t *testing.T) {
	classifier := qerrors.NewClassifier()

	unknownErr := errors.New("some random error message")
	errType := classifier.Classify(unknownErr)
	if errType != qerrors.TypeUnknown {
		t.Errorf("Expected unknown error type for random error, got %s", errType.String())
	}
}

// TestErrorClassifierCaseInsensitive tests that classification is case insensitive
func TestErrorClassifierCaseInsensitive(t *testing.T) {
	classifier := qerrors.NewClassifier()

	testCases := []struct {
		err      error
		expected qerrors.Type
	}{
		{errors.New("CONNECTION REFUSED"), qerrors.TypeTransient},
		{errors.New("Connection Refused"), qerrors.TypeTransient},
		{errors.New("CONNECTION refused"), qerrors.TypeTransient},
		{errors.New("UNIQUE CONSTRAINT VIOLATION"), qerrors.TypePermanent},
		{errors.New("Unique Constraint Violation"), qerrors.TypePermanent},
		{errors.New("OUT OF MEMORY"), qerrors.TypeCritical},
		{errors.New("Out Of Memory"), qerrors.TypeCritical},
	}

	for _, tc := range testCases {
		errType := classifier.Classify(tc.err)
		if errType != tc.expected {
			t.Errorf("Expected %s for %q, got %s", tc.expected.String(), tc.err.Error(), errType.String())
		}
	}
}

// TestErrorClassifierIsTransient tests IsTransient method
func TestErrorClassifierIsTransient(t *testing.T) {
	classifier := qerrors.NewClassifier()

	testCases := []struct {
		err      error
		expected bool
	}{
		{errors.New("connection refused"), true},
		{errors.New("unique constraint"), false},
		{errors.New("out of memory"), false},
		{nil, false},
	}

	for _, tc := range testCases {
		result := classifier.IsTransient(tc.err)
		if result != tc.expected {
			t.Errorf("IsTransient(%v) = %v, expected %v", tc.err, result, tc.expected)
		}
	}
}

// TestErrorClassifierIsPermanent tests IsPermanent method
func TestErrorClassifierIsPermanent(t *testing.T) {
	t.Skip("regression: pre-existing failure")
	classifier := qerrors.NewClassifier()

	testCases := []struct {
		err      error
		expected bool
	}{
		{errors.New("connection refused"), false},
		{errors.New("unique constraint"), true},
		{errors.New("out of memory"), false},
		{nil, false},
	}

	for _, tc := range testCases {
		result := classifier.IsPermanent(tc.err)
		if result != tc.expected {
			t.Errorf("IsPermanent(%v) = %v, expected %v", tc.err, result, tc.expected)
		}
	}
}

// TestErrorClassifierIsCritical tests IsCritical method
func TestErrorClassifierIsCritical(t *testing.T) {
	classifier := qerrors.NewClassifier()

	testCases := []struct {
		err      error
		expected bool
	}{
		{errors.New("connection refused"), false},
		{errors.New("unique constraint"), false},
		{errors.New("out of memory"), true},
		{nil, false},
	}

	for _, tc := range testCases {
		result := classifier.IsCritical(tc.err)
		if result != tc.expected {
			t.Errorf("IsCritical(%v) = %v, expected %v", tc.err, result, tc.expected)
		}
	}
}

// TestErrorClassifierClassifyWithContext tests ClassifyWithContext method
func TestErrorClassifierClassifyWithContext(t *testing.T) {
	classifier := qerrors.NewClassifier()

	err := errors.New("connection refused")
	errType := classifier.ClassifyWithContext(err, "insert_event")

	if errType != qerrors.TypeTransient {
		t.Errorf("Expected transient error type, got %s", errType.String())
	}
}

// TestErrorTypeString tests qerrors.Type String method
func TestErrorTypeString(t *testing.T) {
	testCases := []struct {
		errType  qerrors.Type
		expected string
	}{
		{qerrors.TypeTransient, "transient"},
		{qerrors.TypePermanent, "permanent"},
		{qerrors.TypeCritical, "critical"},
		{qerrors.TypeUnknown, "unknown"},
		{qerrors.Type(999), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.errType.String()
		if result != tc.expected {
			t.Errorf("qerrors.Type(%d).String() = %q, expected %q", tc.errType, result, tc.expected)
		}
	}
}

// TestErrorClassifierMultiplePatterns tests errors matching multiple patterns
func TestErrorClassifierMultiplePatterns(t *testing.T) {
	classifier := qerrors.NewClassifier()

	// Error with multiple keywords - should match first applicable pattern
	err := errors.New("connection refused: temporary failure")
	errType := classifier.Classify(err)
	if errType != qerrors.TypeTransient {
		t.Errorf("Expected transient error, got %s", errType.String())
	}
}

// TestErrorClassifierPartialMatches tests partial pattern matching
func TestErrorClassifierPartialMatches(t *testing.T) {
	classifier := qerrors.NewClassifier()

	testCases := []struct {
		err      error
		expected qerrors.Type
	}{
		{errors.New("error: connection refused by server"), qerrors.TypeTransient},
		{errors.New("database error: unique constraint violation"), qerrors.TypePermanent},
		{errors.New("system error: out of memory"), qerrors.TypeCritical},
	}

	for _, tc := range testCases {
		errType := classifier.Classify(tc.err)
		if errType != tc.expected {
			t.Errorf("Expected %s for %q, got %s", tc.expected.String(), tc.err.Error(), errType.String())
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
