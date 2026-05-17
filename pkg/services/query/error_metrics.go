package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// ErrorMetricsCollector collects comprehensive error metrics
type ErrorMetricsCollector interface {
	// Initialize initializes the metrics collector
	Initialize(ctx context.Context) error
	// RecordError records an error occurrence
	RecordError(ctx context.Context, errorType string, source string, duration time.Duration)
	// RecordRetryAttempt records a retry attempt
	RecordRetryAttempt(ctx context.Context, attempt int, success bool)
	// RecordCircuitBreakerStateChange records a circuit breaker state change
	RecordCircuitBreakerStateChange(ctx context.Context, breaker string, oldState string, newState string)
	// RecordConsistencyCheck records a consistency check result
	RecordConsistencyCheck(ctx context.Context, passed bool, duration time.Duration)
	// RecordDegradationEvent records a degradation event
	RecordDegradationEvent(ctx context.Context, mode string, reason string)
	// GetErrorMetrics returns current error metrics
	GetErrorMetrics(ctx context.Context) *ErrorMetrics
	// GetRetryMetrics returns current retry metrics
	GetRetryMetrics(ctx context.Context) *RetryMetrics
	// GetCircuitBreakerMetrics returns current circuit breaker metrics
	GetCircuitBreakerMetrics(ctx context.Context) *CircuitBreakerMetrics
	// GetConsistencyMetrics returns current consistency metrics
	GetConsistencyMetrics(ctx context.Context) *ConsistencyMetrics
	// GetDegradationMetrics returns current degradation metrics
	GetDegradationMetrics(ctx context.Context) *DegradationMetrics
	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
	// Close closes the metrics collector
	Close(ctx context.Context) error
}

// ErrorMetrics represents error metrics
type ErrorMetrics struct {
	TotalErrors     int64
	TransientErrors int64
	PermanentErrors int64
	CriticalErrors  int64
	UnknownErrors   int64
	AverageDuration time.Duration
	LastErrorTime   time.Time
	ErrorsBySource  map[string]int64
	ErrorsByType    map[string]int64
}

// RetryMetrics represents retry metrics
type RetryMetrics struct {
	TotalAttempts     int64
	SuccessfulRetries int64
	FailedRetries     int64
	AverageAttempts   float64
	LastRetryTime     time.Time
	RetrySuccessRate  float64
}

// CircuitBreakerMetrics represents circuit breaker metrics
type CircuitBreakerMetrics struct {
	TotalStateChanges int64
	ClosedCount       int64
	OpenCount         int64
	HalfOpenCount     int64
	CurrentState      map[string]string
	LastStateChange   time.Time
}

// ConsistencyMetrics represents consistency check metrics
type ConsistencyMetrics struct {
	TotalChecks     int64
	PassedChecks    int64
	FailedChecks    int64
	AverageDuration time.Duration
	LastCheckTime   time.Time
	PassRate        float64
}

// DegradationMetrics represents degradation metrics
type DegradationMetrics struct {
	TotalEvents           int64
	NormalMode            int64
	MongoDBUnavailable    int64
	PostgreSQLUnavailable int64
	BothUnavailable       int64
	CacheUnavailable      int64
	ReadOnlyMode          int64
	LastEventTime         time.Time
}

// DefaultErrorMetricsCollector implements ErrorMetricsCollector
type DefaultErrorMetricsCollector struct {
	mu                    sync.RWMutex
	errorMetrics          *ErrorMetrics
	retryMetrics          *RetryMetrics
	circuitBreakerMetrics *CircuitBreakerMetrics
	consistencyMetrics    *ConsistencyMetrics
	degradationMetrics    *DegradationMetrics
	logger                core.Logger
	metricsCollector      core.MetricsCollector
	initialized           bool
}

// NewErrorMetricsCollector creates a new error metrics collector
func NewErrorMetricsCollector(
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) ErrorMetricsCollector {
	return &DefaultErrorMetricsCollector{
		errorMetrics: &ErrorMetrics{
			ErrorsBySource: make(map[string]int64),
			ErrorsByType:   make(map[string]int64),
		},
		retryMetrics: &RetryMetrics{},
		circuitBreakerMetrics: &CircuitBreakerMetrics{
			CurrentState: make(map[string]string),
		},
		consistencyMetrics: &ConsistencyMetrics{},
		degradationMetrics: &DegradationMetrics{},
		logger:             logger,
		metricsCollector:   metricsCollector,
		initialized:        false,
	}
}

// Initialize initializes the error metrics collector
func (c *DefaultErrorMetricsCollector) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	c.initialized = true
	c.logger.Info("Error metrics collector initialized")

	return nil
}

