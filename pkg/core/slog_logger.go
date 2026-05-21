package core

import (
	"log/slog"
	"os"
)

// SlogLogger implements the Logger interface using Go's standard log/slog.
// This provides type-safe structured logging with slog.String(), slog.Int(), etc.
type SlogLogger struct {
	logger *slog.Logger
	level  LogLevel
	corrID string
	fields map[string]any
}

// NewSlogLogger creates a new Logger backed by log/slog.
// format: "text" or "json" (default: json)
func NewSlogLogger(level LogLevel, format string) *SlogLogger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: slogLevelFromCore(level),
	}

	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &SlogLogger{
		logger: slog.New(handler),
		level:  level,
		fields: make(map[string]any),
	}
}

// NewSlogLoggerWithHandler creates a SlogLogger with a custom slog.Handler.
func NewSlogLoggerWithHandler(handler slog.Handler, level LogLevel) *SlogLogger {
	return &SlogLogger{
		logger: slog.New(handler),
		level:  level,
		fields: make(map[string]any),
	}
}

func slogLevelFromCore(level LogLevel) slog.Level {
	switch level {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	case LogLevelFatal:
		return slog.LevelError // slog has no Fatal level
	default:
		return slog.LevelInfo
	}
}

// Debug logs a debug message
func (l *SlogLogger) Debug(msg string, fields ...any) {
	if l.level > LogLevelDebug {
		return
	}
	l.logger.Debug(msg, l.toSlogArgs(fields...)...)
}

// Info logs an info message
func (l *SlogLogger) Info(msg string, fields ...any) {
	if l.level > LogLevelInfo {
		return
	}
	l.logger.Info(msg, l.toSlogArgs(fields...)...)
}

// Warn logs a warning message
func (l *SlogLogger) Warn(msg string, fields ...any) {
	if l.level > LogLevelWarn {
		return
	}
	l.logger.Warn(msg, l.toSlogArgs(fields...)...)
}

// Error logs an error message
func (l *SlogLogger) Error(msg string, fields ...any) {
	if l.level > LogLevelError {
		return
	}
	l.logger.Error(msg, l.toSlogArgs(fields...)...)
}

// Fatal logs a fatal message and exits
func (l *SlogLogger) Fatal(msg string, fields ...any) {
	l.logger.Error(msg, l.toSlogArgs(fields...)...)
	os.Exit(1)
}

// WithCorrelationID returns a new logger with correlation ID
func (l *SlogLogger) WithCorrelationID(id string) Logger {
	newFields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields["correlation_id"] = id

	return &SlogLogger{
		logger: l.logger.With("correlation_id", id),
		level:  l.level,
		corrID: id,
		fields: newFields,
	}
}

// WithField adds a persistent field to the logger, returning a new instance.
func (l *SlogLogger) WithField(key string, value any) *SlogLogger {
	newFields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &SlogLogger{
		logger: l.logger.With(key, value),
		level:  l.level,
		corrID: l.corrID,
		fields: newFields,
	}
}

// WithFields adds multiple persistent fields to the logger, returning a new instance.
func (l *SlogLogger) WithFields(fields map[string]any) *SlogLogger {
	newFields := make(map[string]any, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &SlogLogger{
		logger: l.logger.With(args...),
		level:  l.level,
		corrID: l.corrID,
		fields: newFields,
	}
}

// SetLevel updates the log level.
func (l *SlogLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level.
func (l *SlogLogger) GetLevel() LogLevel {
	return l.level
}

// toSlogArgs converts variadic interface{} pairs to slog.Attr slices.
// Handles odd-length inputs gracefully by ignoring the last unpaired value.
func (l *SlogLogger) toSlogArgs(fields ...any) []any {
	args := make([]any, 0, len(l.fields)*2+len(fields))

	// Prepend any persistent fields
	for k, v := range l.fields {
		args = append(args, k, v)
	}

	// Add call-site fields
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				args = append(args, key, fields[i+1])
			}
		}
		// Odd-length: ignore last unpaired value
	}

	return args
}
