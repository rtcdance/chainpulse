package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FailureDetector detects service failures
type FailureDetector struct {
	mu                  sync.RWMutex
	id                  string
	services            map[string]*ServiceHealthInfo
	healthCheckInterval time.Duration
	failureThreshold    int
	recoveryThreshold   int
	metrics             *FailureMetrics
	lastCheckTime       time.Time
	predictiveDetection bool
}

// ServiceHealthInfo tracks service health
type ServiceHealthInfo struct {
	ServiceID            string
	Status               string // "healthy", "degraded", "unhealthy", "failed"
	LastHealthCheck      time.Time
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	FailureHistory       []time.Time
	ResponseTime         time.Duration
	ErrorRate            float64
	PredictedFailure     bool
	PredictionScore      float64
}

// FailureMetrics tracks failure detection metrics
type FailureMetrics struct {
	mu                   sync.RWMutex
	FailuresDetected     int64
	FalsePositives       int64
	FalseNegatives       int64
	AverageDetectionTime time.Duration
	TotalDetectionTime   time.Duration
	LastDetectionTime    time.Time
	PredictedFailures    int64
	ActualFailures       int64
}

// NewFailureDetector creates a new failure detector
func NewFailureDetector(id string) *FailureDetector {
	return &FailureDetector{
		id:                  id,
		services:            make(map[string]*ServiceHealthInfo),
		healthCheckInterval: 10 * time.Second,
		failureThreshold:    3,
		recoveryThreshold:   2,
		metrics: &FailureMetrics{
			LastDetectionTime: time.Now(),
		},
		lastCheckTime:       time.Now(),
		predictiveDetection: true,
	}
}

// RegisterService registers a service for monitoring
func (fd *FailureDetector) RegisterService(serviceID string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	fd.services[serviceID] = &ServiceHealthInfo{
		ServiceID:       serviceID,
		Status:          "healthy",
		LastHealthCheck: time.Now(),
		FailureHistory:  make([]time.Time, 0),
	}
}

// ReportHealthCheck reports a health check result
func (fd *FailureDetector) ReportHealthCheck(serviceID string, healthy bool, responseTime time.Duration, errorRate float64) error {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	info, exists := fd.services[serviceID]
	if !exists {
		return fmt.Errorf("service not registered")
	}

	info.LastHealthCheck = time.Now()
	info.ResponseTime = responseTime
	info.ErrorRate = errorRate

	if healthy {
		info.ConsecutiveSuccesses++
		info.ConsecutiveFailures = 0

		// Recover from degraded/unhealthy state
		if info.Status != "healthy" && info.ConsecutiveSuccesses >= fd.recoveryThreshold {
			info.Status = "healthy"
		}
	} else {
		info.ConsecutiveFailures++
		info.ConsecutiveSuccesses = 0
		info.FailureHistory = append(info.FailureHistory, time.Now())

		// Limit failure history size
		if len(info.FailureHistory) > 100 {
			info.FailureHistory = info.FailureHistory[1:]
		}

		// Update status based on consecutive failures
		if info.ConsecutiveFailures == 1 {
			info.Status = "degraded"
		} else if info.ConsecutiveFailures >= fd.failureThreshold {
			info.Status = "failed"

			fd.metrics.mu.Lock()
			fd.metrics.FailuresDetected++
			fd.metrics.LastDetectionTime = time.Now()
			fd.metrics.mu.Unlock()
		}
	}

	return nil
}

// DetectFailures detects failures across all services
func (fd *FailureDetector) DetectFailures(ctx context.Context) ([]string, error) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	start := time.Now()
	defer func() {
		fd.recordDetectionTime(time.Since(start))
	}()

	var failedServices []string

	for serviceID, info := range fd.services {
		// Check for timeout
		if time.Since(info.LastHealthCheck) > 2*fd.healthCheckInterval {
			info.Status = "failed"
			failedServices = append(failedServices, serviceID)
			continue
		}

		// Check for predictive failure
		if fd.predictiveDetection {
			if fd.predictFailure(info) {
				info.PredictedFailure = true
				fd.metrics.mu.Lock()
				fd.metrics.PredictedFailures++
				fd.metrics.mu.Unlock()
			}
		}

		// Check for actual failure
		if info.Status == "failed" {
			failedServices = append(failedServices, serviceID)
		}
	}

	return failedServices, nil
}

// predictFailure predicts if a service will fail
func (fd *FailureDetector) predictFailure(info *ServiceHealthInfo) bool {
	// Simple prediction based on error rate trend
	if len(info.FailureHistory) < 3 {
		return false
	}

	// Calculate error rate trend
	recentFailures := 0
	for i := len(info.FailureHistory) - 3; i < len(info.FailureHistory); i++ {
		if i >= 0 {
			recentFailures++
		}
	}

	// If error rate is increasing, predict failure
	if recentFailures >= 2 && info.ErrorRate > 0.5 {
		info.PredictionScore = float64(recentFailures) / 3.0 * info.ErrorRate
		return info.PredictionScore > 0.7
	}

	return false
}

// recordDetectionTime records failure detection time
func (fd *FailureDetector) recordDetectionTime(duration time.Duration) {
	fd.metrics.mu.Lock()
	defer fd.metrics.mu.Unlock()

	fd.metrics.TotalDetectionTime += duration
	detectionCount := fd.metrics.FailuresDetected + fd.metrics.PredictedFailures
	if detectionCount > 0 {
		fd.metrics.AverageDetectionTime = fd.metrics.TotalDetectionTime / time.Duration(detectionCount)
	}
}

