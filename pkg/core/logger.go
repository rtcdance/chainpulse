package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

type DefaultLogger struct {
	level           atomic.Int32
	correlationID   string
	output          io.Writer
	fields          map[string]any
	enableTimestamp bool
}

type LogEntry struct {
	Timestamp     string         `json:"timestamp,omitempty"`
	Level         string         `json:"level"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
}

func NewProductionLogger() Logger {
	return NewSlogLogger(LogLevelInfo, "json")
}

func NewProductionLoggerWithLevel(level LogLevel) Logger {
	return NewSlogLogger(level, "json")
}

func NewDefaultLogger(level LogLevel) *DefaultLogger {
	l := &DefaultLogger{
		output:          os.Stdout,
		fields:          make(map[string]any),
		enableTimestamp: true,
	}
	l.level.Store(int32(level))
	return l
}

func NewDefaultLoggerWithOutput(level LogLevel, output io.Writer) *DefaultLogger {
	l := &DefaultLogger{
		output:          output,
		fields:          make(map[string]any),
		enableTimestamp: true,
	}
	l.level.Store(int32(level))
	return l
}

func (l *DefaultLogger) Debug(msg string, fields ...any) {
	l.log(LogLevelDebug, msg, fields...)
}

func (l *DefaultLogger) Info(msg string, fields ...any) {
	l.log(LogLevelInfo, msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, fields ...any) {
	l.log(LogLevelWarn, msg, fields...)
}

func (l *DefaultLogger) Error(msg string, fields ...any) {
	l.log(LogLevelError, msg, fields...)
}

func (l *DefaultLogger) Fatal(msg string, fields ...any) {
	l.log(LogLevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *DefaultLogger) WithCorrelationID(id string) Logger {
	newLogger := &DefaultLogger{
		correlationID:   id,
		output:          l.output,
		fields:          make(map[string]any),
		enableTimestamp: l.enableTimestamp,
	}
	newLogger.level.Store(l.level.Load())
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *DefaultLogger) WithField(key string, value any) *DefaultLogger {
	newLogger := &DefaultLogger{
		correlationID:   l.correlationID,
		output:          l.output,
		fields:          make(map[string]any),
		enableTimestamp: l.enableTimestamp,
	}
	newLogger.level.Store(l.level.Load())
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

func (l *DefaultLogger) WithFields(fields map[string]any) *DefaultLogger {
	newLogger := &DefaultLogger{
		correlationID:   l.correlationID,
		output:          l.output,
		fields:          make(map[string]any),
		enableTimestamp: l.enableTimestamp,
	}
	newLogger.level.Store(l.level.Load())
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level.Store(int32(level))
}

func (l *DefaultLogger) GetLevel() LogLevel {
	return LogLevel(l.level.Load())
}

func (l *DefaultLogger) log(level LogLevel, msg string, fields ...any) {
	if level < l.GetLevel() {
		return
	}

	fieldMap := make(map[string]any)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}

	for k, v := range l.fields {
		if _, exists := fieldMap[k]; !exists {
			fieldMap[k] = v
		}
	}

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

	data, err := json.Marshal(entry)
	if err != nil {
		if l.output != os.Stderr {
			_, _ = fmt.Fprintf(os.Stderr, "logger: failed to marshal log entry: %v\n", err)
		}
		return
	}

	if _, err := fmt.Fprintf(l.output, "%s\n", string(data)); err != nil {
		if l.output != os.Stderr {
			_, _ = fmt.Fprintf(os.Stderr, "logger: failed to write to output: %v\n", err)
		}
	}
}

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

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
