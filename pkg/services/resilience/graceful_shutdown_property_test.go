package resilience

import (
	"chainpulse/pkg/core"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Property 19: Graceful Shutdown
// For any system component, graceful shutdown SHALL complete all in-flight requests,
// execute shutdown callbacks, and clean up resources before terminating

func TestProperty19GracefulShutdown(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()

	// Test 1: All in-flight requests complete before shutdown
	t.Run("InFlightRequestsComplete", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		// Simulate in-flight requests
		var requestsCompleted int
		var mu sync.Mutex
		for i := 0; i < 5; i++ {
			go func(index int) {
				handler.IncrementInFlightRequests()
				time.Sleep(50 * time.Millisecond)
				handler.DecrementInFlightRequests()
				mu.Lock()
				requestsCompleted++
				mu.Unlock()
			}(i)
		}

		// Start shutdown
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = handler.Shutdown(context.Background())
		}()

		// Wait for shutdown
		handler.WaitForShutdown()

		// Verify all requests completed
		if handler.GetInFlightRequests() != 0 {
			t.Errorf("Expected 0 in-flight requests, got %d", handler.GetInFlightRequests())
		}
	})

	// Test 2: Shutdown callbacks are executed
	t.Run("ShutdownCallbacksExecuted", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		callbacksExecuted := 0
		for i := 0; i < 3; i++ {
			handler.RegisterShutdownCallback(func(ctx context.Context) error {
				callbacksExecuted++
				return nil
			})
		}

		ctx := context.Background()
		_ = handler.Shutdown(ctx)

		if callbacksExecuted != 3 {
			t.Errorf("Expected 3 callbacks executed, got %d", callbacksExecuted)
		}
	})

	// Test 3: Resources are cleaned up
	t.Run("ResourcesCleanedUp", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		resourcesCleanedUp := 0
		for i := 0; i < 3; i++ {
			handler.RegisterResourceCleanup(func(ctx context.Context) error {
				resourcesCleanedUp++
				return nil
			})
		}

		ctx := context.Background()
		_ = handler.Shutdown(ctx)

		if resourcesCleanedUp != 3 {
			t.Errorf("Expected 3 resources cleaned up, got %d", resourcesCleanedUp)
		}
	})

	// Test 4: Shutdown completes within timeout
	t.Run("ShutdownCompletesWithinTimeout", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)
		handler.SetShutdownTimeout(1 * time.Second)

		// Add in-flight requests that complete quickly
		handler.IncrementInFlightRequests()
		handler.IncrementInFlightRequests()

		go func() {
			time.Sleep(100 * time.Millisecond)
			handler.DecrementInFlightRequests()
			handler.DecrementInFlightRequests()
		}()

		startTime := time.Now()
		ctx := context.Background()
		err := handler.Shutdown(ctx)
		duration := time.Since(startTime)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if duration > 2*time.Second {
			t.Errorf("Expected shutdown to complete within 2 seconds, took %v", duration)
		}
	})

	// Test 5: Shutdown is idempotent
	t.Run("ShutdownIsIdempotent", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		callCount := 0
		handler.RegisterShutdownCallback(func(ctx context.Context) error {
			callCount++
			return nil
		})

		ctx := context.Background()

		// Multiple shutdown calls
		if err := handler.Shutdown(ctx); err != nil {
			t.Logf("First shutdown error: %v", err)
		}
		if err := handler.Shutdown(ctx); err != nil {
			t.Logf("Second shutdown error: %v", err)
		}
		if err := handler.Shutdown(ctx); err != nil {
			t.Logf("Third shutdown error: %v", err)
		}

		// Callback should only be called once
		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}
	})

	// Test 6: Shutdown prevents new requests
	t.Run("ShutdownPreventsNewRequests", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		// Start shutdown
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := handler.Shutdown(context.Background()); err != nil {
				t.Logf("Shutdown error: %v", err)
			}
		}()

		// Wait for shutdown to start
		time.Sleep(150 * time.Millisecond)

		// Check if shutting down
		if !handler.IsShuttingDown() {
			t.Error("Expected handler to be shutting down")
		}
	})

	// Test 7: Shutdown handles callback errors gracefully
	t.Run("ShutdownHandlesCallbackErrors", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		handler.RegisterShutdownCallback(func(ctx context.Context) error {
			return errors.New("callback error")
		})

		handler.RegisterShutdownCallback(func(ctx context.Context) error {
			return nil
		})

		ctx := context.Background()
		err := handler.Shutdown(ctx)

		// Should return error but continue with other callbacks
		if err == nil {
			t.Error("Expected error from callback")
		}
	})

	// Test 8: Shutdown handles cleanup errors gracefully
	t.Run("ShutdownHandlesCleanupErrors", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		handler.RegisterResourceCleanup(func(ctx context.Context) error {
			return errors.New("cleanup error")
		})

		handler.RegisterResourceCleanup(func(ctx context.Context) error {
			return nil
		})

		ctx := context.Background()
		err := handler.Shutdown(ctx)

		// Should return error but continue with other cleanups
		if err == nil {
			t.Error("Expected error from cleanup")
		}
	})

	// Test 9: Shutdown with concurrent in-flight requests
	t.Run("ShutdownWithConcurrentRequests", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		done := make(chan bool, 20)

		// Simulate concurrent requests
		for i := 0; i < 20; i++ {
			go func(index int) {
				handler.IncrementInFlightRequests()
				time.Sleep(time.Duration(index*5) * time.Millisecond)
				handler.DecrementInFlightRequests()
				done <- true
			}(i)
		}

		// Start shutdown
		go func() {
			time.Sleep(50 * time.Millisecond)
			if err := handler.Shutdown(context.Background()); err != nil {
				t.Logf("Shutdown error: %v", err)
			}
		}()

		// Wait for all requests
		for i := 0; i < 20; i++ {
			<-done
		}

		if handler.GetInFlightRequests() != 0 {
			t.Errorf("Expected 0 in-flight requests, got %d", handler.GetInFlightRequests())
		}
	})

	// Test 10: Shutdown manager coordinates multiple handlers
	t.Run("ShutdownManagerCoordinatesHandlers", func(t *testing.T) {
		manager := NewShutdownManager(logger)

		handlers := make([]*ShutdownHandler, 5)
		for i := 0; i < 5; i++ {
			handlers[i] = NewShutdownHandler(logger, metricsCollector)
			manager.RegisterHandler(fmt.Sprintf("handler_%d", i), handlers[i])
		}

		ctx := context.Background()
		err := manager.ShutdownAll(ctx)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify all handlers are shutting down
		for i, handler := range handlers {
			if !handler.IsShuttingDown() {
				t.Errorf("Expected handler %d to be shutting down", i)
			}
		}
	})

	// Test 11: Graceful shutdown context
	t.Run("GracefulShutdownContext", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)
		gsc := NewGracefulShutdownContext(handler)

		if gsc.IsShuttingDown() {
			t.Error("Expected not to be shutting down initially")
		}

		// Start shutdown
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = handler.Shutdown(context.Background())
		}()

		// Wait for shutdown
		gsc.WaitForShutdown()

		if !gsc.IsShuttingDown() {
			t.Error("Expected to be shutting down")
		}
	})

	// Test 12: Shutdown preserves error information
	t.Run("ShutdownPreservesErrorInfo", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		expectedErr := errors.New("specific error")
		handler.RegisterShutdownCallback(func(ctx context.Context) error {
			return expectedErr
		})

		ctx := context.Background()
		err := handler.Shutdown(ctx)

		if err == nil {
			t.Error("Expected error to be returned")
		}

		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error message %s, got %s", expectedErr.Error(), err.Error())
		}
	})

	// Test 13: Shutdown with timeout context
	t.Run("ShutdownWithTimeoutContext", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)
		handler.SetShutdownTimeout(500 * time.Millisecond)

		// Add in-flight request that won't complete
		handler.IncrementInFlightRequests()

		ctx := context.Background()
		startTime := time.Now()
		err := handler.Shutdown(ctx)
		duration := time.Since(startTime)

		if err == nil {
			t.Error("Expected timeout error")
		}

		// Should timeout around 500ms
		if duration < 400*time.Millisecond || duration > 1*time.Second {
			t.Errorf("Expected timeout around 500ms, got %v", duration)
		}
	})

	// Test 14: Shutdown channel closes after shutdown
	t.Run("ShutdownChannelCloses", func(t *testing.T) {
		handler := NewShutdownHandler(logger, metricsCollector)

		shutdownChan := handler.GetShutdownChan()

		// Start shutdown
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = handler.Shutdown(context.Background())
		}()

		// Wait for channel to close
		<-shutdownChan

		// Try to receive again - should not block
		select {
		case <-shutdownChan:
			// Channel is closed
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected channel to be closed")
		}
	})

	// Test 15: Shutdown with multiple managers
	t.Run("ShutdownWithMultipleManagers", func(t *testing.T) {
		manager1 := NewShutdownManager(logger)
		manager2 := NewShutdownManager(logger)

		handler1 := NewShutdownHandler(logger, metricsCollector)
		handler2 := NewShutdownHandler(logger, metricsCollector)

		manager1.RegisterHandler("handler1", handler1)
		manager2.RegisterHandler("handler2", handler2)

		ctx := context.Background()
		if err := manager1.ShutdownAll(ctx); err != nil {
			t.Logf("Manager1 shutdown error: %v", err)
		}
		if err := manager2.ShutdownAll(ctx); err != nil {
			t.Logf("Manager2 shutdown error: %v", err)
		}

		if !handler1.IsShuttingDown() {
			t.Error("Expected handler1 to be shutting down")
		}

		if !handler2.IsShuttingDown() {
			t.Error("Expected handler2 to be shutting down")
		}
	})
}
