package resilience

import (
	"chainpulse/pkg/core"
	"context"
	"errors"
	"testing"
)

func TestErrorClassifierTransient(t *testing.T) {
	classifier := NewDefaultErrorClassifier()

	transientErrors := []error{
		errors.New("timeout"),
		errors.New("connection refused"),
		errors.New("connection reset"),
		errors.New("temporary failure"),
		errors.New("unavailable"),
		errors.New("deadline exceeded"),
	}

	for _, err := range transientErrors {
		if !classifier.IsTransient(err) {
			t.Errorf("Expected error to be transient: %v", err)
		}

		if classifier.IsPermanent(err) {
			t.Errorf("Expected error not to be permanent: %v", err)
		}

		if classifier.IsCritical(err) {
			t.Errorf("Expected error not to be critical: %v", err)
		}

		category := classifier.Classify(err)
		if category != ErrorCategoryTransient {
			t.Errorf("Expected transient category, got %s", category)
		}
	}
}

func TestErrorClassifierPermanent(t *testing.T) {
	classifier := NewDefaultErrorClassifier()

	permanentErrors := []error{
		errors.New("invalid configuration"),
		errors.New("unauthorized access"),
		errors.New("forbidden resource"),
		errors.New("not found"),
		errors.New("bad request"),
		errors.New("corrupted data"),
		errors.New("authentication failed"),
	}

	for _, err := range permanentErrors {
		if classifier.IsTransient(err) {
			t.Errorf("Expected error not to be transient: %v", err)
		}

		if !classifier.IsPermanent(err) {
			t.Errorf("Expected error to be permanent: %v", err)
		}

		if classifier.IsCritical(err) {
			t.Errorf("Expected error not to be critical: %v", err)
		}

		category := classifier.Classify(err)
		if category != ErrorCategoryPermanent {
			t.Errorf("Expected permanent category, got %s", category)
		}
	}
}

func TestErrorClassifierCritical(t *testing.T) {
	classifier := NewDefaultErrorClassifier()

	criticalErrors := []error{
		errors.New("data corruption detected"),
		errors.New("critical error occurred"),
		errors.New("fatal system error"),
		errors.New("panic in handler"),
	}

	for _, err := range criticalErrors {
		if classifier.IsTransient(err) {
			t.Errorf("Expected error not to be transient: %v", err)
		}

		if classifier.IsPermanent(err) {
			t.Errorf("Expected error not to be permanent: %v", err)
		}

		if !classifier.IsCritical(err) {
			t.Errorf("Expected error to be critical: %v", err)
		}

		category := classifier.Classify(err)
		if category != ErrorCategoryCritical {
			t.Errorf("Expected critical category, got %s", category)
		}
	}
}

func TestErrorClassifierNil(t *testing.T) {
	classifier := NewDefaultErrorClassifier()

	category := classifier.Classify(nil)
	if category != ErrorCategoryUnknown {
		t.Errorf("Expected unknown category for nil error, got %s", category)
	}

	if classifier.IsTransient(nil) {
		t.Error("Expected nil error not to be transient")
	}

	if classifier.IsPermanent(nil) {
		t.Error("Expected nil error not to be permanent")
	}

	if classifier.IsCritical(nil) {
		t.Error("Expected nil error not to be critical")
	}
}

func TestDefaultErrorLogger(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	errorLogger := NewDefaultErrorLogger(logger)

	ctx := context.Background()
	err := errors.New("test error")
	context := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	// Should not panic
	errorLogger.LogError(ctx, err, ErrorCategoryTransient, context)
	errorLogger.LogError(ctx, err, ErrorCategoryPermanent, context)
	errorLogger.LogError(ctx, err, ErrorCategoryCritical, context)
	errorLogger.LogError(ctx, nil, ErrorCategoryUnknown, context)
}

func TestDefaultContextProvider(t *testing.T) {
	provider := NewDefaultContextProvider("corr-123", "req-456", "user-789")

	ctx := context.Background()
	context := provider.GetContext(ctx)

	if context["correlationID"] != "corr-123" {
		t.Errorf("Expected correlationID to be corr-123, got %v", context["correlationID"])
	}

	if context["requestID"] != "req-456" {
		t.Errorf("Expected requestID to be req-456, got %v", context["requestID"])
	}

	if context["userID"] != "user-789" {
		t.Errorf("Expected userID to be user-789, got %v", context["userID"])
	}

	if context["timestamp"] == nil {
		t.Error("Expected timestamp to be set")
	}
}

func TestErrorHandlerRegistration(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	// Register error logger
	errorLogger := NewDefaultErrorLogger(logger)
	handler.RegisterErrorLogger("test", errorLogger)

	// Register context provider
	contextProvider := NewDefaultContextProvider("corr-123", "req-456", "user-789")
	handler.RegisterContextProvider("test", contextProvider)

	// Verify registration
	handler.errorLoggersMu.RLock()
	if len(handler.errorLoggers) != 1 {
		t.Errorf("Expected 1 error logger, got %d", len(handler.errorLoggers))
	}
	handler.errorLoggersMu.RUnlock()

	handler.contextProvidersMu.RLock()
	if len(handler.contextProviders) != 1 {
		t.Errorf("Expected 1 context provider, got %d", len(handler.contextProviders))
	}
	handler.contextProvidersMu.RUnlock()

	// Unregister
	handler.UnregisterErrorLogger("test")
	handler.UnregisterContextProvider("test")

	// Verify unregistration
	handler.errorLoggersMu.RLock()
	if len(handler.errorLoggers) != 0 {
		t.Errorf("Expected 0 error loggers, got %d", len(handler.errorLoggers))
	}
	handler.errorLoggersMu.RUnlock()

	handler.contextProvidersMu.RLock()
	if len(handler.contextProviders) != 0 {
		t.Errorf("Expected 0 context providers, got %d", len(handler.contextProviders))
	}
	handler.contextProvidersMu.RUnlock()
}

