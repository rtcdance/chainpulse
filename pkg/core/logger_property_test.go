package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestProperty23_CorrelationIDPropagation tests that correlation IDs are properly propagated
func TestProperty23_CorrelationIDPropagation(t *testing.T) {
	correlationIDs := []string{"trace-1", "trace-2", "trace-3", "request-abc", ""}

	for _, corrID := range correlationIDs {
		buf := &bytes.Buffer{}
		logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

		if corrID != "" {
			logger = logger.WithCorrelationID(corrID).(*DefaultLogger)
		}

		logger.Info("test message")

		output := buf.String()
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}

		if corrID == "" {
			if entry.CorrelationID != "" {
				t.Errorf("expected empty correlation ID, got %s", entry.CorrelationID)
			}
		} else {
			if entry.CorrelationID != corrID {
				t.Errorf("expected correlation ID %s, got %s", corrID, entry.CorrelationID)
			}
		}
	}
}

// TestProperty23_StructuredLogFormat tests that logs are properly structured
func TestProperty23_StructuredLogFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.WithCorrelationID("test-id").(*DefaultLogger).Info("test message", "key", "value")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.Level == "" {
		t.Error("expected level field")
	}
	if entry.Message == "" {
		t.Error("expected message field")
	}
	if entry.CorrelationID == "" {
		t.Error("expected correlation ID field")
	}
}

// TestProperty23_MultipleLogLevels tests that all log levels include correlation IDs
func TestProperty23_MultipleLogLevels(t *testing.T) {
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}

	for _, level := range levels {
		buf := &bytes.Buffer{}
		logger := NewDefaultLoggerWithOutput(LogLevelDebug, buf)
		logger = logger.WithCorrelationID("test-id").(*DefaultLogger)

		switch level {
		case LogLevelDebug:
			logger.Debug("debug message")
		case LogLevelInfo:
			logger.Info("info message")
		case LogLevelWarn:
			logger.Warn("warn message")
		case LogLevelError:
			logger.Error("error message")
		}

		output := buf.String()
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}

		if entry.CorrelationID != "test-id" {
			t.Errorf("expected correlation ID for level %v", level)
		}
	}
}

// TestProperty23_FieldPreservation tests that fields are preserved in logs
func TestProperty23_FieldPreservation(t *testing.T) {
	testCases := []struct {
		name   string
		fields map[string]any
	}{
		{
			name: "string fields",
			fields: map[string]any{
				"user_id": "user-123",
				"action":  "login",
			},
		},
		{
			name: "numeric fields",
			fields: map[string]any{
				"count":    42,
				"duration": 1.5,
			},
		},
		{
			name: "mixed fields",
			fields: map[string]any{
				"user_id":  "user-123",
				"count":    42,
				"success":  true,
				"duration": 1.5,
			},
		},
	}

	for _, tc := range testCases {
		buf := &bytes.Buffer{}
		logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)
		logger = logger.WithFields(tc.fields)
		logger.Info("test message")

		output := buf.String()
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry for %s: %v", tc.name, err)
		}

		for key, expectedValue := range tc.fields {
			actualValue, exists := entry.Fields[key]
			if !exists {
				t.Errorf("field %s not found in log entry for %s", key, tc.name)
				continue
			}

			// Handle numeric comparison (JSON unmarshals numbers as float64)
			switch exp := expectedValue.(type) {
			case float64:
				if actualVal, ok := actualValue.(float64); !ok || actualVal != exp {
					t.Errorf("field %s mismatch for %s: expected %v, got %v", key, tc.name, expectedValue, actualValue)
				}
			case int:
				// JSON unmarshals integers as float64
				if actualVal, ok := actualValue.(float64); !ok || actualVal != float64(exp) {
					t.Errorf("field %s mismatch for %s: expected %v, got %v", key, tc.name, float64(exp), actualValue)
				}
			case bool:
				if actualVal, ok := actualValue.(bool); !ok || actualVal != exp {
					t.Errorf("field %s mismatch for %s: expected %v, got %v", key, tc.name, expectedValue, actualValue)
				}
			case string:
				if actualVal, ok := actualValue.(string); !ok || actualVal != exp {
					t.Errorf("field %s mismatch for %s: expected %v, got %v", key, tc.name, expectedValue, actualValue)
				}
			default:
				if expectedValue != actualValue {
					t.Errorf("field %s mismatch for %s: expected %v, got %v", key, tc.name, expectedValue, actualValue)
				}
			}
		}
	}
}

// TestProperty23_LevelConsistency tests that log levels are consistent
func TestProperty23_LevelConsistency(t *testing.T) {
	testCases := []struct {
		method   string
		expected string
		logFunc  func(*DefaultLogger)
	}{
		{"Debug", "DEBUG", func(l *DefaultLogger) { l.Debug("test message") }},
		{"Info", "INFO", func(l *DefaultLogger) { l.Info("test message") }},
		{"Warn", "WARN", func(l *DefaultLogger) { l.Warn("test message") }},
		{"Error", "ERROR", func(l *DefaultLogger) { l.Error("test message") }},
	}

	for _, tc := range testCases {
		buf := &bytes.Buffer{}
		logger := NewDefaultLoggerWithOutput(LogLevelDebug, buf)
		logger = logger.WithCorrelationID("test-id").(*DefaultLogger)
		tc.logFunc(logger)

		output := buf.String()
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v", err)
		}

		if entry.Level != tc.expected {
			t.Errorf("expected level %s, got %s", tc.expected, entry.Level)
		}
	}
}

