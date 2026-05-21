package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockMetricsCollector for testing
type MockMetricsCollector struct {
	counters map[string]int64
}

func (mmc *MockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	if mmc.counters == nil {
		mmc.counters = make(map[string]int64)
	}
	mmc.counters[name] += value
}

func (mmc *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
}

func (mmc *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
}

func (mmc *MockMetricsCollector) GetMetrics() map[string]any {
	return map[string]any{}
}

// TestNewMQErrorHandler tests handler creation
func TestNewMQErrorHandler(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}

	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	assert.NotNil(t, handler)
	assert.Equal(t, 3, handler.maxRetries)
	assert.Equal(t, 100*time.Millisecond, handler.baseRetryDelay)
	assert.Equal(t, 30*time.Second, handler.maxRetryDelay)
	assert.Equal(t, 5*time.Second, handler.timeoutDuration)
	assert.False(t, handler.degradedMode)
}

// TestClassifyErrorNilMQ tests classifying nil error
func TestClassifyErrorNilMQ(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	errorType := handler.ClassifyError(nil)

	assert.Equal(t, ErrorType("unknown"), errorType)
}

// TestClassifyErrorTimeout tests classifying timeout error
func TestClassifyErrorTimeout(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	err := ErrTimeout
	errorType := handler.ClassifyError(err)

	assert.Equal(t, ErrorType("timeout"), errorType)
}

// TestClassifyErrorConnection tests classifying connection error
func TestClassifyErrorConnection(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	tests := []struct {
		name    string
		err     error
		wantErr ErrorType
	}{
		{"connection refused", ErrConnectionRefused, ErrorType("connection")},
		{"connection reset", ErrConnectionReset, ErrorType("connection")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := handler.ClassifyError(tt.err)
			assert.Equal(t, tt.wantErr, errorType)
		})
	}
}

// TestClassifyErrorPermanent tests classifying permanent error
func TestClassifyErrorPermanent(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.RegisterPermanentError("invalid_message")

	err := errors.New("invalid_message")
	errorType := handler.ClassifyError(err)

	assert.Equal(t, ErrorTypePermanent, errorType)
}

// TestClassifyErrorTransient tests classifying transient error
func TestClassifyErrorTransient(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.RegisterTransientError("temporary_failure")

	err := errors.New("temporary_failure")
	errorType := handler.ClassifyError(err)

	assert.Equal(t, ErrorTypeTransient, errorType)
}

// TestHandleErrorNil tests handling nil error
func TestHandleErrorNil(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	shouldRetry, err := handler.HandleError(ctx, nil, "test_op")

	assert.False(t, shouldRetry)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), handler.consecutiveErrors)
}

// TestHandleErrorConnection tests handling connection error
func TestHandleErrorConnection(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	err := errors.New("connection refused")
	shouldRetry, _ := handler.HandleError(ctx, err, "test_op")

	assert.True(t, shouldRetry)
	assert.Equal(t, int64(1), handler.consecutiveErrors)
}

// TestHandleErrorPermanent tests handling permanent error
func TestHandleErrorPermanent(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	handler.RegisterPermanentError("invalid_message")

	err := errors.New("invalid_message")
	shouldRetry, returnErr := handler.HandleError(ctx, err, "test_op")

	assert.False(t, shouldRetry)
	assert.Error(t, returnErr)
	assert.True(t, handler.degradedMode)
}

// TestHandleErrorRecovery tests error recovery
func TestHandleErrorRecovery(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	// Simulate error then recovery
	handler.degradedMode = true
	_, _ = handler.HandleError(ctx, nil, "test_op")

	assert.False(t, handler.degradedMode)
}

// TestRetryWithBackoffSuccessMQ tests successful retry
func TestRetryWithBackoffSuccessMQ(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) < 2 {
			return errors.New("connection refused")
		}
		return nil
	}

	err := handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	assert.Equal(t, int64(1), handler.successfulRecoveries)
}

