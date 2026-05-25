package core

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestSlogLevelFromCore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		level LogLevel
		want  slog.Level
	}{
		{"debug", LogLevelDebug, slog.LevelDebug},
		{"info", LogLevelInfo, slog.LevelInfo},
		{"warn", LogLevelWarn, slog.LevelWarn},
		{"error", LogLevelError, slog.LevelError},
		{"fatal", LogLevelFatal, slog.LevelError},
		{"unknown", LogLevel(999), slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slogLevelFromCore(tt.level); got != tt.want {
				t.Errorf("slogLevelFromCore(%d) = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestNewSlogLoggerWithHandler(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelDebug)
	if l == nil {
		t.Fatal("NewSlogLoggerWithHandler returned nil")
	}
	if l.GetLevel() != LogLevelDebug {
		t.Errorf("expected LogLevelDebug, got %v", l.GetLevel())
	}
}

func TestSlogLoggerSetAndGetLevel(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	if l.GetLevel() != LogLevelDebug {
		t.Errorf("GetLevel() = %v, want %v", l.GetLevel(), LogLevelDebug)
	}
	l.SetLevel(LogLevelWarn)
	if l.GetLevel() != LogLevelWarn {
		t.Errorf("GetLevel() = %v, want %v", l.GetLevel(), LogLevelWarn)
	}
}

func TestSlogLoggerWithField(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithField("chain", "ethereum")
	if l2 == nil {
		t.Fatal("WithField returned nil")
	}
	if len(l2.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(l2.fields))
	}
	if l2.fields["chain"] != "ethereum" {
		t.Errorf("expected chain=ethereum, got %v", l2.fields["chain"])
	}
	if len(l.fields) != 0 {
		t.Error("original logger fields should not be modified")
	}
}

func TestSlogLoggerWithFields(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithFields(map[string]any{"chain": "ethereum", "network": "mainnet"})
	if l2 == nil {
		t.Fatal("WithFields returned nil")
	}
	if len(l2.fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(l2.fields))
	}
	if l2.fields["chain"] != "ethereum" {
		t.Errorf("expected chain=ethereum, got %v", l2.fields["chain"])
	}
	if len(l.fields) != 0 {
		t.Error("original logger fields should not be modified")
	}
}

func TestSlogLoggerWithCorrelationID(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithCorrelationID("trace-123")
	if l2 == nil {
		t.Fatal("WithCorrelationID returned nil")
	}
	sl := l2.(*SlogLogger)
	if sl.corrID != "trace-123" {
		t.Errorf("expected corrID=trace-123, got %v", sl.corrID)
	}
	if len(sl.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(sl.fields))
	}
}

func TestSlogLoggerToSlogArgs(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithField("persistent", "val")

	tests := []struct {
		name   string
		fields []any
		want   int
	}{
		{"no_fields", nil, 2},
		{"with_fields", []any{"key1", "value1", "key2", 42}, 6},
		{"odd_length", []any{"key1", "value1", "key2"}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := l2.toSlogArgs(tt.fields...)
			if len(args) != tt.want {
				t.Errorf("toSlogArgs() returned %d args, want %d", len(args), tt.want)
			}
		})
	}
}

func TestSlogLoggerInfo(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelDebug)
	l.Info("test message", "key", "value")
	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
	if !bytes.Contains([]byte(output), []byte("test message")) {
		t.Error("expected 'test message' in log output")
	}
}

func TestSlogLoggerWarn(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelDebug)
	l.Warn("warning message", "key", "value")
	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}

func TestSlogLoggerError(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelDebug)
	l.Error("error message", "key", "value")
	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}

func TestSlogLoggerDebugSuppressed(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelInfo)
	l.Debug("should not appear")
	output := buf.String()
	if len(output) > 0 {
		t.Error("expected no log output when level is above debug")
	}
}

func TestSlogLoggerInfoSuppressed(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelWarn)
	l.Info("should not appear")
	output := buf.String()
	if len(output) > 0 {
		t.Error("expected no log output when level is above info")
	}
}

func TestSlogLoggerWarnSuppressed(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelError)
	l.Warn("should not appear")
	output := buf.String()
	if len(output) > 0 {
		t.Error("expected no log output when level is above warn")
	}
}

func TestSlogLoggerErrorSuppressed(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := NewSlogLoggerWithHandler(handler, LogLevelFatal)
	l.Error("should not appear")
	output := buf.String()
	if len(output) > 0 {
		t.Error("expected no log output when level is above error")
	}
}

func TestSlogLoggerWithCorrelationID_Empty(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithCorrelationID("")
	if l2 == nil {
		t.Fatal("WithCorrelationID returned nil")
	}
}

func TestSlogLoggerWithField_Overwrite(t *testing.T) {
	t.Parallel()
	l := NewSlogLogger(LogLevelDebug, "text")
	l2 := l.WithField("chain", "polygon")
	l3 := l2.WithField("chain", "arbitrum")
	if l3.fields["chain"] != "arbitrum" {
		t.Errorf("expected chain=arbitrum, got %v", l3.fields["chain"])
	}
}

func TestSlogLoggerDebug(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := NewSlogLoggerWithHandler(handler, LogLevelDebug)
	l.Debug("debug message", "key", "value")
	output := buf.String()
	if output == "" {
		t.Error("expected log output")
	}
}
