package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
)

// HealthCheckHandler handles health check requests
type HealthCheckHandler struct {
	dbManager database.DatabaseManager
	logger    core.Logger
	metrics   core.MetricsCollector
	initialized bool

	// Component health cache
	mu                    sync.RWMutex
	lastHealthCheckTime   time.Time
	lastHealthCheckStatus *HealthCheckResponse
	healthCheckInterval   time.Duration
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status     string                      `json:"status"`
	Timestamp  int64                       `json:"timestamp"`
	Version    string                      `json:"version"`
	Components map[string]*ComponentStatus `json:"components"`
}

// ComponentStatus represents the status of a component
type ComponentStatus struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	Timestamp    int64  `json:"timestamp"`
	ResponseTime int64  `json:"responseTime"`
}

// ReadinessResponse represents a readiness probe response
type ReadinessResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Ready     bool   `json:"ready"`
	Message   string `json:"message"`
}

// LivenessResponse represents a liveness probe response
type LivenessResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Alive     bool   `json:"alive"`
	Message   string `json:"message"`
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(
	dbManager database.DatabaseManager,
	logger core.Logger,
	metrics core.MetricsCollector,
) *HealthCheckHandler {
	return &HealthCheckHandler{
		dbManager:           dbManager,
		logger:              logger,
		metrics:             metrics,
		initialized:         false,
		healthCheckInterval: 30 * time.Second,
	}
}

// Initialize initializes the health check handler
func (h *HealthCheckHandler) Initialize(ctx context.Context) error {
	if h.initialized {
		return nil
	}

	if h.dbManager == nil {
		return fmt.Errorf("database manager is required")
	}

	h.initialized = true
	h.logger.Info("Health check handler initialized")
	return nil
}