// GetServiceStatus returns the status of a service
func (fd *FailureDetector) GetServiceStatus(serviceID string) (string, error) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	info, exists := fd.services[serviceID]
	if !exists {
		return "", fmt.Errorf("service not found")
	}

	return info.Status, nil
}

// GetFailedServices returns all failed services
func (fd *FailureDetector) GetFailedServices() []string {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	var failed []string
	for serviceID, info := range fd.services {
		if info.Status == "failed" {
			failed = append(failed, serviceID)
		}
	}

	return failed
}

// GetMetrics returns failure detection metrics
func (fd *FailureDetector) GetMetrics() map[string]any {
	fd.metrics.mu.RLock()
	defer fd.metrics.mu.RUnlock()

	return map[string]any{
		"failures_detected":      fd.metrics.FailuresDetected,
		"false_positives":        fd.metrics.FalsePositives,
		"false_negatives":        fd.metrics.FalseNegatives,
		"average_detection_time": fd.metrics.AverageDetectionTime.String(),
		"total_detection_time":   fd.metrics.TotalDetectionTime.String(),
		"predicted_failures":     fd.metrics.PredictedFailures,
		"actual_failures":        fd.metrics.ActualFailures,
	}
}

// AutomaticFailover handles automatic failover
type AutomaticFailover struct {
	mu                 sync.RWMutex
	id                 string
	failureDetector    *FailureDetector
	failoverStrategies map[string]FailoverStrategy
	metrics            *FailoverMetrics
	lastFailoverTime   time.Time
	failoverCooldown   time.Duration
}

// FailoverStrategy defines how to handle failover
type FailoverStrategy interface {
	Execute(ctx context.Context, failedService string) error
	Rollback(ctx context.Context, failedService string) error
}

// FailoverMetrics tracks failover metrics
type FailoverMetrics struct {
	mu                    sync.RWMutex
	FailoversExecuted     int64
	FailoversSuccessful   int64
	FailoversFailed       int64
	RollbacksExecuted     int64
	AverageFailoverTime   time.Duration
	TotalFailoverTime     time.Duration
	LastFailoverTime      time.Time
	DataConsistencyIssues int64
}

// NewAutomaticFailover creates a new automatic failover
func NewAutomaticFailover(id string, fd *FailureDetector) *AutomaticFailover {
	return &AutomaticFailover{
		id:                 id,
		failureDetector:    fd,
		failoverStrategies: make(map[string]FailoverStrategy),
		metrics: &FailoverMetrics{
			LastFailoverTime: time.Now(),
		},
		lastFailoverTime: time.Now(),
		failoverCooldown: 5 * time.Minute,
	}
}

// RegisterFailoverStrategy registers a failover strategy
func (af *AutomaticFailover) RegisterFailoverStrategy(serviceID string, strategy FailoverStrategy) {
	af.mu.Lock()
	defer af.mu.Unlock()

	af.failoverStrategies[serviceID] = strategy
}

// ExecuteFailover executes failover for failed services
func (af *AutomaticFailover) ExecuteFailover(ctx context.Context, failedServices []string) error {
	af.mu.Lock()
	defer af.mu.Unlock()

	// Check cooldown
	if time.Since(af.lastFailoverTime) < af.failoverCooldown {
		return fmt.Errorf("failover cooldown active")
	}

	start := time.Now()
	defer func() {
		af.recordFailoverTime(time.Since(start))
	}()

	for _, serviceID := range failedServices {
		strategy, exists := af.failoverStrategies[serviceID]
		if !exists {
			continue
		}

		err := strategy.Execute(ctx, serviceID)
		if err != nil {
			af.metrics.mu.Lock()
			af.metrics.FailoversFailed++
			af.metrics.mu.Unlock()
			continue
		}

		af.metrics.mu.Lock()
		af.metrics.FailoversExecuted++
		af.metrics.FailoversSuccessful++
		af.metrics.mu.Unlock()
	}

	af.lastFailoverTime = time.Now()
	return nil
}

// Rollback rolls back a failover
func (af *AutomaticFailover) Rollback(ctx context.Context, serviceID string) error {
	af.mu.Lock()
	defer af.mu.Unlock()

	strategy, exists := af.failoverStrategies[serviceID]
	if !exists {
		return fmt.Errorf("no failover strategy for service")
	}

	err := strategy.Rollback(ctx, serviceID)
	if err != nil {
		return err
	}

	af.metrics.mu.Lock()
	af.metrics.RollbacksExecuted++
	af.metrics.mu.Unlock()

	return nil
}

// recordFailoverTime records failover execution time
func (af *AutomaticFailover) recordFailoverTime(duration time.Duration) {
	af.metrics.mu.Lock()
	defer af.metrics.mu.Unlock()

	af.metrics.TotalFailoverTime += duration
	if af.metrics.FailoversExecuted > 0 {
		af.metrics.AverageFailoverTime = af.metrics.TotalFailoverTime / time.Duration(af.metrics.FailoversExecuted)
	}
	af.metrics.LastFailoverTime = time.Now()
}

// GetMetrics returns failover metrics
func (af *AutomaticFailover) GetMetrics() map[string]any {
	af.metrics.mu.RLock()
	defer af.metrics.mu.RUnlock()

	return map[string]any{
		"failovers_executed":      af.metrics.FailoversExecuted,
		"failovers_successful":    af.metrics.FailoversSuccessful,
		"failovers_failed":        af.metrics.FailoversFailed,
		"rollbacks_executed":      af.metrics.RollbacksExecuted,
		"average_failover_time":   af.metrics.AverageFailoverTime.String(),
		"total_failover_time":     af.metrics.TotalFailoverTime.String(),
		"data_consistency_issues": af.metrics.DataConsistencyIssues,
	}
}
