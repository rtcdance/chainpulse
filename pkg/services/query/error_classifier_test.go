package query

import (
	"errors"
	"strings"
	"testing"
)

// TestErrorClassifierTransientErrors tests classification of transient errors
func TestErrorClassifierTransientErrors(t *testing.T) {
	classifier := NewErrorClassifier()

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
		errType := classifier.ClassifyError(err)
		if errType != ErrorTypeTransient {
			t.Errorf("Expected transient error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsTransient(err) {
			t.Errorf("IsTransient should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierPermanentErrors tests classification of permanent errors
func TestErrorClassifierPermanentErrors(t *testing.T) {
	classifier := NewErrorClassifier()

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
		errType := classifier.ClassifyError(err)
		if errType != ErrorTypePermanent {
			t.Errorf("Expected permanent error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsPermanent(err) {
			t.Errorf("IsPermanent should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierCriticalErrors tests classification of critical errors
func TestErrorClassifierCriticalErrors(t *testing.T) {
	classifier := NewErrorClassifier()

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
		errType := classifier.ClassifyError(err)
		if errType != ErrorTypeCritical {
			t.Errorf("Expected critical error for %q, got %s", err.Error(), errType.String())
		}

		if !classifier.IsCritical(err) {
			t.Errorf("IsCritical should return true for %q", err.Error())
		}
	}
}

// TestErrorClassifierNilError tests classification of nil error
func TestErrorClassifierNilError(t *testing.T) {
	classifier := NewErrorClassifier()

	errType := classifier.ClassifyError(nil)
	if errType != ErrorTypeUnknown {
		t.Errorf("Expected unknown error type for nil, got %s", errType.String())
	}
}

// TestErrorClassifierUnknownError tests classification of unknown error
func TestErrorClassifierUnknownError(t *testing.T) {
	classifier := NewErrorClassifier()

	unknownErr := errors.New("some random error message")
	errType := classifier.ClassifyError(unknownErr)
	if errType != ErrorTypeUnknown {
		t.Errorf("Expected unknown error type for random error, got %s", errType.String())
	}
}

// TestErrorClassifierCaseInsensitive tests that classification is case insensitive
func TestErrorClassifierCaseInsensitive(t *testing.T) {
	classifier := NewErrorClassifier()

	testCases := []struct {
		err      error
		expected ErrorType
	}{
		{errors.New("CONNECTION REFUSED"), ErrorTypeTransient},
		{errors.New("Connection Refused"), ErrorTypeTransient},
		{errors.New("CONNECTION refused"), ErrorTypeTransient},
		{errors.New("UNIQUE CONSTRAINT VIOLATION"), ErrorTypePermanent},
		{errors.New("Unique Constraint Violation"), ErrorTypePermanent},
		{errors.New("OUT OF MEMORY"), ErrorTypeCritical},
		{errors.New("Out Of Memory"), ErrorTypeCritical},
	}

	for _, tc := range testCases {
		errType := classifier.ClassifyError(tc.err)
		if errType != tc.expected {
			t.Errorf("Expected %s for %q, got %s", tc.expected.String(), tc.err.Error(), errType.String())
		}
	}
}

// TestErrorClassifierIsTransient tests IsTransient method
func TestErrorClassifierIsTransient(t *testing.T) {
	classifier := NewErrorClassifier()

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
	classifier := NewErrorClassifier()

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
	classifier := NewErrorClassifier()

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

// TestErrorClassifierClassifyErrorWithContext tests ClassifyErrorWithContext method
func TestErrorClassifierClassifyErrorWithContext(t *testing.T) {
	classifier := NewErrorClassifier()

	err := errors.New("connection refused")
	errType, context := classifier.ClassifyErrorWithContext(err, "insert_event")

	if errType != ErrorTypeTransient {
		t.Errorf("Expected transient error type, got %s", errType.String())
	}

	if !contains(context, "operation=insert_event") {
		t.Errorf("Context should contain operation, got %s", context)
	}

	if !contains(context, "error_type=transient") {
		t.Errorf("Context should contain error_type, got %s", context)
	}

	if !contains(context, "connection refused") {
		t.Errorf("Context should contain error message, got %s", context)
	}
}

// TestErrorTypeString tests ErrorType String method
func TestErrorTypeString(t *testing.T) {
	testCases := []struct {
		errType  ErrorType
		expected string
	}{
		{ErrorTypeTransient, "transient"},
		{ErrorTypePermanent, "permanent"},
		{ErrorTypeCritical, "critical"},
		{ErrorTypeUnknown, "unknown"},
		{ErrorType(999), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.errType.String()
		if result != tc.expected {
			t.Errorf("ErrorType(%d).String() = %q, expected %q", tc.errType, result, tc.expected)
		}
	}
}

// TestErrorClassifierMultiplePatterns tests errors matching multiple patterns
func TestErrorClassifierMultiplePatterns(t *testing.T) {
	classifier := NewErrorClassifier()

	// Error with multiple keywords - should match first applicable pattern
	err := errors.New("connection refused: temporary failure")
	errType := classifier.ClassifyError(err)
	if errType != ErrorTypeTransient {
		t.Errorf("Expected transient error, got %s", errType.String())
	}
}

// TestErrorClassifierPartialMatches tests partial pattern matching
func TestErrorClassifierPartialMatches(t *testing.T) {
	classifier := NewErrorClassifier()

	testCases := []struct {
		err      error
		expected ErrorType
	}{
		{errors.New("error: connection refused by server"), ErrorTypeTransient},
		{errors.New("database error: unique constraint violation"), ErrorTypePermanent},
		{errors.New("system error: out of memory"), ErrorTypeCritical},
	}

	for _, tc := range testCases {
		errType := classifier.ClassifyError(tc.err)
		if errType != tc.expected {
			t.Errorf("Expected %s for %q, got %s", tc.expected.String(), tc.err.Error(), errType.String())
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
