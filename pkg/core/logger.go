package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// DefaultLogger provides structured logging with correlation IDs
type DefaultLogger struct {
	mu              sync.RWMutex
	level           LogLevel
	correlationID   string
	output          io.Writer
	fields          map[string]interface{}
	enableTimestamp bool
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     string                 `json:"timestamp,omitempty"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}

// NewDefaultLogger creates a new logger instance
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:           level,
		output:          os.Stdout,
		fields:          make(map[string]interface{}),
		enableTimestamp: true,
	}
}

// NewDefaultLoggerWithOutput creates a logger with custom output
func NewDefaultLoggerWithOutput(level LogLevel, output io.Writer) *DefaultLogger {
	return &DefaultLogger{
		level:           level,
		output:          output,
		fields:          make(map[string]interface{}),
		enableTimestamp: true,
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	l.log(LogLevelDebug, msg, fields...)
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	l.log(LogLevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	l.log(LogLevelWarn, msg, fields...)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	l.log(LogLevelError, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *DefaultLogger) Fatal(msg string, fields ...interface{}) {
	l.log(LogLevelFatal, msg, fields...)
	os.Exit(1)
}

// WithCorrelationID returns a new logger with correlation ID
func (l *DefaultLogger) WithCorrelationID(id string) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &DefaultLogger{
		level:           l.level,
		correlationID:   id,
		output:          l.output,
		fields:          make(map[string]interface{}),
		enableTimestamp: l.enableTimestamp,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithField adds a field to the logger
func (l *DefaultLogger) WithField(key string, value interface{}) *DefaultLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &DefaultLogger{
		level:           l.level,
		correlationID:   l.correlationID,
		output:          l.output,
		fields:          make(map[string]interface{}),
		enableTimestamp: l.enableTimestamp,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *DefaultLogger) WithFields(fields map[string]interface{}) *DefaultLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &DefaultLogger{
		level:           l.level,
		correlationID:   l.correlationID,
		output:          l.output,
		fields:          make(map[string]interface{}),
		enableTimestamp: l.enableTimestamp,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// SetLevel sets the log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *DefaultLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// log is the internal logging function
func (l *DefaultLogger) log(level LogLevel, msg string, fields ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check if we should log this level
	if level < l.level {
		return
	}

	// Parse fields
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}

	// Merge with existing fields
	for k, v := range l.fields {
		if _, exists := fieldMap[k]; !exists {
			fieldMap[k] = v
		}
	}

	// Create log entry
	entry := LogEntry{
		Level:   level.String(),
		Message: msg,
		Fields:  fieldMap,
	}

	if l.enableTimestamp {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if l.correlationID != "" {
		entry.CorrelationID = l.correlationID
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		_, _ = fmt.Fprintf(l.output, "error marshaling log entry: %v\n", err)
		return
	}

	// Write to output
	_, _ = fmt.Fprintf(l.output, "%s\n", string(data))
}

// LogLevel represents the log level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string to LogLevel
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}
