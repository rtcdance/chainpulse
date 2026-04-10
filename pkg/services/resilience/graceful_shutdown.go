package resilience

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"chainpulse/pkg/core"
)

// ShutdownHandler manages graceful shutdown of the system
type ShutdownHandler struct {
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	shutdownTimeout    time.Duration
	shutdownSignals    []os.Signal
	shutdownCallbacks  []ShutdownCallback
	callbacksMu        sync.RWMutex
	isShuttingDown     int32
	shutdownOnce       sync.Once
	shutdownChan       chan struct{}
	inFlightRequests   int64
	resourceCleanups   []ResourceCleanup
	resourceCleanupsMu sync.RWMutex
}

// ShutdownCallback is called during shutdown
type ShutdownCallback func(ctx context.Context) error

// ResourceCleanup cleans up a resource
type ResourceCleanup func(ctx context.Context) error

// NewShutdownHandler creates a new shutdown handler
func NewShutdownHandler(logger core.Logger, metricsCollector core.MetricsCollector) *ShutdownHandler {
	return &ShutdownHandler{
		logger:            logger,
		metricsCollector:  metricsCollector,
		shutdownTimeout:   30 * time.Second,
		shutdownSignals:   []os.Signal{syscall.SIGTERM, syscall.SIGINT},
		shutdownCallbacks: make([]ShutdownCallback, 0),
		resourceCleanups:  make([]ResourceCleanup, 0),
		shutdownChan:      make(chan struct{}),
	}
}

// SetShutdownTimeout sets the shutdown timeout
func (h *ShutdownHandler) SetShutdownTimeout(timeout time.Duration) {
	h.shutdownTimeout = timeout
}

// RegisterShutdownCallback registers a callback to be called during shutdown
func (h *ShutdownHandler) RegisterShutdownCallback(callback ShutdownCallback) {
	h.callbacksMu.Lock()
	defer h.callbacksMu.Unlock()
	h.shutdownCallbacks = append(h.shutdownCallbacks, callback)
}

// RegisterResourceCleanup registers a resource cleanup function
func (h *ShutdownHandler) RegisterResourceCleanup(cleanup ResourceCleanup) {
	h.resourceCleanupsMu.Lock()
	defer h.resourceCleanupsMu.Unlock()
	h.resourceCleanups = append(h.resourceCleanups, cleanup)
}

// IncrementInFlightRequests increments the in-flight request counter
func (h *ShutdownHandler) IncrementInFlightRequests() {
	atomic.AddInt64(&h.inFlightRequests, 1)
}

// DecrementInFlightRequests decrements the in-flight request counter
func (h *ShutdownHandler) DecrementInFlightRequests() {
	atomic.AddInt64(&h.inFlightRequests, -1)
}

// GetInFlightRequests returns the number of in-flight requests
func (h *ShutdownHandler) GetInFlightRequests() int64 {
	return atomic.LoadInt64(&h.inFlightRequests)
}

// IsShuttingDown returns true if the system is shutting down
func (h *ShutdownHandler) IsShuttingDown() bool {
	return atomic.LoadInt32(&h.isShuttingDown) == 1
}

// GetShutdownChan returns the shutdown channel
func (h *ShutdownHandler) GetShutdownChan() <-chan struct{} {
	return h.shutdownChan
}

// ListenForShutdownSignals listens for shutdown signals
func (h *ShutdownHandler) ListenForShutdownSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, h.shutdownSignals...)

	go func() {
		sig := <-sigChan
		h.logger.Info(fmt.Sprintf("Received shutdown signal: %v", sig))
		_ = h.Shutdown(context.Background())
	}()
}

// Shutdown initiates graceful shutdown
func (h *ShutdownHandler) Shutdown(ctx context.Context) error {
	var shutdownErr error

	h.shutdownOnce.Do(func() {
		// Mark as shutting down
		atomic.StoreInt32(&h.isShuttingDown, 1)
		h.logger.Info("Starting graceful shutdown")

		// Record shutdown start
		h.metricsCollector.RecordCounter("shutdown_initiated", 1, map[string]string{})

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, h.shutdownTimeout)
		defer cancel()

		// Wait for in-flight requests to complete
		shutdownErr = h.waitForInFlightRequests(shutdownCtx)
		if shutdownErr != nil {
			h.logger.Warn(fmt.Sprintf("Timeout waiting for in-flight requests: %v", shutdownErr))
		}

		// Execute shutdown callbacks
		callbackErr := h.executeShutdownCallbacks(shutdownCtx)
		if callbackErr != nil {
			h.logger.Warn(fmt.Sprintf("Error executing shutdown callbacks: %v", callbackErr))
			if shutdownErr == nil {
				shutdownErr = callbackErr
			}
		}

		// Clean up resources
		cleanupErr := h.cleanupResources(shutdownCtx)
		if cleanupErr != nil {
			h.logger.Warn(fmt.Sprintf("Error cleaning up resources: %v", cleanupErr))
			if shutdownErr == nil {
				shutdownErr = cleanupErr
			}
		}

		// Close shutdown channel
		close(h.shutdownChan)

		// Record shutdown complete
		h.metricsCollector.RecordCounter("shutdown_complete", 1, map[string]string{})
		h.logger.Info("Graceful shutdown complete")
	})

	return shutdownErr
}

