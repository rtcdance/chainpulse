package core

import (
	"context"
	"sync"
	"time"
)

// DefaultHealthChecker monitors system health
type DefaultHealthChecker struct {
	mu              sync.RWMutex
	pluginRegistry  PluginRegistry
	configManager   ConfigManager
	eventBus        EventBus
	metricsCollector MetricsCollector
	logger          Logger
	lastCheckTime   time.Time
	lastCheckStatus HealthStatus
	checkInterval   time.Duration
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    int64                  `json:"duration_ms"`
	Components  map[string]interface{} `json:"components"`
	Message     string                 `json:"message"`
}

// ComponentHealth represents the health of a component
type ComponentHealth struct {
	Name    string                 `json:"name"`
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewDefaultHealthChecker creates a new health checker
func NewDefaultHealthChecker(
	registry PluginRegistry,
	config ConfigManager,
	bus EventBus,
	metrics MetricsCollector,
	logger Logger,
) *DefaultHealthChecker {
	return &DefaultHealthChecker{
		pluginRegistry:   registry,
		configManager:    config,
		eventBus:         bus,
		metricsCollector: metrics,
		logger:           logger,
		checkInterval:    30 * time.Second,
		lastCheckStatus: HealthStatus{
			Status:  "unknown",
			Message: "no check performed yet",
			Details: make(map[string]interface{}),
		},
	}
}

// Check performs a health check on the system
func (h *DefaultHealthChecker) Check(ctx context.Context) (HealthStatus, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	startTime := time.Now()
	status := HealthStatus{
		Status:  "healthy",
		Details: make(map[string]interface{}),
	}

	// Check plugins
	pluginHealth := h.checkPlugins(ctx)
	status.Details["plugins"] = pluginHealth

	// Check configuration
	configHealth := h.checkConfiguration(ctx)
	status.Details["configuration"] = configHealth

	// Check event bus
	eventBusHealth := h.checkEventBus(ctx)
	status.Details["event_bus"] = eventBusHealth

	// Check metrics
	metricsHealth := h.checkMetrics(ctx)
	status.Details["metrics"] = metricsHealth

	// Determine overall status
	if !pluginHealth["healthy"].(bool) ||
		!configHealth["healthy"].(bool) ||
		!eventBusHealth["healthy"].(bool) ||
		!metricsHealth["healthy"].(bool) {
		status.Status = "unhealthy"
		status.Message = "one or more components are unhealthy"
	} else {
		status.Message = "all components are healthy"
	}

	duration := time.Since(startTime)
	status.Details["check_duration_ms"] = duration.Milliseconds()

	h.lastCheckTime = time.Now()
	h.lastCheckStatus = status

	return status, nil
}

// GetLastCheckStatus returns the last health check status
func (h *DefaultHealthChecker) GetLastCheckStatus() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheckStatus
}

// GetLastCheckTime returns the time of the last health check
func (h *DefaultHealthChecker) GetLastCheckTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheckTime
}

// SetCheckInterval sets the health check interval
func (h *DefaultHealthChecker) SetCheckInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkInterval = interval
}

// GetCheckInterval returns the health check interval
func (h *DefaultHealthChecker) GetCheckInterval() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.checkInterval
}

// checkPlugins checks the health of all plugins
func (h *DefaultHealthChecker) checkPlugins(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"count":   0,
		"details": make([]map[string]interface{}, 0),
	}

	if h.pluginRegistry == nil {
		result["healthy"] = false
		result["message"] = "plugin registry not available"
		return result
	}

	plugins := h.pluginRegistry.List()
	result["count"] = len(plugins)

	details := make([]map[string]interface{}, 0)
	for _, plugin := range plugins {
		pluginHealth := map[string]interface{}{
			"name":    plugin.Name(),
			"version": plugin.Version(),
			"healthy": true,
		}

		// Check plugin health
		if err := plugin.Health(); err != nil {
			pluginHealth["healthy"] = false
			pluginHealth["error"] = err.Error()
			result["healthy"] = false
		}

		details = append(details, pluginHealth)
	}

	result["details"] = details
	return result
}

// checkConfiguration checks the health of configuration
func (h *DefaultHealthChecker) checkConfiguration(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
	}

	if h.configManager == nil {
		result["healthy"] = false
		result["message"] = "configuration manager not available"
		return result
	}

	// Try to load configuration
	config, err := h.configManager.Load()
	if err != nil {
		result["healthy"] = false
		result["error"] = err.Error()
		return result
	}

	// Validate configuration
	if err := h.configManager.Validate(config); err != nil {
		result["healthy"] = false
		result["error"] = err.Error()
		return result
	}

	result["message"] = "configuration is valid"
	result["deployment_mode"] = config.DeploymentMode
	result["log_level"] = config.LogLevel

	return result
}

// checkEventBus checks the health of the event bus
func (h *DefaultHealthChecker) checkEventBus(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
	}

	if h.eventBus == nil {
		result["healthy"] = false
		result["message"] = "event bus not available"
		return result
	}

	// Event bus is healthy if it's available
	result["message"] = "event bus is operational"

	return result
}

// checkMetrics checks the health of metrics collection
func (h *DefaultHealthChecker) checkMetrics(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
	}

	if h.metricsCollector == nil {
		result["healthy"] = false
		result["message"] = "metrics collector not available"
		return result
	}

	// Get metrics to verify collector is working
	metrics := h.metricsCollector.GetMetrics()
	result["message"] = "metrics collector is operational"
	result["metrics_count"] = len(metrics)

	return result
}

// IsHealthy returns true if the system is healthy
func (h *DefaultHealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheckStatus.Status == "healthy"
}

// GetHealthSummary returns a summary of the health status
func (h *DefaultHealthChecker) GetHealthSummary() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"status":           h.lastCheckStatus.Status,
		"message":          h.lastCheckStatus.Message,
		"last_check_time":  h.lastCheckTime,
		"check_interval":   h.checkInterval.String(),
		"details":          h.lastCheckStatus.Details,
	}
}

// PerformHealthCheck performs a health check and logs the result
func (h *DefaultHealthChecker) PerformHealthCheck(ctx context.Context) error {
	status, err := h.Check(ctx)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("health check failed", "error", err.Error())
		}
		return err
	}

	if h.logger != nil {
		if status.Status == "healthy" {
			h.logger.Info("health check passed", "status", status.Status)
		} else {
			h.logger.Warn("health check failed", "status", status.Status, "message", status.Message)
		}
	}

	return nil
}

// WatchHealth starts a goroutine that periodically checks health
func (h *DefaultHealthChecker) WatchHealth(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.PerformHealthCheck(ctx); err != nil {
				h.logger.Warn("health check failed", "error", err)
			}
		}
	}
}
