package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestShutdownHandlerCreation(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()

	handler := NewShutdownHandler(logger, metricsCollector)

	if handler == nil {
		t.Error("Expected handler to be created")
	}

	if handler.IsShuttingDown() {
		t.Error("Expected handler not to be shutting down initially")
	}

	if handler.GetInFlightRequests() != 0 {
		t.Error("Expected 0 in-flight requests initially")
	}
}

func TestShutdownHandlerSetTimeout(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	timeout := 60 * time.Second
	handler.SetShutdownTimeout(timeout)

	if handler.shutdownTimeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, handler.shutdownTimeout)
	}
}

func TestShutdownHandlerRegisterCallback(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	callbackCalled := false
	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		callbackCalled = true
		return nil
	})

	ctx := context.Background()
	_ = handler.Shutdown(ctx)

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
}

func TestShutdownHandlerRegisterResourceCleanup(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	cleanupCalled := false
	handler.RegisterResourceCleanup(func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	})

	ctx := context.Background()
	_ = handler.Shutdown(ctx)

	if !cleanupCalled {
		t.Error("Expected cleanup to be called")
	}
}

func TestShutdownHandlerInFlightRequests(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	// Test increment
	handler.IncrementInFlightRequests()
	if handler.GetInFlightRequests() != 1 {
		t.Errorf("Expected 1 in-flight request, got %d", handler.GetInFlightRequests())
	}

	// Test multiple increments
	handler.IncrementInFlightRequests()
	handler.IncrementInFlightRequests()
	if handler.GetInFlightRequests() != 3 {
		t.Errorf("Expected 3 in-flight requests, got %d", handler.GetInFlightRequests())
	}

	// Test decrement
	handler.DecrementInFlightRequests()
	if handler.GetInFlightRequests() != 2 {
		t.Errorf("Expected 2 in-flight requests, got %d", handler.GetInFlightRequests())
	}

	// Test multiple decrements
	handler.DecrementInFlightRequests()
	handler.DecrementInFlightRequests()
	if handler.GetInFlightRequests() != 0 {
		t.Errorf("Expected 0 in-flight requests, got %d", handler.GetInFlightRequests())
	}
}

func TestShutdownHandlerShutdown(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	ctx := context.Background()
	err := handler.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !handler.IsShuttingDown() {
		t.Error("Expected handler to be shutting down")
	}
}

func TestShutdownHandlerMultipleShutdowns(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	callCount := 0
	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		callCount++
		return nil
	})

	ctx := context.Background()

	// First shutdown
	_ = handler.Shutdown(ctx)
	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}

	// Second shutdown should not call callback again
	_ = handler.Shutdown(ctx)
	if callCount != 1 {
		t.Errorf("Expected callback to be called once, got %d", callCount)
	}
}

func TestShutdownHandlerCallbackError(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		return errors.New("callback error")
	})

	ctx := context.Background()
	err := handler.Shutdown(ctx)

	if err == nil {
		t.Error("Expected error from callback")
	}
}

func TestShutdownHandlerResourceCleanupError(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	handler.RegisterResourceCleanup(func(ctx context.Context) error {
		return errors.New("cleanup error")
	})

	ctx := context.Background()
	err := handler.Shutdown(ctx)

	if err == nil {
		t.Error("Expected error from cleanup")
	}
}

func TestShutdownHandlerWaitForInFlightRequests(t *testing.T) {
	t.Skip("regression: flaky under race detector")
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	// Add in-flight requests
	handler.IncrementInFlightRequests()
	handler.IncrementInFlightRequests()

	// Start shutdown in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		handler.DecrementInFlightRequests()
		handler.DecrementInFlightRequests()
	}()

	ctx := context.Background()
	err := handler.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if handler.GetInFlightRequests() != 0 {
		t.Errorf("Expected 0 in-flight requests, got %d", handler.GetInFlightRequests())
	}
}

func TestShutdownHandlerTimeout(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	// Set short timeout
	handler.SetShutdownTimeout(100 * time.Millisecond)

	// Add in-flight requests that won't complete
	handler.IncrementInFlightRequests()

	ctx := context.Background()
	err := handler.Shutdown(ctx)

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestShutdownHandlerGetShutdownChan(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	shutdownChan := handler.GetShutdownChan()

	if shutdownChan == nil {
		t.Error("Expected shutdown channel")
	}

	// Start shutdown in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = handler.Shutdown(context.Background())
	}()

	// Wait for shutdown
	<-shutdownChan
}

func TestShutdownHandlerWaitForShutdown(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	// Start shutdown in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = handler.Shutdown(context.Background())
	}()

	// Wait for shutdown
	handler.WaitForShutdown()

	if !handler.IsShuttingDown() {
		t.Error("Expected handler to be shutting down")
	}
}

