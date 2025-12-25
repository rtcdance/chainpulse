package logger

import (
	"context"
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface defines the logging methods
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	WithFields(fields map[string]interface{}) Logger
	WithTrace(ctx context.Context) Logger
	Sync() error
}

// ZapLogger wraps the zap logger
type ZapLogger struct {
	sugaredLogger *zap.SugaredLogger
}

// NewLogger creates a new logger instance
func NewLogger(debugMode bool) (Logger, error) {
	var cfg zap.Config
	
	if debugMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	zapLogger := &ZapLogger{
		sugaredLogger: logger.Sugar(),
	}

	return zapLogger, nil
}

// NewLoggerDefault creates a new logger instance with default settings (production)
func NewLoggerDefault() (Logger, error) {
	return NewLogger(false)
}

// Info logs an info message
func (zl *ZapLogger) Info(msg string, args ...interface{}) {
	zl.sugaredLogger.Infof(msg, args...)
}

// Error logs an error message
func (zl *ZapLogger) Error(msg string, args ...interface{}) {
	zl.sugaredLogger.Errorf(msg, args...)
}

// Warn logs a warning message
func (zl *ZapLogger) Warn(msg string, args ...interface{}) {
	zl.sugaredLogger.Warnf(msg, args...)
}

// Debug logs a debug message
func (zl *ZapLogger) Debug(msg string, args ...interface{}) {
	zl.sugaredLogger.Debugf(msg, args...)
}

// WithFields adds fields to the logger
func (zl *ZapLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := zl.sugaredLogger.With()
	for k, v := range fields {
		newLogger = newLogger.With(k, v)
	}
	
	return &ZapLogger{
		sugaredLogger: newLogger,
	}
}

// WithTrace adds trace context to the logger
func (zl *ZapLogger) WithTrace(ctx context.Context) Logger {
	// In a real implementation, this would extract trace information from context
	// For now, we'll just return the logger as is
	return zl
}

// Sync flushes any buffered log entries
func (zl *ZapLogger) Sync() error {
	return zl.sugaredLogger.Sync()
}

// StdLogger wraps the standard logger with our logger interface
type StdLogger struct {
	logger  *log.Logger
	debugMode bool
}

// NewStdLogger creates a new standard logger
func NewStdLogger(debugMode bool) Logger {
	return &StdLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		debugMode: debugMode,
	}
}

// Info logs an info message
func (sl *StdLogger) Info(msg string, args ...interface{}) {
	sl.logger.Printf("[INFO] "+msg, args...)
}

// Error logs an error message
func (sl *StdLogger) Error(msg string, args ...interface{}) {
	sl.logger.Printf("[ERROR] "+msg, args...)
}

// Warn logs a warning message
func (sl *StdLogger) Warn(msg string, args ...interface{}) {
	sl.logger.Printf("[WARN] "+msg, args...)
}

// Debug logs a debug message
func (sl *StdLogger) Debug(msg string, args ...interface{}) {
	if sl.debugMode {
		sl.logger.Printf("[DEBUG] "+msg, args...)
	}
}

// WithFields adds fields to the logger (stub implementation)
func (sl *StdLogger) WithFields(fields map[string]interface{}) Logger {
	return sl
}

// WithTrace adds trace context to the logger (stub implementation)
func (sl *StdLogger) WithTrace(ctx context.Context) Logger {
	return sl
}

// Sync is a no-op for the standard logger
func (sl *StdLogger) Sync() error {
	return nil
}