// HandleHealth handles GET /health request
func (h *HealthCheckHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("health_check_overall_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response := h.performHealthCheck(ctx)
	statusCode := http.StatusOK
	if response.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	h.metrics.RecordCounter("health_check_overall_status", 1, nil)
	h.respondJSON(w, statusCode, response)
}

// HandleReady handles GET /health/ready request (Kubernetes readiness probe)
func (h *HealthCheckHandler) HandleReady(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("health_check_ready_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	ready := h.checkReadiness(ctx)

	response := &ReadinessResponse{
		Status:    "ready",
		Timestamp: time.Now().Unix(),
		Ready:     ready,
		Message:   "readiness check complete",
	}

	if !ready {
		response.Status = "not_ready"
		response.Message = "system is not ready to accept traffic"
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	h.metrics.RecordCounter("health_check_ready_status", 1, nil)
	h.respondJSON(w, statusCode, response)
}

// HandleLive handles GET /health/live request (Kubernetes liveness probe)
func (h *HealthCheckHandler) HandleLive(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("health_check_live_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	alive := h.checkLiveness(ctx)

	response := &LivenessResponse{
		Status:    "alive",
		Timestamp: time.Now().Unix(),
		Alive:     alive,
		Message:   "liveness check complete",
	}

	if !alive {
		response.Status = "dead"
		response.Message = "system is not responding"
	}

	statusCode := http.StatusOK
	if !alive {
		statusCode = http.StatusServiceUnavailable
	}

	h.metrics.RecordCounter("health_check_live_status", 1, nil)
	h.respondJSON(w, statusCode, response)
}

// HandleComponents handles GET /health/components request
func (h *HealthCheckHandler) HandleComponents(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("health_check_components_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response := h.performHealthCheck(ctx)
	statusCode := http.StatusOK

	h.metrics.RecordCounter("health_check_components_status", 1, nil)
	h.respondJSON(w, statusCode, response)
}

// Helper methods

// performHealthCheck performs a comprehensive health check
func (h *HealthCheckHandler) performHealthCheck(ctx context.Context) *HealthCheckResponse {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if we have a recent cached result
	if h.lastHealthCheckStatus != nil && time.Since(h.lastHealthCheckTime) < h.healthCheckInterval {
		return h.lastHealthCheckStatus
	}

	response := &HealthCheckResponse{
		Status:     "healthy",
		Timestamp:  time.Now().Unix(),
		Version:    "1.0.0",
		Components: make(map[string]*ComponentStatus),
	}

	// Check MongoDB
	mongoStatus := h.checkMongoDBHealth(ctx)
	response.Components["mongodb"] = mongoStatus
	if mongoStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check PostgreSQL
	postgresStatus := h.checkPostgresHealth(ctx)
	response.Components["postgresql"] = postgresStatus
	if postgresStatus.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check Redis
	redisStatus := h.checkRedisHealth(ctx)
	response.Components["redis"] = redisStatus
	if redisStatus.Status != "healthy" {
		response.Status = "degraded"
	}

	// Cache the result
	h.lastHealthCheckStatus = response
	h.lastHealthCheckTime = time.Now()

	return response
}

// checkReadiness checks if the system is ready to accept traffic
func (h *HealthCheckHandler) checkReadiness(ctx context.Context) bool {
	// Check critical dependencies
	mongoStatus := h.checkMongoDBHealth(ctx)
	postgresStatus := h.checkPostgresHealth(ctx)

	// System is ready if critical databases are healthy
	return mongoStatus.Status == "healthy" && postgresStatus.Status == "healthy"
}

// checkLiveness checks if the system is alive
func (h *HealthCheckHandler) checkLiveness(ctx context.Context) bool {
	// Basic liveness check - just verify handler is initialized
	// and can respond to requests
	if !h.initialized {
		return false
	}

	// Try a simple operation to verify system is responsive
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// checkMongoDBHealth checks MongoDB health
func (h *HealthCheckHandler) checkMongoDBHealth(ctx context.Context) *ComponentStatus {
	start := time.Now()
	status := &ComponentStatus{
		Name:      "MongoDB",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}

	if h.dbManager == nil {
		status.Status = "unhealthy"
		status.Error = "database manager not available"
		status.ResponseTime = time.Since(start).Milliseconds()
		return status
	}

	// Perform a simple health check
	err := h.dbManager.CheckMongoHealth(ctx)
	if err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		h.logger.Error("MongoDB health check failed", "error", err.Error())
	}

	status.ResponseTime = time.Since(start).Milliseconds()
	return status
}

// checkPostgresHealth checks PostgreSQL health
func (h *HealthCheckHandler) checkPostgresHealth(ctx context.Context) *ComponentStatus {
	start := time.Now()
	status := &ComponentStatus{
		Name:      "PostgreSQL",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}

	if h.dbManager == nil {
		status.Status = "unhealthy"
		status.Error = "database manager not available"
		status.ResponseTime = time.Since(start).Milliseconds()
		return status
	}

	// Perform a simple health check
	err := h.dbManager.CheckPostgresHealth(ctx)
	if err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		h.logger.Error("PostgreSQL health check failed", "error", err.Error())
	}

	status.ResponseTime = time.Since(start).Milliseconds()
	return status
}

// checkRedisHealth checks Redis cache health
func (h *HealthCheckHandler) checkRedisHealth(ctx context.Context) *ComponentStatus {
	start := time.Now()
	status := &ComponentStatus{
		Name:      "Redis Cache",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}

	if h.dbManager == nil {
		status.Status = "unhealthy"
		status.Error = "database manager not available"
		status.ResponseTime = time.Since(start).Milliseconds()
		return status
	}

	// Redis health check is optional - if not available, mark as degraded
	status.Status = "degraded"
	status.Error = "Redis cache not available"

	status.ResponseTime = time.Since(start).Milliseconds()
	return status
}

// respondJSON responds with JSON data
func (h *HealthCheckHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err.Error())
	}
}

// respondError responds with an error message
func (h *HealthCheckHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"status":    "error",
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	h.respondJSON(w, statusCode, response)
}

// Health returns the health status of the health check handler
func (h *HealthCheckHandler) Health(ctx context.Context) *core.HealthStatus {
	if !h.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "health check handler not initialized",
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "health check handler is operational",
		Timestamp: time.Now(),
	}
}

// Close closes the health check handler
func (h *HealthCheckHandler) Close(ctx context.Context) error {
	if !h.initialized {
		return nil
	}

	h.initialized = false
	return nil
}
