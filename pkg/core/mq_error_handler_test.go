package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewMQErrorHandler(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	if eh == nil {
		t.Fatal("NewMQErrorHandler returned nil")
	}
	if eh.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", eh.maxRetries)
	}
	if eh.baseRetryDelay != 100*time.Millisecond {
		t.Errorf("baseRetryDelay = %v, want 100ms", eh.baseRetryDelay)
	}
	if eh.maxRetryDelay != 30*time.Second {
		t.Errorf("maxRetryDelay = %v, want 30s", eh.maxRetryDelay)
	}
	if eh.timeoutDuration != 5*time.Second {
		t.Errorf("timeoutDuration = %v, want 5s", eh.timeoutDuration)
	}
}

func TestCalculateBackoffDelay(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 5, 1*time.Second)

	tests := []struct {
		name    string
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{"attempt_0", 0, 1 * time.Second, 1 * time.Second},
		{"attempt_1", 1, 900 * time.Millisecond, 1 * time.Second},
		{"attempt_2", 2, 1800 * time.Millisecond, 2 * time.Second},
		{"attempt_3", 3, 3600 * time.Millisecond, 4 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := eh.CalculateBackoffDelay(tt.attempt)
			if delay < tt.min || delay > tt.max {
				t.Errorf("CalculateBackoffDelay(%d) = %v, want between %v and %v",
					tt.attempt, delay, tt.min, tt.max)
			}
		})
	}

	delay := eh.CalculateBackoffDelay(10)
	if delay > eh.maxRetryDelay {
		t.Errorf("CalculateBackoffDelay(large) = %v, exceeds max %v", delay, eh.maxRetryDelay)
	}
	if delay < eh.maxRetryDelay*9/10 {
		t.Errorf("CalculateBackoffDelay(large) = %v, too far below max %v", delay, eh.maxRetryDelay)
	}
}

func TestMQErrorHandlerSetMaxRetries(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.SetMaxRetries(10)
	if eh.maxRetries != 10 {
		t.Errorf("maxRetries = %d, want 10", eh.maxRetries)
	}
}

func TestMQErrorHandlerSetMaxRetryDelay(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.SetMaxRetryDelay(60 * time.Second)
	if eh.maxRetryDelay != 60*time.Second {
		t.Errorf("maxRetryDelay = %v, want 60s", eh.maxRetryDelay)
	}
}

func TestMQErrorHandlerSetTimeoutDuration(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.SetTimeoutDuration(10 * time.Second)
	if eh.timeoutDuration != 10*time.Second {
		t.Errorf("timeoutDuration = %v, want 10s", eh.timeoutDuration)
	}
}

func TestMQErrorHandlerGetTimeoutDuration(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	if got := eh.GetTimeoutDuration(); got != 5*time.Second {
		t.Errorf("GetTimeoutDuration() = %v, want 5s", got)
	}
}

func TestMQErrorHandlerRegisterPermanentError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.RegisterPermanentError("err_perm")
	if !eh.permanentErrorCodes["err_perm"] {
		t.Error("expected err_perm to be registered as permanent")
	}
}

func TestMQErrorHandlerRegisterTransientError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.RegisterTransientError("err_temp")
	if !eh.transientErrorCodes["err_temp"] {
		t.Error("expected err_temp to be registered as transient")
	}
}

func TestMQErrorHandlerDegradedMode(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	if eh.IsInDegradedMode() {
		t.Error("expected not in degraded mode initially")
	}
	eh.SetDegradedMode(true)
	if !eh.IsInDegradedMode() {
		t.Error("expected to be in degraded mode")
	}
	eh.SetDegradedMode(false)
	if eh.IsInDegradedMode() {
		t.Error("expected not in degraded mode after reset")
	}
}

func TestMQErrorHandlerConsecutiveErrors(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	if got := eh.GetConsecutiveErrorCount(); got != 0 {
		t.Errorf("GetConsecutiveErrorCount() = %d, want 0", got)
	}
	eh.ResetConsecutiveErrorCount()
	if got := eh.GetConsecutiveErrorCount(); got != 0 {
		t.Errorf("GetConsecutiveErrorCount() = %d, want 0", got)
	}
}

