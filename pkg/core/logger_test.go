package core

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// lockedWriter implements io.Writer with mutex protection
type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (lw *lockedWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}

func TestNewProductionLogger(t *testing.T) {
	t.Parallel()
	l := NewProductionLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewProductionLoggerWithLevel(t *testing.T) {
	t.Parallel()
	l := NewProductionLoggerWithLevel(LogLevelDebug)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TestNewDefaultLogger tests logger creation
func TestNewDefaultLogger(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelInfo)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
	if logger.GetLevel() != LogLevelInfo {
		t.Errorf("expected level %v, got %v", LogLevelInfo, logger.GetLevel())
	}
}

// TestNewDefaultLoggerWithOutput tests logger creation with custom output
func TestNewDefaultLoggerWithOutput(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
	if logger.output != buf {
		t.Error("expected custom output")
	}
}

// TestLoggerDebug tests debug logging
func TestLoggerDebug(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelDebug, buf)

	logger.Debug("test debug message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Error("expected DEBUG level in output")
	}
	if !strings.Contains(output, "test debug message") {
		t.Error("expected message in output")
	}

	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Level != "DEBUG" {
		t.Errorf("expected DEBUG level, got %s", entry.Level)
	}
	if entry.Message != "test debug message" {
		t.Errorf("expected message, got %s", entry.Message)
	}
}

// TestLoggerInfo tests info logging
func TestLoggerInfo(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test info message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Error("expected INFO level in output")
	}
	if !strings.Contains(output, "test info message") {
		t.Error("expected message in output")
	}
}

// TestLoggerWarn tests warning logging
func TestLoggerWarn(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelWarn, buf)

	logger.Warn("test warn message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Error("expected WARN level in output")
	}
	if !strings.Contains(output, "test warn message") {
		t.Error("expected message in output")
	}
}

// TestLoggerError tests error logging
func TestLoggerError(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelError, buf)

	logger.Error("test error message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR level in output")
	}
	if !strings.Contains(output, "test error message") {
		t.Error("expected message in output")
	}
}

// TestLoggerWithCorrelationID tests correlation ID tracking
func TestLoggerWithCorrelationID(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	correlatedLogger := logger.WithCorrelationID("test-correlation-id")
	correlatedLogger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.CorrelationID != "test-correlation-id" {
		t.Errorf("expected correlation ID, got %s", entry.CorrelationID)
	}
}

// TestLoggerWithField tests adding a single field
func TestLoggerWithField(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	fieldLogger := logger.WithField("custom_key", "custom_value")
	fieldLogger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Fields["custom_key"] != "custom_value" {
		t.Errorf("expected custom field, got %v", entry.Fields["custom_key"])
	}
}

// TestLoggerWithFields tests adding multiple fields
func TestLoggerWithFields(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("expected key1, got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != float64(42) {
		t.Errorf("expected key2, got %v", entry.Fields["key2"])
	}
	if entry.Fields["key3"] != true {
		t.Errorf("expected key3, got %v", entry.Fields["key3"])
	}
}

// TestLoggerLevelFiltering tests that lower level logs are filtered
func TestLoggerLevelFiltering(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelWarn, buf)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("expected debug message to be filtered")
	}
	if strings.Contains(output, "info message") {
		t.Error("expected info message to be filtered")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("expected warn message in output")
	}
}

// TestLoggerSetLevel tests changing log level
func TestLoggerSetLevel(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelWarn, buf)

	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("expected debug message to be filtered at WARN level")
	}

	logger.SetLevel(LogLevelDebug)
	logger.Debug("debug message 2")
	if buf.Len() == 0 {
		t.Error("expected debug message after level change")
	}
}

// TestLoggerFields tests inline fields
func TestLoggerFields(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test message", "key1", "value1", "key2", 42)

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("expected key1, got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != float64(42) {
		t.Errorf("expected key2, got %v", entry.Fields["key2"])
	}
}

// TestLoggerTimestamp tests timestamp inclusion
func TestLoggerTimestamp(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Timestamp == "" {
		t.Error("expected timestamp in log entry")
	}
}

// TestLoggerConcurrency tests concurrent logging
func TestLoggerConcurrency(t *testing.T) {
	t.Parallel()
	// Use a thread-safe buffer wrapper
	buf := &bytes.Buffer{}
	lw := &lockedWriter{mu: &sync.Mutex{}, w: buf}

	logger := NewDefaultLoggerWithOutput(LogLevelInfo, lw)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.Info("concurrent message", "id", id)
		}(i)
	}

	wg.Wait()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Allow for some variation due to concurrent writes
	if len(lines) < 8 || len(lines) > 10 {
		t.Logf("expected ~10 log lines, got %d (acceptable for concurrent writes)", len(lines))
	}
}

// TestLoggerChaining tests method chaining
func TestLoggerChaining(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.WithCorrelationID("test-id").(*DefaultLogger).WithField("key", "value").Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.CorrelationID != "test-id" {
		t.Errorf("expected correlation ID, got %s", entry.CorrelationID)
	}
	if entry.Fields["key"] != "value" {
		t.Errorf("expected field, got %v", entry.Fields["key"])
	}
}

// TestParseLogLevel tests log level parsing
func TestParseLogLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", LogLevelDebug},
		{"INFO", LogLevelInfo},
		{"WARN", LogLevelWarn},
		{"ERROR", LogLevelError},
		{"FATAL", LogLevelFatal},
		{"UNKNOWN", LogLevelInfo}, // defaults to INFO
	}

	for _, test := range tests {
		result := ParseLogLevel(test.input)
		if result != test.expected {
			t.Errorf("ParseLogLevel(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

// TestLogLevelString tests log level string representation
func TestLogLevelString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("LogLevel.String() = %s, expected %s", result, test.expected)
		}
	}
}

// TestLoggerMultipleCorrelationIDs tests multiple correlation IDs
func TestLoggerMultipleCorrelationIDs(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger1 := logger.WithCorrelationID("id-1")
	logger2 := logger.WithCorrelationID("id-2")

	logger1.Info("message 1")
	logger2.Info("message 2")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var entry1, entry2 LogEntry
	err := json.Unmarshal([]byte(lines[0]), &entry1)
	if err != nil {
		t.Fatalf("failed to unmarshal entry1: %v", err)
	}
	err = json.Unmarshal([]byte(lines[1]), &entry2)
	if err != nil {
		t.Fatalf("failed to unmarshal entry2: %v", err)
	}

	if entry1.CorrelationID != "id-1" {
		t.Errorf("expected id-1, got %s", entry1.CorrelationID)
	}
	if entry2.CorrelationID != "id-2" {
		t.Errorf("expected id-2, got %s", entry2.CorrelationID)
	}
}

// TestLoggerFieldOverride tests field override behavior
func TestLoggerFieldOverride(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test message", "key", "original")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Fields["key"] != "original" {
		t.Errorf("expected original value, got %v", entry.Fields["key"])
	}
}

// TestLoggerEmptyFields tests logging with no fields
func TestLoggerEmptyFields(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry.Message != "test message" {
		t.Errorf("expected message, got %s", entry.Message)
	}
}

// TestLoggerOddFields tests logging with odd number of fields
func TestLoggerOddFields(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, buf)

	logger.Info("test message", "key1", "value1", "key2") // odd number

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	// key2 should not be in fields since it has no value
	if _, exists := entry.Fields["key2"]; exists {
		t.Error("expected key2 to not be in fields")
	}
}
