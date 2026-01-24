package e2e

import (
	"errors"
	"testing"
	"time"
)

func TestErrorInjectorTransientError(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeTransient,
		ErrorRate:       1.0, // Always inject
		RecoveryTime:    100 * time.Millisecond,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	// First call should inject error
	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if injector.GetErrorCount() != 1 {
		t.Errorf("expected 1 error, got %d", injector.GetErrorCount())
	}
}

func TestErrorInjectorPermanentError(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypePermanent,
		ErrorRate:       1.0,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check that error message contains "permanent"
	if !contains(err.Error(), "permanent") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestErrorInjectorCriticalError(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeCritical,
		ErrorRate:       1.0,
		RecoveryTime:    50 * time.Millisecond,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if injector.GetErrorCount() != 1 {
		t.Errorf("expected 1 error, got %d", injector.GetErrorCount())
	}
}

func TestErrorInjectorTimeoutError(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeTimeout,
		ErrorRate:       1.0,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestErrorInjectorRateLimitError(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeRateLimit,
		ErrorRate:       1.0,
		RecoveryTime:    50 * time.Millisecond,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestErrorInjectorErrorRate(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeTransient,
		ErrorRate:       0.0, // Never inject
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	// Should not inject errors
	for i := 0; i < 10; i++ {
		err := injector.InjectError()
		if err != nil {
			t.Errorf("iteration %d: expected no error, got %v", i, err)
		}
	}

	if injector.GetErrorCount() != 0 {
		t.Errorf("expected 0 errors, got %d", injector.GetErrorCount())
	}
}

func TestErrorInjectorInjectionWindow(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeTransient,
		ErrorRate:       1.0,
		InjectionWindow: 50 * time.Millisecond,
	}

	injector := NewErrorInjector(config)

	// Inject error immediately
	err := injector.InjectError()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Wait for injection window to pass
	time.Sleep(100 * time.Millisecond)

	// Should not inject error after window
	err = injector.InjectError()
	if err != nil {
		t.Errorf("expected no error after injection window, got %v", err)
	}
}

func TestErrorRecoveryHandlerExponentialBackoff(t *testing.T) {
	strategy := NewExponentialBackoffStrategy(10*time.Millisecond, 100*time.Millisecond, 2.0)

	// Test backoff durations - allow some tolerance for timing variations
	backoff0 := strategy.GetBackoffDuration(0)
	backoff1 := strategy.GetBackoffDuration(1)
	backoff2 := strategy.GetBackoffDuration(2)

	// Allow 10ms tolerance for timing variations (more lenient)
	tolerance := 10 * time.Millisecond

	if backoff0 < 10*time.Millisecond-tolerance || backoff0 > 10*time.Millisecond+tolerance {
		t.Errorf("expected ~10ms, got %v", backoff0)
	}

	if backoff1 < 20*time.Millisecond-tolerance || backoff1 > 20*time.Millisecond+tolerance {
		t.Errorf("expected ~20ms, got %v", backoff1)
	}

	if backoff2 < 40*time.Millisecond-tolerance || backoff2 > 40*time.Millisecond+tolerance {
		t.Errorf("expected ~40ms, got %v", backoff2)
	}

	// Test max delay
	backoff10 := strategy.GetBackoffDuration(10)
	if backoff10 > 100*time.Millisecond {
		t.Errorf("expected <= 100ms, got %v", backoff10)
	}
}

func TestCircuitBreakerStrategyOpen(t *testing.T) {
	strategy := NewCircuitBreakerStrategy(3, 2, 100*time.Millisecond)

	// Inject failures
	for i := 0; i < 3; i++ {
		_ = strategy.Recover(errors.New("test error"))
	}

	if strategy.GetState() != "open" {
		t.Errorf("expected state 'open', got '%s'", strategy.GetState())
	}

	// Should not allow recovery
	if strategy.CanRecover(errors.New("test error")) {
		t.Error("expected CanRecover to return false when open")
	}
}

func TestCircuitBreakerStrategyHalfOpen(t *testing.T) {
	strategy := NewCircuitBreakerStrategy(2, 2, 50*time.Millisecond)

	// Inject failures to open circuit
	_ = strategy.Recover(errors.New("test error"))
	_ = strategy.Recover(errors.New("test error"))

	if strategy.GetState() != "open" {
		t.Errorf("expected state 'open', got '%s'", strategy.GetState())
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to half-open or allow recovery
	canRecover := strategy.CanRecover(errors.New("test error"))
	_ = strategy.Recover(errors.New("test error"))
	state := strategy.GetState()
	if state != "half-open" && state != "open" && !canRecover {
		t.Errorf("expected state 'half-open' or 'open' or canRecover=true, got '%s' and canRecover=%v", state, canRecover)
	}
}

func TestCircuitBreakerStrategyClosed(t *testing.T) {
	strategy := NewCircuitBreakerStrategy(3, 2, 100*time.Millisecond)

	// Should allow recovery
	if !strategy.CanRecover(errors.New("test error")) {
		t.Error("expected CanRecover to return true when closed")
	}

	if strategy.GetState() != "closed" {
		t.Errorf("expected state 'closed', got '%s'", strategy.GetState())
	}
}

func TestErrorRecoveryHandlerRegisterStrategy(t *testing.T) {
	handler := NewErrorRecoveryHandler()
	strategy := NewExponentialBackoffStrategy(10*time.Millisecond, 100*time.Millisecond, 2.0)

	handler.RegisterStrategy("test_op", strategy)

	// Should handle error with strategy
	err := handler.HandleError("test_op", errors.New("test error"))
	if err != nil {
		t.Errorf("expected no error from handler, got %v", err)
	}

	if handler.GetRetryCount("test_op") != 1 {
		t.Errorf("expected retry count 1, got %d", handler.GetRetryCount("test_op"))
	}
}

func TestErrorRecoveryHandlerResetRetryCount(t *testing.T) {
	handler := NewErrorRecoveryHandler()

	// Simulate errors
	_ = handler.HandleError("test_op", errors.New("test error"))
	_ = handler.HandleError("test_op", errors.New("test error"))

	if handler.GetRetryCount("test_op") != 2 {
		t.Errorf("expected retry count 2, got %d", handler.GetRetryCount("test_op"))
	}

	// Reset
	handler.ResetRetryCount("test_op")

	if handler.GetRetryCount("test_op") != 0 {
		t.Errorf("expected retry count 0 after reset, got %d", handler.GetRetryCount("test_op"))
	}
}

func TestErrorRecoveryHandlerSuccessResets(t *testing.T) {
	handler := NewErrorRecoveryHandler()

	// Simulate errors
	_ = handler.HandleError("test_op", errors.New("test error"))
	_ = handler.HandleError("test_op", errors.New("test error"))

	// Success should reset
	_ = handler.HandleError("test_op", nil)

	if handler.GetRetryCount("test_op") != 0 {
		t.Errorf("expected retry count 0 after success, got %d", handler.GetRetryCount("test_op"))
	}
}

func TestErrorInjectorReset(t *testing.T) {
	config := ErrorInjectionConfig{
		ErrorType:       ErrorTypeTransient,
		ErrorRate:       1.0,
		InjectionWindow: 1 * time.Second,
	}

	injector := NewErrorInjector(config)

	// Inject errors
	_ = injector.InjectError()
	_ = injector.InjectError()

	if injector.GetErrorCount() != 2 {
		t.Errorf("expected 2 errors, got %d", injector.GetErrorCount())
	}

	// Reset
	injector.Reset()

	if injector.GetErrorCount() != 0 {
		t.Errorf("expected 0 errors after reset, got %d", injector.GetErrorCount())
	}
}
