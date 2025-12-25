package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger(false)
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if logger == nil {
		t.Error("Expected logger instance, got nil")
	}
}

func TestNewLoggerDefault(t *testing.T) {
	logger, err := NewLoggerDefault()
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if logger == nil {
		t.Error("Expected logger instance, got nil")
	}
}

func TestLoggerInfo(t *testing.T) {
	logger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Just test that the method can be called without panicking
	logger.Info("Test message", "key", "value")
}

func TestLoggerError(t *testing.T) {
	logger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Just test that the method can be called without panicking
	logger.Error("Test error message", "error", "some error")
}

func TestLoggerWarn(t *testing.T) {
	logger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Just test that the method can be called without panicking
	logger.Warn("Test warning message", "warning", "some warning")
}

func TestLoggerDebug(t *testing.T) {
	logger, err := NewLogger(true) // Enable debug mode
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Just test that the method can be called without panicking
	logger.Debug("Test debug message", "debug", "some debug")
}

func TestLoggerWithFields(t *testing.T) {
	logger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	fields := map[string]interface{}{
		"test": "value",
		"num":  42,
	}
	
	newLogger := logger.WithFields(fields)
	if newLogger == nil {
		t.Error("Expected logger instance with fields, got nil")
	}
}

func TestLoggerSync(t *testing.T) {
	logger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	err = logger.Sync()
	if err != nil {
		t.Errorf("Expected no error during sync, got %v", err)
	}
}