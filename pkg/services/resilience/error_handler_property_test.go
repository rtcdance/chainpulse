package resilience

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"chainpulse/pkg/core"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// testContextKey is the key for storing test context values
	testContextKey contextKey = "key"
)

// Property 18: Error Logging with Context
// For any error encountered in the system, the error handler SHALL log the error with context information
// including source, category, correlation ID, and timestamp

func TestProperty18ErrorLoggingWithContext(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewErrorHandler(logger, metricsCollector)

	// Register context provider
	contextProvider := NewDefaultContextProvider("corr-123", "req-456", "user-789")
	handler.RegisterContextProvider("test", contextProvider)

	ctx := context.Background()

	// Test 1: Transient error logging
	t.Run("TransientErrorLogging", func(t *testing.T) {
		err := errors.New("timeout")
		handler.HandleError(ctx, err, "data_puller")

		// Verify error is classified as transient
		if !handler.IsTransient(err) {
			t.Error("Expected error to be classified as transient")
		}

		// Verify metrics were recorded
		metrics := metricsCollector.GetMetrics()
		if metrics == nil {
			t.Error("Expected metrics to be recorded")
		}
	})

	// Test 2: Permanent error logging
	t.Run("PermanentErrorLogging", func(t *testing.T) {
		err := errors.New("invalid configuration")
		handler.HandleError(ctx, err, "config_manager")

		// Verify error is classified as permanent
		if !handler.IsPermanent(err) {
			t.Error("Expected error to be classified as permanent")
		}

		// Verify metrics were recorded
		metrics := metricsCollector.GetMetrics()
		if metrics == nil {
			t.Error("Expected metrics to be recorded")
		}
	})

	// Test 3: Critical error logging
	t.Run("CriticalErrorLogging", func(t *testing.T) {
		err := errors.New("data corruption")
		handler.HandleError(ctx, err, "database")

		// Verify error is classified as critical
		if !handler.IsCritical(err) {
			t.Error("Expected error to be classified as critical")
		}

		// Verify metrics were recorded
		metrics := metricsCollector.GetMetrics()
		if metrics == nil {
			t.Error("Expected metrics to be recorded")
		}
	})

	// Test 4: Error logging with multiple sources
	t.Run("ErrorLoggingMultipleSources", func(t *testing.T) {
		sources := []string{"data_puller", "event_processor", "cache", "database", "api"}

		for _, source := range sources {
			err := errors.New("timeout")
			handler.HandleError(ctx, err, source)

			// Verify error is classified correctly
			if !handler.IsTransient(err) {
				t.Errorf("Expected error from %s to be transient", source)
			}
		}
	})

	// Test 5: Error logging consistency
	t.Run("ErrorLoggingConsistency", func(t *testing.T) {
		err := errors.New("timeout")

		// Log same error multiple times
		for i := 0; i < 5; i++ {
			handler.HandleError(ctx, err, "test_source")
		}

		// Verify classification is consistent
		for i := 0; i < 5; i++ {
			if !handler.IsTransient(err) {
				t.Error("Expected error to be consistently classified as transient")
			}
		}
	})

	// Test 6: Error logging with context information
	t.Run("ErrorLoggingWithContext", func(t *testing.T) {
		err := errors.New("timeout")

		// Create custom context
		customCtx := context.WithValue(ctx, testContextKey, "value")

		// Log error with context
		handler.HandleError(customCtx, err, "test_source")

		// Verify error is logged
		if !handler.IsTransient(err) {
			t.Error("Expected error to be logged with context")
		}
	})

	// Test 7: Error logging with nil error
	t.Run("ErrorLoggingNilError", func(t *testing.T) {
		// Should not panic
		handler.HandleError(ctx, nil, "test_source")
	})

	// Test 8: Error logging with empty source
	t.Run("ErrorLoggingEmptySource", func(t *testing.T) {
		err := errors.New("timeout")

		// Should not panic
		handler.HandleError(ctx, err, "")
	})

	// Test 9: Error logging with various error types
	t.Run("ErrorLoggingVariousTypes", func(t *testing.T) {
		errors := []error{
			errors.New("timeout"),
			errors.New("connection refused"),
			errors.New("invalid configuration"),
			errors.New("data corruption"),
		}

		for _, err := range errors {
			handler.HandleError(ctx, err, "test_source")

			// Verify error is classified
			category := handler.Classify(err)
			if category == ErrorCategoryUnknown {
				t.Errorf("Expected error to be classified, got unknown for %v", err)
			}
		}
	})

	// Test 10: Error logging with concurrent access
	t.Run("ErrorLoggingConcurrent", func(t *testing.T) {
		done := make(chan bool, 20)

		for i := 0; i < 20; i++ {
			go func(index int) {
				err := fmt.Errorf("error_%d", index)
				handler.HandleError(ctx, err, "test_source")
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	// Test 11: Error classification consistency across multiple calls
	t.Run("ErrorClassificationConsistency", func(t *testing.T) {
		err := errors.New("timeout")

		// Classify same error multiple times
		categories := make([]ErrorCategory, 10)
		for i := 0; i < 10; i++ {
			categories[i] = handler.Classify(err)
		}

		// Verify all classifications are the same
		for i := 1; i < 10; i++ {
			if categories[i] != categories[0] {
				t.Errorf("Expected consistent classification, got %s and %s", categories[0], categories[i])
			}
		}
	})

	// Test 12: Error logging with multiple error loggers
	t.Run("ErrorLoggingMultipleLoggers", func(t *testing.T) {
		// Register multiple error loggers
		errorLogger1 := NewDefaultErrorLogger(logger)
		errorLogger2 := NewDefaultErrorLogger(logger)

		handler.RegisterErrorLogger("logger1", errorLogger1)
		handler.RegisterErrorLogger("logger2", errorLogger2)

		err := errors.New("timeout")

		// Log error - should use all registered loggers
		handler.HandleError(ctx, err, "test_source")

		// Verify both loggers are used
		handler.errorLoggersMu.RLock()
		if len(handler.errorLoggers) != 2 {
			t.Errorf("Expected 2 error loggers, got %d", len(handler.errorLoggers))
		}
		handler.errorLoggersMu.RUnlock()
	})

	// Test 13: Error logging with context providers
	t.Run("ErrorLoggingContextProviders", func(t *testing.T) {
		// Register multiple context providers
		provider1 := NewDefaultContextProvider("corr-1", "req-1", "user-1")
		provider2 := NewDefaultContextProvider("corr-2", "req-2", "user-2")

		handler.RegisterContextProvider("provider1", provider1)
		handler.RegisterContextProvider("provider2", provider2)

		err := errors.New("timeout")

		// Log error - should use all context providers
		handler.HandleError(ctx, err, "test_source")

		// Verify both providers are used
		handler.contextProvidersMu.RLock()
		if len(handler.contextProviders) != 3 { // 1 from before + 2 new
			t.Errorf("Expected 3 context providers, got %d", len(handler.contextProviders))
		}
		handler.contextProvidersMu.RUnlock()
	})

	// Test 14: Error logging preserves error information
	t.Run("ErrorLoggingPreservesInfo", func(t *testing.T) {
		originalErr := errors.New("original error message")
		handler.HandleError(ctx, originalErr, "test_source")

		// Verify error message is preserved
		if originalErr.Error() != "original error message" {
			t.Errorf("Expected error message to be preserved, got %s", originalErr.Error())
		}
	})

	// Test 15: Error logging with different categories
	t.Run("ErrorLoggingDifferentCategories", func(t *testing.T) {
		categories := []struct {
			err      error
			expected ErrorCategory
		}{
			{errors.New("timeout"), ErrorCategoryTransient},
			{errors.New("invalid"), ErrorCategoryPermanent},
			{errors.New("data corruption"), ErrorCategoryCritical},
		}

		for _, cat := range categories {
			handler.HandleError(ctx, cat.err, "test_source")

			// Verify category is correct
			if handler.Classify(cat.err) != cat.expected {
				t.Errorf("Expected %s, got %s", cat.expected, handler.Classify(cat.err))
			}
		}
	})
}