func TestMQErrorHandlerClassifyError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{"nil", nil, ErrorTypeUnknown},
		{"timeout", ErrTimeout, ErrorTypeTimeout},
		{"deadline", ErrDeadlineExceeded, ErrorTypeTimeout},
		{"connection_refused", ErrConnectionRefused, ErrorTypeConnection},
		{"connection_reset", ErrConnectionReset, ErrorTypeConnection},
		{"unauthorized", ErrUnauthorized, ErrorTypePermanent},
		{"forbidden", ErrForbidden, ErrorTypePermanent},
		{"not_found", ErrNotFound, ErrorTypePermanent},
		{"data_corruption", ErrDataCorruption, ErrorTypeCritical},
		{"critical_failure", ErrCriticalFailure, ErrorTypeCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eh.ClassifyError(tt.err); got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMQErrorHandlerGetRecoveryStats(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	stats := eh.GetRecoveryStats()
	if stats == nil {
		t.Fatal("GetRecoveryStats returned nil")
	}
	if stats["max_retries"] != 3 {
		t.Errorf("max_retries = %v, want 3", stats["max_retries"])
	}
}

func TestMQErrorHandlerClassifyError_Extended(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{"bad_request", ErrBadRequest, ErrorTypePermanent},
		{"invalid_state", ErrInvalidState, ErrorTypePermanent},
		{"auth_failed", ErrAuthFailed, ErrorTypePermanent},
		{"context_deadline", context.DeadlineExceeded, ErrorTypeTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eh.ClassifyError(tt.err); got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMQErrorHandlerClassifyError_SystemError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{"transient", NewSystemError(ErrorTypeTransient, "TEST", "msg", nil), ErrorTypeTransient},
		{"permanent", NewSystemError(ErrorTypePermanent, "TEST", "msg", nil), ErrorTypePermanent},
		{"critical", NewSystemError(ErrorTypeCritical, "TEST", "msg", nil), ErrorTypeCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eh.ClassifyError(tt.err); got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMQErrorHandlerClassifyError_RegisteredCodes(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	eh.RegisterPermanentError("perm_code")
	eh.RegisterTransientError("trans_code")

	p := errors.New("perm_code")
	if got := eh.ClassifyError(p); got != ErrorTypePermanent {
		t.Errorf("ClassifyError(registered permanent) = %v, want %v", got, ErrorTypePermanent)
	}

	tr := errors.New("trans_code")
	if got := eh.ClassifyError(tr); got != ErrorTypeTransient {
		t.Errorf("ClassifyError(registered transient) = %v, want %v", got, ErrorTypeTransient)
	}
}

func TestMQErrorHandlerClassifyError_DefaultTransient(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	if got := eh.ClassifyError(errors.New("random unknown error")); got != ErrorTypeTransient {
		t.Errorf("ClassifyError(unknown) = %v, want %v", got, ErrorTypeTransient)
	}
}

func TestMQErrorHandlerRetryWithBackoff_Success(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Millisecond)

	ctx := context.Background()
	calls := 0
	err := eh.RetryWithBackoff(ctx, func() error {
		calls++
		return nil
	}, "test-op")
	if err != nil {
		t.Fatalf("RetryWithBackoff error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestMQErrorHandlerRetryWithBackoff_RetrySuccess(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Millisecond)

	ctx := context.Background()
	calls := 0
	err := eh.RetryWithBackoff(ctx, func() error {
		calls++
		if calls < 2 {
			return errors.New("transient error")
		}
		return nil
	}, "test-op")
	if err != nil {
		t.Fatalf("RetryWithBackoff error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestMQErrorHandlerRetryWithBackoff_PermanentError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Millisecond)

	ctx := context.Background()
	calls := 0
	err := eh.RetryWithBackoff(ctx, func() error {
		calls++
		return ErrNotFound
	}, "test-op")
	if err == nil {
		t.Fatal("expected error for permanent error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestMQErrorHandlerRetryWithBackoff_AllRetriesExhausted(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Millisecond)

	ctx := context.Background()
	calls := 0
	transientErr := errors.New("transient error")
	err := eh.RetryWithBackoff(ctx, func() error {
		calls++
		return transientErr
	}, "test-op")
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestMQErrorHandlerHandleError_NilError(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	shouldRetry, err := eh.HandleError(context.Background(), nil, "test-op")
	if err != nil {
		t.Errorf("HandleError(nil) returned error: %v", err)
	}
	if shouldRetry {
		t.Error("HandleError(nil) should not retry")
	}
}

func TestMQErrorHandlerHandleError_NilErrorRecoverFromDegraded(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)
	eh.SetDegradedMode(true)

	shouldRetry, err := eh.HandleError(context.Background(), nil, "test-op")
	if err != nil {
		t.Errorf("HandleError(nil) returned error: %v", err)
	}
	if shouldRetry {
		t.Error("HandleError(nil) should not retry")
	}
	if eh.IsInDegradedMode() {
		t.Error("expected to recover from degraded mode")
	}
}

func TestMQErrorHandlerHandleError_Connection(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	shouldRetry, _ := eh.HandleError(context.Background(), ErrConnectionRefused, "test-op")
	if !shouldRetry {
		t.Error("HandleError(connection) should retry")
	}
}

func TestMQErrorHandlerHandleError_Timeout(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	shouldRetry, _ := eh.HandleError(context.Background(), ErrTimeout, "test-op")
	if !shouldRetry {
		t.Error("HandleError(timeout) should retry")
	}
}

func TestMQErrorHandlerHandleError_Permanent(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	shouldRetry, err := eh.HandleError(context.Background(), ErrNotFound, "test-op")
	if shouldRetry {
		t.Error("HandleError(permanent) should not retry")
	}
	if err == nil {
		t.Error("HandleError(permanent) should return error")
	}
}

func TestMQErrorHandlerHandleError_Transient(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 3, 1*time.Second)

	shouldRetry, _ := eh.HandleError(context.Background(), errors.New("transient err"), "test-op")
	if !shouldRetry {
		t.Error("HandleError(transient) should retry")
	}
}

func TestMQErrorHandlerRetryWithBackoff_ContextCancelled(t *testing.T) {
	t.Parallel()
	logger := NewDefaultLogger(LogLevelDebug)
	metrics := NewDefaultMetricsCollector()
	eh := NewMQErrorHandler(logger, metrics, 10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	done := make(chan error, 1)
	go func() {
		done <- eh.RetryWithBackoff(ctx, func() error {
			calls++
			return errors.New("error")
		}, "test-op")
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected context cancellation error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancellation")
	}
}