// RecordError records an error occurrence
func (c *DefaultErrorMetricsCollector) RecordError(ctx context.Context, errorType string, source string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return
	}

	c.errorMetrics.TotalErrors++
	c.errorMetrics.LastErrorTime = time.Now()

	// Update error type counters
	switch errorType {
	case "transient":
		c.errorMetrics.TransientErrors++
	case "permanent":
		c.errorMetrics.PermanentErrors++
	case "critical":
		c.errorMetrics.CriticalErrors++
	default:
		c.errorMetrics.UnknownErrors++
	}

	// Update source counters
	c.errorMetrics.ErrorsBySource[source]++

	// Update type counters
	c.errorMetrics.ErrorsByType[errorType]++

	// Update average duration
	if c.errorMetrics.TotalErrors == 1 {
		c.errorMetrics.AverageDuration = duration
	} else {
		totalDuration := c.errorMetrics.AverageDuration * time.Duration(c.errorMetrics.TotalErrors-1)
		totalDuration += duration
		c.errorMetrics.AverageDuration = totalDuration / time.Duration(c.errorMetrics.TotalErrors)
	}

	// Record metrics
	c.metricsCollector.RecordCounter("error_total", c.errorMetrics.TotalErrors, nil)
	c.metricsCollector.RecordCounter(fmt.Sprintf("error_%s_total", errorType), c.errorMetrics.TotalErrors, nil)
	c.metricsCollector.RecordCounter(fmt.Sprintf("error_source_%s_total", source), c.errorMetrics.ErrorsBySource[source], nil)

	c.logger.Warn("Error recorded", "type", errorType, "source", source, "duration_ms", duration.Milliseconds())
}

// RecordRetryAttempt records a retry attempt
func (c *DefaultErrorMetricsCollector) RecordRetryAttempt(ctx context.Context, attempt int, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return
	}

	c.retryMetrics.TotalAttempts++
	c.retryMetrics.LastRetryTime = time.Now()

	if success {
		c.retryMetrics.SuccessfulRetries++
	} else {
		c.retryMetrics.FailedRetries++
	}

	// Update average attempts
	if c.retryMetrics.TotalAttempts == 1 {
		c.retryMetrics.AverageAttempts = float64(attempt)
	} else {
		totalAttempts := c.retryMetrics.AverageAttempts * float64(c.retryMetrics.TotalAttempts-1)
		totalAttempts += float64(attempt)
		c.retryMetrics.AverageAttempts = totalAttempts / float64(c.retryMetrics.TotalAttempts)
	}

	// Update success rate
	if c.retryMetrics.TotalAttempts > 0 {
		c.retryMetrics.RetrySuccessRate = float64(c.retryMetrics.SuccessfulRetries) / float64(c.retryMetrics.TotalAttempts)
	}

	// Record metrics
	c.metricsCollector.RecordCounter("retry_attempts_total", c.retryMetrics.TotalAttempts, nil)
	c.metricsCollector.RecordCounter("retry_successes_total", c.retryMetrics.SuccessfulRetries, nil)
	c.metricsCollector.RecordCounter("retry_failures_total", c.retryMetrics.FailedRetries, nil)
	c.metricsCollector.RecordGauge("retry_success_rate", c.retryMetrics.RetrySuccessRate, nil)

	c.logger.Debug("Retry attempt recorded", "attempt", attempt, "success", success)
}

// RecordCircuitBreakerStateChange records a circuit breaker state change
func (c *DefaultErrorMetricsCollector) RecordCircuitBreakerStateChange(ctx context.Context, breaker string, oldState string, newState string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return
	}

	c.circuitBreakerMetrics.TotalStateChanges++
	c.circuitBreakerMetrics.LastStateChange = time.Now()
	c.circuitBreakerMetrics.CurrentState[breaker] = newState

	// Update state counters
	switch newState {
	case "Closed":
		c.circuitBreakerMetrics.ClosedCount++
	case "Open":
		c.circuitBreakerMetrics.OpenCount++
	case "HalfOpen":
		c.circuitBreakerMetrics.HalfOpenCount++
	}

	// Record metrics
	c.metricsCollector.RecordCounter("circuit_breaker_state_changes_total", c.circuitBreakerMetrics.TotalStateChanges, nil)
	c.metricsCollector.RecordGauge(fmt.Sprintf("circuit_breaker_%s_state", breaker), 1, nil)

	c.logger.Info("Circuit breaker state changed", "breaker", breaker, "old_state", oldState, "new_state", newState)
}