// TestRetryWithBackoffExhausted tests exhausted retries
func TestRetryWithBackoffExhausted(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 2, 10*time.Millisecond)
	ctx := context.Background()

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("connection refused")
	}

	err := handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.Error(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	assert.True(t, handler.degradedMode)
}

// TestRetryWithBackoffPermanentErrorMQ tests permanent error no retry
func TestRetryWithBackoffPermanentErrorMQ(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	handler.RegisterPermanentError("invalid_message")

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("invalid_message")
	}

	err := handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

// TestRetryWithBackoffContextCanceledMQ tests context cancellation
func TestRetryWithBackoffContextCanceledMQ(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 5, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("connection refused")
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.Error(t, err)
	assert.Less(t, atomic.LoadInt32(&attempts), int32(5))
}

// TestCalculateBackoffDelay tests backoff delay calculation
func TestCalculateBackoffDelay(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	tests := []struct {
		name     string
		attempt  int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{"attempt 1", 1, 50 * time.Millisecond, 150 * time.Millisecond},
		{"attempt 2", 2, 150 * time.Millisecond, 350 * time.Millisecond},
		{"attempt 3", 3, 350 * time.Millisecond, 750 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := handler.CalculateBackoffDelay(tt.attempt)
			assert.GreaterOrEqual(t, delay, tt.minDelay)
			assert.LessOrEqual(t, delay, tt.maxDelay)
		})
	}
}

// TestCalculateBackoffDelayCapped tests backoff delay capping
func TestCalculateBackoffDelayCapped(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 10, 1*time.Second)
	handler.maxRetryDelay = 5 * time.Second

	delay := handler.CalculateBackoffDelay(10)

	assert.LessOrEqual(t, delay, 5*time.Second)
}

// TestRegisterPermanentError tests registering permanent error
func TestRegisterPermanentError(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.RegisterPermanentError("invalid_message")

	assert.True(t, handler.permanentErrorCodes["invalid_message"])
}

// TestRegisterTransientError tests registering transient error
func TestRegisterTransientError(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.RegisterTransientError("temporary_failure")

	assert.True(t, handler.transientErrorCodes["temporary_failure"])
}

// TestIsInDegradedMode tests degraded mode check
func TestIsInDegradedMode(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	assert.False(t, handler.IsInDegradedMode())

	handler.SetDegradedMode(true)

	assert.True(t, handler.IsInDegradedMode())
}

// TestSetDegradedMode tests setting degraded mode
func TestSetDegradedMode(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.SetDegradedMode(true)
	assert.True(t, handler.degradedMode)

	handler.SetDegradedMode(false)
	assert.False(t, handler.degradedMode)
}

// TestGetConsecutiveErrorCount tests getting consecutive error count
func TestGetConsecutiveErrorCount(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	assert.Equal(t, int64(0), handler.GetConsecutiveErrorCount())

	err := errors.New("connection refused")
	_, _ = handler.HandleError(ctx, err, "test_op")

	assert.Equal(t, int64(1), handler.GetConsecutiveErrorCount())
}

// TestResetConsecutiveErrorCount tests resetting consecutive error count
func TestResetConsecutiveErrorCount(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)
	ctx := context.Background()

	err := errors.New("connection refused")
	_, _ = handler.HandleError(ctx, err, "test_op")
	assert.Equal(t, int64(1), handler.GetConsecutiveErrorCount())

	handler.ResetConsecutiveErrorCount()
	assert.Equal(t, int64(0), handler.GetConsecutiveErrorCount())
}

// TestGetRecoveryStats tests getting recovery statistics
func TestGetRecoveryStats(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	stats := handler.GetRecoveryStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "recovery_attempts")
	assert.Contains(t, stats, "successful_recoveries")
	assert.Contains(t, stats, "consecutive_errors")
	assert.Contains(t, stats, "degraded_mode")
	assert.Contains(t, stats, "max_retries")
}

// TestSetTimeoutDuration tests setting timeout duration
func TestSetTimeoutDuration(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.SetTimeoutDuration(10 * time.Second)

	assert.Equal(t, 10*time.Second, handler.GetTimeoutDuration())
}