func TestErrorHandlerHandleError(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	ctx := context.Background()
	err := errors.New("timeout")

	// Should not panic
	handler.HandleError(ctx, err, "test_source")

	// Verify classification
	if !handler.IsTransient(err) {
		t.Error("Expected error to be transient")
	}

	// Test with permanent error
	err = errors.New("invalid configuration")
	handler.HandleError(ctx, err, "test_source")

	if !handler.IsPermanent(err) {
		t.Error("Expected error to be permanent")
	}

	// Test with critical error
	err = errors.New("data corruption")
	handler.HandleError(ctx, err, "test_source")

	if !handler.IsCritical(err) {
		t.Error("Expected error to be critical")
	}

	// Test with nil error
	handler.HandleError(ctx, nil, "test_source")
}

func TestErrorHandlerClassify(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	tests := []struct {
		err      error
		expected ErrorCategory
	}{
		{errors.New("timeout"), ErrorCategoryTransient},
		{errors.New("invalid"), ErrorCategoryPermanent},
		{errors.New("data corruption"), ErrorCategoryCritical},
		{errors.New("unknown error"), ErrorCategoryUnknown},
		{nil, ErrorCategoryUnknown},
	}

	for _, test := range tests {
		category := handler.Classify(test.err)
		if category != test.expected {
			t.Errorf("Expected %s, got %s for error %v", test.expected, category, test.err)
		}
	}
}

func TestErrorHandlerConcurrentHandling(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	ctx := context.Background()
	done := make(chan bool, 20)

	// Concurrent error handling
	for i := 0; i < 20; i++ {
		go func(index int) {
			err := errors.New("timeout")
			handler.HandleError(ctx, err, "test_source")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestErrorHandlerWithContextProviders(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	// Register context provider
	contextProvider := NewDefaultContextProvider("corr-123", "req-456", "user-789")
	handler.RegisterContextProvider("test", contextProvider)

	ctx := context.Background()
	err := errors.New("timeout")

	// Should not panic and should use context provider
	handler.HandleError(ctx, err, "test_source")
}

func TestErrorHandlerMetricsRecording(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	ctx := context.Background()

	// Record transient error
	err := errors.New("timeout")
	handler.HandleError(ctx, err, "test_source")

	// Record permanent error
	err = errors.New("invalid")
	handler.HandleError(ctx, err, "test_source")

	// Record critical error
	err = errors.New("data corruption")
	handler.HandleError(ctx, err, "test_source")

	// Verify metrics were recorded
	metrics := metricsCollector.Export()
	if metrics == nil {
		t.Error("Expected metrics to be recorded")
	}
}

func TestErrorHandlerMultipleLoggers(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	// Register multiple error loggers
	errorLogger1 := NewDefaultErrorLogger(logger)
	errorLogger2 := NewDefaultErrorLogger(logger)

	handler.RegisterErrorLogger("logger1", errorLogger1)
	handler.RegisterErrorLogger("logger2", errorLogger2)

	ctx := context.Background()
	err := errors.New("timeout")

	// Should use all registered loggers
	handler.HandleError(ctx, err, "test_source")

	// Verify both loggers are registered
	handler.errorLoggersMu.RLock()
	if len(handler.errorLoggers) != 2 {
		t.Errorf("Expected 2 error loggers, got %d", len(handler.errorLoggers))
	}
	handler.errorLoggersMu.RUnlock()
}

func TestErrorHandlerErrorClassification(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	// Test various error messages
	tests := []struct {
		errMsg   string
		expected ErrorCategory
	}{
		{"connection timeout", ErrorCategoryTransient},
		{"connection refused", ErrorCategoryTransient},
		{"connection reset", ErrorCategoryTransient},
		{"temporary failure", ErrorCategoryTransient},
		{"service unavailable", ErrorCategoryTransient},
		{"deadline exceeded", ErrorCategoryTransient},
		{"invalid request", ErrorCategoryPermanent},
		{"unauthorized", ErrorCategoryPermanent},
		{"forbidden", ErrorCategoryPermanent},
		{"not found", ErrorCategoryPermanent},
		{"bad request", ErrorCategoryPermanent},
		{"corrupted data", ErrorCategoryPermanent},
		{"authentication failed", ErrorCategoryPermanent},
		{"data corruption detected", ErrorCategoryCritical},
		{"critical error", ErrorCategoryCritical},
		{"fatal error", ErrorCategoryCritical},
		{"panic", ErrorCategoryCritical},
		{"unknown error", ErrorCategoryUnknown},
	}

	for _, test := range tests {
		err := errors.New(test.errMsg)
		category := handler.Classify(err)
		if category != test.expected {
			t.Errorf("For error '%s': expected %s, got %s", test.errMsg, test.expected, category)
		}
	}
}