// TestProperty23_CorrelationIDIsolation tests that correlation IDs don't leak between loggers
func TestProperty23_CorrelationIDIsolation(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger1 := NewDefaultLoggerWithOutput(LogLevelInfo, buf1)
	logger2 := NewDefaultLoggerWithOutput(LogLevelInfo, buf2)

	logger1.WithCorrelationID("id-1").(*DefaultLogger).Info("message 1")
	logger2.WithCorrelationID("id-2").(*DefaultLogger).Info("message 2")

	var entry1, entry2 LogEntry
	if err := json.Unmarshal(buf1.Bytes(), &entry1); err != nil {
		t.Fatalf("failed to unmarshal entry1: %v", err)
	}
	if err := json.Unmarshal(buf2.Bytes(), &entry2); err != nil {
		t.Fatalf("failed to unmarshal entry2: %v", err)
	}

	if entry1.CorrelationID != "id-1" {
		t.Errorf("logger1 correlation ID corrupted: expected id-1, got %s", entry1.CorrelationID)
	}
	if entry2.CorrelationID != "id-2" {
		t.Errorf("logger2 correlation ID corrupted: expected id-2, got %s", entry2.CorrelationID)
	}
}

// TestProperty23_FieldIsolation tests that fields don't leak between loggers
func TestProperty23_FieldIsolation(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger1 := NewDefaultLoggerWithOutput(LogLevelInfo, buf1)
	logger2 := NewDefaultLoggerWithOutput(LogLevelInfo, buf2)

	logger1.WithField("key", "value1").Info("message 1")
	logger2.WithField("key", "value2").Info("message 2")

	var entry1, entry2 LogEntry
	if err := json.Unmarshal(buf1.Bytes(), &entry1); err != nil {
		t.Fatalf("failed to unmarshal entry1: %v", err)
	}
	if err := json.Unmarshal(buf2.Bytes(), &entry2); err != nil {
		t.Fatalf("failed to unmarshal entry2: %v", err)
	}

	if entry1.Fields["key"] != "value1" {
		t.Errorf("logger1 field corrupted: expected value1, got %v", entry1.Fields["key"])
	}
	if entry2.Fields["key"] != "value2" {
		t.Errorf("logger2 field corrupted: expected value2, got %v", entry2.Fields["key"])
	}
}

// TestProperty23_LargeFieldValues tests that large field values are handled correctly
func TestProperty23_LargeFieldValues(t *testing.T) {
	largeString := strings.Repeat("x", 10000)

	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)
	logger.WithCorrelationID("test-id").(*DefaultLogger).WithField("large_field", largeString).Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["large_field"] != largeString {
		t.Error("large field value was truncated or corrupted")
	}
}

// TestProperty23_SpecialCharactersInFields tests that special characters in fields are handled
func TestProperty23_SpecialCharactersInFields(t *testing.T) {
	specialChars := map[string]string{
		"quotes":    "\"quoted\"",
		"backslash": "back\\slash",
		"newline":   "line1\nline2",
		"tab":       "col1\tcol2",
		"unicode":   "你好世界",
	}

	specialCharsInterface := make(map[string]any)
	for k, v := range specialChars {
		specialCharsInterface[k] = v
	}

	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)
	logger.WithCorrelationID("test-id").(*DefaultLogger).WithFields(specialCharsInterface).Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	for key, expectedValue := range specialChars {
		actualValue, exists := entry.Fields[key]
		if !exists {
			t.Errorf("field %s not found", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("field %s corrupted: expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

// TestProperty23_EmptyCorrelationID tests that empty correlation IDs are handled
func TestProperty23_EmptyCorrelationID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)
	logger.WithCorrelationID("").(*DefaultLogger).Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.CorrelationID != "" {
		t.Errorf("expected empty correlation ID, got %s", entry.CorrelationID)
	}
}

// TestProperty23_ChainedOperations tests that chained operations maintain consistency
func TestProperty23_ChainedOperations(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.WithCorrelationID("test-id").(*DefaultLogger).
		WithField("key1", "value1").
		WithField("key2", "value2").
		Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.CorrelationID != "test-id" {
		t.Errorf("correlation ID lost in chain: expected test-id, got %s", entry.CorrelationID)
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("key1 lost in chain: expected value1, got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != "value2" {
		t.Errorf("key2 lost in chain: expected value2, got %v", entry.Fields["key2"])
	}
}

// TestProperty23_DistributedTracingScenario tests a realistic distributed tracing scenario
func TestProperty23_DistributedTracingScenario(t *testing.T) {
	requestID := "req-12345"
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	operations := []string{"auth", "validate", "process", "store"}
	for _, op := range operations {
		logger.WithCorrelationID(requestID).(*DefaultLogger).
			WithField("operation", op).
			Info(fmt.Sprintf("executing %s", op))
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != len(operations) {
		t.Errorf("expected %d log lines, got %d", len(operations), len(lines))
	}

	for i, line := range lines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry %d: %v", i, err)
		}

		if entry.CorrelationID != requestID {
			t.Errorf("log %d has wrong correlation ID: expected %s, got %s", i, requestID, entry.CorrelationID)
		}
	}
}