// RecordConsistencyCheck records a consistency check result
func (c *DefaultErrorMetricsCollector) RecordConsistencyCheck(ctx context.Context, passed bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return
	}

	c.consistencyMetrics.TotalChecks++
	c.consistencyMetrics.LastCheckTime = time.Now()

	if passed {
		c.consistencyMetrics.PassedChecks++
	} else {
		c.consistencyMetrics.FailedChecks++
	}

	// Update average duration
	if c.consistencyMetrics.TotalChecks == 1 {
		c.consistencyMetrics.AverageDuration = duration
	} else {
		totalDuration := c.consistencyMetrics.AverageDuration * time.Duration(c.consistencyMetrics.TotalChecks-1)
		totalDuration += duration
		c.consistencyMetrics.AverageDuration = totalDuration / time.Duration(c.consistencyMetrics.TotalChecks)
	}

	// Update pass rate
	if c.consistencyMetrics.TotalChecks > 0 {
		c.consistencyMetrics.PassRate = float64(c.consistencyMetrics.PassedChecks) / float64(c.consistencyMetrics.TotalChecks)
	}

	// Record metrics
	c.metricsCollector.RecordCounter("consistency_checks_total", c.consistencyMetrics.TotalChecks, nil)
	c.metricsCollector.RecordCounter("consistency_checks_passed", c.consistencyMetrics.PassedChecks, nil)
	c.metricsCollector.RecordCounter("consistency_checks_failed", c.consistencyMetrics.FailedChecks, nil)
	c.metricsCollector.RecordGauge("consistency_pass_rate", c.consistencyMetrics.PassRate, nil)

	c.logger.Debug("Consistency check recorded", "passed", passed, "duration_ms", duration.Milliseconds())
}

// RecordDegradationEvent records a degradation event
func (c *DefaultErrorMetricsCollector) RecordDegradationEvent(ctx context.Context, mode string, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return
	}

	c.degradationMetrics.TotalEvents++
	c.degradationMetrics.LastEventTime = time.Now()

	// Update mode counters
	switch mode {
	case "Normal":
		c.degradationMetrics.NormalMode++
	case "MongoDBUnavailable":
		c.degradationMetrics.MongoDBUnavailable++
	case "PostgreSQLUnavailable":
		c.degradationMetrics.PostgreSQLUnavailable++
	case "BothUnavailable":
		c.degradationMetrics.BothUnavailable++
	case "CacheUnavailable":
		c.degradationMetrics.CacheUnavailable++
	case "ReadOnly":
		c.degradationMetrics.ReadOnlyMode++
	}

	// Record metrics
	c.metricsCollector.RecordCounter("degradation_events_total", c.degradationMetrics.TotalEvents, nil)
	c.metricsCollector.RecordGauge(fmt.Sprintf("degradation_%s_total", mode), 1, nil)

	c.logger.Warn("Degradation event recorded", "mode", mode, "reason", reason)
}

// GetErrorMetrics returns current error metrics
func (c *DefaultErrorMetricsCollector) GetErrorMetrics(ctx context.Context) *ErrorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	metrics := *c.errorMetrics
	metrics.ErrorsBySource = make(map[string]int64)
	metrics.ErrorsByType = make(map[string]int64)

	for k, v := range c.errorMetrics.ErrorsBySource {
		metrics.ErrorsBySource[k] = v
	}
	for k, v := range c.errorMetrics.ErrorsByType {
		metrics.ErrorsByType[k] = v
	}

	return &metrics
}

// GetRetryMetrics returns current retry metrics
func (c *DefaultErrorMetricsCollector) GetRetryMetrics(ctx context.Context) *RetryMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := *c.retryMetrics
	return &metrics
}

// GetCircuitBreakerMetrics returns current circuit breaker metrics
func (c *DefaultErrorMetricsCollector) GetCircuitBreakerMetrics(ctx context.Context) *CircuitBreakerMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := *c.circuitBreakerMetrics
	metrics.CurrentState = make(map[string]string)

	for k, v := range c.circuitBreakerMetrics.CurrentState {
		metrics.CurrentState[k] = v
	}

	return &metrics
}

// GetConsistencyMetrics returns current consistency metrics
func (c *DefaultErrorMetricsCollector) GetConsistencyMetrics(ctx context.Context) *ConsistencyMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := *c.consistencyMetrics
	return &metrics
}

// GetDegradationMetrics returns current degradation metrics
func (c *DefaultErrorMetricsCollector) GetDegradationMetrics(ctx context.Context) *DegradationMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := *c.degradationMetrics
	return &metrics
}

// Health returns the health status
func (c *DefaultErrorMetricsCollector) Health(ctx context.Context) *core.HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "error metrics collector not initialized",
		}
	}

	return &core.HealthStatus{
		Status: "healthy",
	}
}

// Close closes the error metrics collector
func (c *DefaultErrorMetricsCollector) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.initialized = false
	return nil
}