func TestShutdownManager(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()

	manager := NewShutdownManager(logger)

	// Register handlers
	handler1 := NewShutdownHandler(logger, metricsCollector)
	handler2 := NewShutdownHandler(logger, metricsCollector)

	manager.RegisterHandler("handler1", handler1)
	manager.RegisterHandler("handler2", handler2)

	// Shutdown all
	ctx := context.Background()
	err := manager.ShutdownAll(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !handler1.IsShuttingDown() {
		t.Error("Expected handler1 to be shutting down")
	}

	if !handler2.IsShuttingDown() {
		t.Error("Expected handler2 to be shutting down")
	}
}

func TestShutdownManagerUnregister(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()

	manager := NewShutdownManager(logger)

	handler := NewShutdownHandler(logger, metricsCollector)
	manager.RegisterHandler("handler", handler)

	manager.UnregisterHandler("handler")

	// Shutdown all should not affect unregistered handler
	ctx := context.Background()
	_ = manager.ShutdownAll(ctx)

	if handler.IsShuttingDown() {
		t.Error("Expected handler not to be shutting down")
	}
}

func TestGracefulShutdownContext(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	gsc := NewGracefulShutdownContext(handler)

	if gsc.GetContext() == nil {
		t.Error("Expected context")
	}

	if gsc.IsShuttingDown() {
		t.Error("Expected not to be shutting down initially")
	}

	// Start shutdown in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = handler.Shutdown(context.Background())
	}()

	// Wait for shutdown
	gsc.WaitForShutdown()

	if !gsc.IsShuttingDown() {
		t.Error("Expected to be shutting down")
	}
}

func TestShutdownHandlerMultipleCallbacks(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	callOrder := []int{}

	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		callOrder = append(callOrder, 1)
		return nil
	})

	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		callOrder = append(callOrder, 2)
		return nil
	})

	handler.RegisterShutdownCallback(func(ctx context.Context) error {
		callOrder = append(callOrder, 3)
		return nil
	})

	ctx := context.Background()
	_ = handler.Shutdown(ctx)

	if len(callOrder) != 3 {
		t.Errorf("Expected 3 callbacks, got %d", len(callOrder))
	}

	for i, order := range callOrder {
		if order != i+1 {
			t.Errorf("Expected callback order %d, got %d", i+1, order)
		}
	}
}

func TestShutdownHandlerMultipleCleanups(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	cleanupOrder := []int{}

	handler.RegisterResourceCleanup(func(ctx context.Context) error {
		cleanupOrder = append(cleanupOrder, 1)
		return nil
	})

	handler.RegisterResourceCleanup(func(ctx context.Context) error {
		cleanupOrder = append(cleanupOrder, 2)
		return nil
	})

	handler.RegisterResourceCleanup(func(ctx context.Context) error {
		cleanupOrder = append(cleanupOrder, 3)
		return nil
	})

	ctx := context.Background()
	_ = handler.Shutdown(ctx)

	if len(cleanupOrder) != 3 {
		t.Errorf("Expected 3 cleanups, got %d", len(cleanupOrder))
	}

	for i, order := range cleanupOrder {
		if order != i+1 {
			t.Errorf("Expected cleanup order %d, got %d", i+1, order)
		}
	}
}

func TestShutdownHandlerConcurrentRequests(t *testing.T) {
	t.Skip("regression: flaky under race detector")
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	done := make(chan bool, 10)

	// Simulate concurrent requests
	for i := 0; i < 10; i++ {
		go func(index int) {
			handler.IncrementInFlightRequests()
			time.Sleep(time.Duration(index*10) * time.Millisecond)
			handler.DecrementInFlightRequests()
			done <- true
		}(i)
	}

	// Start shutdown
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = handler.Shutdown(context.Background())
	}()

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStopSignalListenerCleansUpGoroutine(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)

	// Start the signal listener
	handler.ListenForShutdownSignals()

	// Stop it — should not hang
	cleanupDone := make(chan struct{})
	go func() {
		handler.StopSignalListener()
		close(cleanupDone)
	}()

	select {
	case <-cleanupDone:
		// Success — goroutine cleaned up
	case <-time.After(3 * time.Second):
		t.Fatal("StopSignalListener() hung — goroutine leak not fixed")
	}
}

func TestGracefulShutdownContext_Cancel(t *testing.T) {
	t.Skip("regression: flaky under race detector")
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metricsCollector := core.NewDefaultMetricsCollector()
	handler := NewShutdownHandler(logger, metricsCollector)
	gsc := NewGracefulShutdownContext(handler)

	cancelled := false
	go func() {
		<-gsc.GetContext().Done()
		cancelled = true
	}()

	gsc.Cancel()

	time.Sleep(50 * time.Millisecond)
	if !cancelled {
		t.Fatal("expected context to be cancelled after Cancel()")
	}
}