// waitForInFlightRequests waits for all in-flight requests to complete
func (h *ShutdownHandler) waitForInFlightRequests(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		inFlight := h.GetInFlightRequests()
		if inFlight == 0 {
			h.logger.Info("All in-flight requests completed")
			return nil
		}

		h.logger.Debug(fmt.Sprintf("Waiting for %d in-flight requests to complete", inFlight))

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for in-flight requests (remaining: %d)", inFlight)
		case <-ticker.C:
			// Continue waiting
		}
	}
}

// executeShutdownCallbacks executes all shutdown callbacks
func (h *ShutdownHandler) executeShutdownCallbacks(ctx context.Context) error {
	h.callbacksMu.RLock()
	callbacks := make([]ShutdownCallback, len(h.shutdownCallbacks))
	copy(callbacks, h.shutdownCallbacks)
	h.callbacksMu.RUnlock()

	var lastErr error

	for i, callback := range callbacks {
		h.logger.Debug(fmt.Sprintf("Executing shutdown callback %d/%d", i+1, len(callbacks)))

		err := callback(ctx)
		if err != nil {
			h.logger.Warn(fmt.Sprintf("Shutdown callback %d failed: %v", i+1, err))
			lastErr = err
		}
	}

	return lastErr
}

// cleanupResources cleans up all registered resources
func (h *ShutdownHandler) cleanupResources(ctx context.Context) error {
	h.resourceCleanupsMu.RLock()
	cleanups := make([]ResourceCleanup, len(h.resourceCleanups))
	copy(cleanups, h.resourceCleanups)
	h.resourceCleanupsMu.RUnlock()

	var lastErr error

	for i, cleanup := range cleanups {
		h.logger.Debug(fmt.Sprintf("Cleaning up resource %d/%d", i+1, len(cleanups)))

		err := cleanup(ctx)
		if err != nil {
			h.logger.Warn(fmt.Sprintf("Resource cleanup %d failed: %v", i+1, err))
			lastErr = err
		}
	}

	return lastErr
}

// WaitForShutdown waits for shutdown to complete
func (h *ShutdownHandler) WaitForShutdown() {
	<-h.shutdownChan
}

// ShutdownManager manages shutdown for multiple components
type ShutdownManager struct {
	handlers map[string]*ShutdownHandler
	mu       sync.RWMutex
	logger   core.Logger
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(logger core.Logger) *ShutdownManager {
	return &ShutdownManager{
		handlers: make(map[string]*ShutdownHandler),
		logger:   logger,
	}
}

// RegisterHandler registers a shutdown handler
func (m *ShutdownManager) RegisterHandler(name string, handler *ShutdownHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[name] = handler
}

// UnregisterHandler unregisters a shutdown handler
func (m *ShutdownManager) UnregisterHandler(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.handlers, name)
}

// ShutdownAll shuts down all registered handlers
func (m *ShutdownManager) ShutdownAll(ctx context.Context) error {
	m.mu.RLock()
	handlers := make(map[string]*ShutdownHandler)
	for name, handler := range m.handlers {
		handlers[name] = handler
	}
	m.mu.RUnlock()

	var lastErr error

	for name, handler := range handlers {
		m.logger.Info(fmt.Sprintf("Shutting down handler: %s", name))

		err := handler.Shutdown(ctx)
		if err != nil {
			m.logger.Warn(fmt.Sprintf("Handler %s shutdown failed: %v", name, err))
			lastErr = err
		}
	}

	return lastErr
}

// GracefulShutdownContext provides context for graceful shutdown
type GracefulShutdownContext struct {
	handler *ShutdownHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewGracefulShutdownContext creates a new graceful shutdown context
func NewGracefulShutdownContext(handler *ShutdownHandler) *GracefulShutdownContext {
	ctx, cancel := context.WithCancel(context.Background())

	return &GracefulShutdownContext{
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// GetContext returns the context
func (g *GracefulShutdownContext) GetContext() context.Context {
	return g.ctx
}

// Cancel cancels the context
func (g *GracefulShutdownContext) Cancel() {
	g.cancel()
}

// IsShuttingDown returns true if shutting down
func (g *GracefulShutdownContext) IsShuttingDown() bool {
	return g.handler.IsShuttingDown()
}

// WaitForShutdown waits for shutdown
func (g *GracefulShutdownContext) WaitForShutdown() {
	g.handler.WaitForShutdown()
}
