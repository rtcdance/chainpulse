package shared

import (
	"sync"
	"time"
)

// HealthCheck tracks health status of components
type HealthCheck struct {
	components map[string]*ComponentHealth
	mu         sync.RWMutex
}

// ComponentHealth tracks health of a single component
type ComponentHealth struct {
	name              string
	status            HealthStatus
	lastCheckTime     time.Time
	lastErrorTime     time.Time
	consecutiveErrors int
	checkInterval     time.Duration
	errorThreshold    int
	mu                sync.RWMutex
}

// HealthStatus represents component health status
type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusDegraded
	StatusUnhealthy
)

// NewHealthCheck creates a new health check instance
func NewHealthCheck() *HealthCheck {
	return &HealthCheck{
		components: make(map[string]*ComponentHealth),
	}
}

// NewComponentHealth creates a new component health instance
func NewComponentHealth(name string, checkInterval time.Duration, errorThreshold int) *ComponentHealth {
	return &ComponentHealth{
		name:           name,
		status:         StatusHealthy,
		checkInterval:  checkInterval,
		errorThreshold: errorThreshold,
	}
}

// RegisterComponent registers a component for health checking
func (hc *HealthCheck) RegisterComponent(name string, checkInterval time.Duration, errorThreshold int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.components[name] = NewComponentHealth(name, checkInterval, errorThreshold)
}

// RecordSuccess records a successful health check
func (hc *HealthCheck) RecordSuccess(component string) {
	hc.mu.Lock()
	ch, ok := hc.components[component]
	hc.mu.Unlock()

	if !ok {
		return
	}

	ch.recordSuccess()
}

// recordSuccess records success in component health
func (ch *ComponentHealth) recordSuccess() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.lastCheckTime = time.Now()
	ch.consecutiveErrors = 0

	if ch.status != StatusHealthy {
		ch.status = StatusHealthy
	}
}

// RecordError records a failed health check
func (hc *HealthCheck) RecordError(component string) {
	hc.mu.Lock()
	ch, ok := hc.components[component]
	hc.mu.Unlock()

	if !ok {
		return
	}

	ch.recordError()
}

// recordError records error in component health
func (ch *ComponentHealth) recordError() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.lastCheckTime = time.Now()
	ch.lastErrorTime = time.Now()
	ch.consecutiveErrors++

	if ch.consecutiveErrors >= ch.errorThreshold {
		ch.status = StatusUnhealthy
	} else if ch.consecutiveErrors > 0 {
		ch.status = StatusDegraded
	}
}

// GetComponentStatus returns the status of a component
func (hc *HealthCheck) GetComponentStatus(component string) HealthStatus {
	hc.mu.RLock()
	ch, ok := hc.components[component]
	hc.mu.RUnlock()

	if !ok {
		return StatusUnhealthy
	}

	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.status
}

// GetComponentMetrics returns metrics for a component
func (hc *HealthCheck) GetComponentMetrics(component string) map[string]interface{} {
	hc.mu.RLock()
	ch, ok := hc.components[component]
	hc.mu.RUnlock()

	if !ok {
		return map[string]interface{}{
			"component": component,
			"error":     "component not found",
		}
	}

	return ch.getMetrics()
}

// getMetrics returns metrics from component health
func (ch *ComponentHealth) getMetrics() map[string]interface{} {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	statusStr := "unknown"
	switch ch.status {
	case StatusHealthy:
		statusStr = "healthy"
	case StatusDegraded:
		statusStr = "degraded"
	case StatusUnhealthy:
		statusStr = "unhealthy"
	}

	return map[string]interface{}{
		"component":           ch.name,
		"status":              statusStr,
		"last_check_time":     ch.lastCheckTime,
		"last_error_time":     ch.lastErrorTime,
		"consecutive_errors":  ch.consecutiveErrors,
		"check_interval":      ch.checkInterval.String(),
		"error_threshold":     ch.errorThreshold,
	}
}

// GetOverallHealth returns overall gateway health
func (hc *HealthCheck) GetOverallHealth() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if len(hc.components) == 0 {
		return StatusHealthy
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, ch := range hc.components {
		ch.mu.RLock()
		switch ch.status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		}
		ch.mu.RUnlock()
	}

	// If any component is unhealthy, overall is unhealthy
	if unhealthyCount > 0 {
		return StatusUnhealthy
	}

	// If any component is degraded, overall is degraded
	if degradedCount > 0 {
		return StatusDegraded
	}

	return StatusHealthy
}

// GetAllMetrics returns metrics for all components
func (hc *HealthCheck) GetAllMetrics() map[string]map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for name, ch := range hc.components {
		result[name] = ch.getMetrics()
	}

	return result
}

// GetHealthSummary returns a summary of overall health
func (hc *HealthCheck) GetHealthSummary() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for _, ch := range hc.components {
		ch.mu.RLock()
		switch ch.status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnhealthy:
			unhealthyCount++
		}
		ch.mu.RUnlock()
	}

	overallStatus := "unknown"
	switch hc.GetOverallHealth() {
	case StatusHealthy:
		overallStatus = "healthy"
	case StatusDegraded:
		overallStatus = "degraded"
	case StatusUnhealthy:
		overallStatus = "unhealthy"
	}

	return map[string]interface{}{
		"overall_status":    overallStatus,
		"total_components":  len(hc.components),
		"healthy_count":     healthyCount,
		"degraded_count":    degradedCount,
		"unhealthy_count":   unhealthyCount,
	}
}

// NeedsHealthCheck returns whether a component needs a health check
func (hc *HealthCheck) NeedsHealthCheck(component string) bool {
	hc.mu.RLock()
	ch, ok := hc.components[component]
	hc.mu.RUnlock()

	if !ok {
		return false
	}

	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return time.Since(ch.lastCheckTime) > ch.checkInterval
}

// GetComponentsNeedingCheck returns all components that need health checks
func (hc *HealthCheck) GetComponentsNeedingCheck() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var result []string
	for name, ch := range hc.components {
		ch.mu.RLock()
		if time.Since(ch.lastCheckTime) > ch.checkInterval {
			result = append(result, name)
		}
		ch.mu.RUnlock()
	}

	return result
}