// TestSetMaxRetries tests setting max retries
func TestSetMaxRetries(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.SetMaxRetries(5)

	assert.Equal(t, 5, handler.maxRetries)
}

// TestSetMaxRetryDelay tests setting max retry delay
func TestSetMaxRetryDelay(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 100*time.Millisecond)

	handler.SetMaxRetryDelay(60 * time.Second)

	assert.Equal(t, 60*time.Second, handler.maxRetryDelay)
}

// TestMultipleErrorTypes tests handling multiple error types
func TestMultipleErrorTypes(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	handler.RegisterPermanentError("invalid")
	handler.RegisterTransientError("temporary")

	tests := []struct {
		name        string
		errMsg      string
		shouldRetry bool
	}{
		{"connection error", "connection refused", true},
		{"timeout error", "context deadline exceeded", true},
		{"permanent error", "invalid", false},
		{"transient error", "temporary", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			shouldRetry, _ := handler.HandleError(ctx, err, "test_op")
			assert.Equal(t, tt.shouldRetry, shouldRetry)
		})
	}
}

// TestErrorMetricsRecording tests error metrics recording
func TestErrorMetricsRecording(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	err := errors.New("connection refused")
	_, _ = handler.HandleError(ctx, err, "test_op")

	assert.Greater(t, metrics.counters["mq_errors"], int64(0))
}

// TestConsecutiveErrorsTracking tests tracking consecutive errors
func TestConsecutiveErrorsTracking(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	err := errors.New("connection refused")

	for i := 1; i <= 3; i++ {
		_, _ = handler.HandleError(ctx, err, "test_op")
		assert.Equal(t, int64(i), handler.GetConsecutiveErrorCount())
	}

	// Reset on success
	_, _ = handler.HandleError(ctx, nil, "test_op")
	assert.Equal(t, int64(0), handler.GetConsecutiveErrorCount())
}

// TestBackoffDelayIncreases tests that backoff delay increases with attempts
func TestBackoffDelayIncreases(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 5, 100*time.Millisecond)

	delay1 := handler.CalculateBackoffDelay(1)
	delay2 := handler.CalculateBackoffDelay(2)
	delay3 := handler.CalculateBackoffDelay(3)

	assert.Less(t, delay1, delay2)
	assert.Less(t, delay2, delay3)
}

// TestRetrySuccessfulAfterMultipleAttempts tests successful retry after multiple attempts
func TestRetrySuccessfulAfterMultipleAttempts(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 5, 10*time.Millisecond)
	ctx := context.Background()

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) < 4 {
			return errors.New("connection refused")
		}
		return nil
	}

	err := handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.NoError(t, err)
	assert.Equal(t, int32(4), atomic.LoadInt32(&attempts))
	assert.Equal(t, int64(1), handler.successfulRecoveries)
}

// TestDegradedModeOnPermanentError tests degraded mode on permanent error
func TestDegradedModeOnPermanentError(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	handler.RegisterPermanentError("invalid_message")

	err := errors.New("invalid_message")
	_, _ = handler.HandleError(ctx, err, "test_op")

	assert.True(t, handler.IsInDegradedMode())
}

// TestRecoveryAttemptsTracking tests tracking recovery attempts
func TestRecoveryAttemptsTracking(t *testing.T) {
	t.Parallel()
	logger := &EventBusTestLogger{}
	metrics := &MockMetricsCollector{}
	handler := NewMQErrorHandler(logger, metrics, 3, 10*time.Millisecond)
	ctx := context.Background()

	var attempts int32
	operation := func() error {
		atomic.AddInt32(&attempts, 1)
		if atomic.LoadInt32(&attempts) < 2 {
			return errors.New("connection refused")
		}
		return nil
	}

	_ = handler.RetryWithBackoff(ctx, operation, "test_op")

	assert.Greater(t, handler.recoveryAttempts, int64(0))
}
