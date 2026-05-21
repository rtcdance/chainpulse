package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// DatabaseHealthChecker provides health-check access to database backends.
// This interface follows the Interface Segregation Principle: handlers only
// depend on the methods they actually use, not the full DatabaseManager.
type DatabaseHealthChecker interface {
	CheckMongoHealth(ctx context.Context) error
	CheckPostgresHealth(ctx context.Context) error
}

// HealthCheckHandler handles health check requests
type HealthCheckHandler struct {
	dbManager   DatabaseHealthChecker
	cachePlugin core.CachePlugin
	mqPlugin    core.MQPlugin
	logger      core.Logger
	metrics     core.MetricsCollector
	initialized bool

	// Component health cache
	mu                       sync.RWMutex
	lastHealthCheckTime      time.Time
	lastHealthCheckStatus    *HealthCheckResponse
	healthCheckInterval      time.Duration
	runtimeComponentProvider func(context.Context) *ComponentStatus
	readinessDetailsProvider func(context.Context) map[string]any
	rolloutReportProducer    RolloutReportProducer
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
	Name         string         `json:"name"`
	Status       string         `json:"status"`
	Error        string         `json:"error,omitempty"`
	Timestamp    int64          `json:"timestamp"`
	ResponseTime int64          `json:"responseTime"`
	Details      map[string]any `json:"details,omitempty"`
}

// ReadinessResponse represents a readiness probe response
type ReadinessResponse struct {
	Status    string         `json:"status"`
	Timestamp int64          `json:"timestamp"`
	Ready     bool           `json:"ready"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
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
	dbManager DatabaseHealthChecker,
	args ...any,
) *HealthCheckHandler {
	var (
		cachePlugin core.CachePlugin
		logger      core.Logger
		metrics     core.MetricsCollector
	)

	// Support both the current constructor shape
	//   NewHealthCheckHandler(db, cache, logger, metrics)
	// and the older test-only shape
	//   NewHealthCheckHandler(db, logger, metrics)
	switch len(args) {
	case 3:
		if cache, ok := args[0].(core.CachePlugin); ok {
			cachePlugin = cache
		}
		logger, _ = args[1].(core.Logger)
		metrics, _ = args[2].(core.MetricsCollector)
	case 2:
		logger, _ = args[0].(core.Logger)
		metrics, _ = args[1].(core.MetricsCollector)
	}

	return &HealthCheckHandler{
		dbManager:           dbManager,
		cachePlugin:         cachePlugin,
		logger:              logger,
		metrics:             metrics,
		initialized:         false,
		healthCheckInterval: core.DefaultTimeout,
	}
}

// InitializedForTests marks the handler initialized for focused route tests
// that do not exercise full dependency bootstrapping.
func (h *HealthCheckHandler) InitializedForTests() {
	h.initialized = true
}

// WithMQPlugin sets the MQ plugin for Kafka health checking.
func (h *HealthCheckHandler) WithMQPlugin(mq core.MQPlugin) *HealthCheckHandler {
	h.mqPlugin = mq
	return h
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

// SetRuntimeComponentProvider registers an optional runtime component provider
// used to enrich health responses with service-specific details.
func (h *HealthCheckHandler) SetRuntimeComponentProvider(provider func(context.Context) *ComponentStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.runtimeComponentProvider = provider
	h.lastHealthCheckStatus = nil
	h.lastHealthCheckTime = time.Time{}
}

// SetReadinessDetailsProvider registers an optional provider that enriches
// readiness responses with rollout or service-specific details.
func (h *HealthCheckHandler) SetReadinessDetailsProvider(provider func(context.Context) map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.readinessDetailsProvider = provider
}

// SetRolloutReportProvider registers an optional provider that exposes a
// service-specific rollout report surface.
func (h *HealthCheckHandler) SetRolloutReportProvider(provider func(context.Context) *RolloutReportDetails) {
	h.SetRolloutReportProducer(RolloutReportProducerFunc(provider))
}

// SetRolloutReportProducer registers an optional producer that exposes a
// service-specific rollout report surface.
func (h *HealthCheckHandler) SetRolloutReportProducer(producer RolloutReportProducer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.rolloutReportProducer = producer
}

// HandleHealth handles GET /health request
func (h *HealthCheckHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordHistogram("health_check_overall_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), core.DefaultTimeout)
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
		h.metrics.RecordHistogram("health_check_ready_time_ms", float64(duration), nil)
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
	if details := h.readinessDetails(ctx); len(details) > 0 {
		response.Details = details
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

// HandleRollout handles GET /health/rollout request.
func (h *HealthCheckHandler) HandleRollout(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordHistogram("health_check_rollout_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	report := h.rolloutReport(ctx)
	response := &RolloutReportResponse{
		Status:    "available",
		Timestamp: time.Now().Unix(),
		Available: true,
		Message:   "rollout report ready",
	}
	statusCode := http.StatusOK
	if report == nil || report.IsEmpty() {
		response.Status = "unavailable"
		response.Available = false
		response.Message = "rollout report not available"
		statusCode = http.StatusServiceUnavailable
	} else {
		response.Details = report
	}

	h.metrics.RecordCounter("health_check_rollout_status", 1, nil)
	h.respondJSON(w, statusCode, response)
}

// HandleLive handles GET /health/live request (Kubernetes liveness probe)
func (h *HealthCheckHandler) HandleLive(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordHistogram("health_check_live_time_ms", float64(duration), nil)
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
		h.metrics.RecordHistogram("health_check_components_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), core.DefaultTimeout)
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

	// Check Kafka (if configured)
	if h.mqPlugin != nil {
		kafkaStatus := h.checkKafkaHealth(ctx)
		response.Components["kafka"] = kafkaStatus
		if kafkaStatus.Status != "healthy" {
			response.Status = "unhealthy"
		}
	}

	if h.runtimeComponentProvider != nil {
		if runtimeStatus := h.runtimeComponentProvider(ctx); runtimeStatus != nil {
			response.Components["indexing_runtime"] = runtimeStatus
		}
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

func (h *HealthCheckHandler) readinessDetails(ctx context.Context) map[string]any {
	h.mu.RLock()
	provider := h.readinessDetailsProvider
	h.mu.RUnlock()

	if provider == nil {
		return nil
	}
	return provider(ctx)
}

func (h *HealthCheckHandler) rolloutReport(ctx context.Context) *RolloutReportDetails {
	h.mu.RLock()
	producer := h.rolloutReportProducer
	h.mu.RUnlock()

	if producer == nil {
		return nil
	}
	return producer.BuildRolloutReport(ctx)
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
		status.Error = "MongoDB health check failed"
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
		status.Error = "PostgreSQL health check failed"
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

	if h.cachePlugin == nil {
		status.Status = "healthy"
		status.Details = map[string]any{
			"posture": "cache-plugin-not-wired",
		}
		status.ResponseTime = time.Since(start).Milliseconds()
		return status
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := h.cachePlugin.HealthCheck(pingCtx); err != nil {
		status.Status = "degraded"
		status.Error = "Redis health check failed"
	}

	status.ResponseTime = time.Since(start).Milliseconds()
	return status
}

// checkKafkaHealth checks the health of the Kafka message queue
func (h *HealthCheckHandler) checkKafkaHealth(ctx context.Context) *ComponentStatus {
	start := time.Now()
	status := &ComponentStatus{
		Name:      "Kafka",
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}

	// HealthCheck via type assertion (MQPlugin interface doesn't require Health)
	if hc, ok := h.mqPlugin.(interface{ Health(context.Context) error }); ok {
		if err := hc.Health(ctx); err != nil {
			status.Status = "unhealthy"
			status.Error = "Kafka health check failed"
			h.logger.Error("Kafka health check failed", "error", err.Error())
		}
	}

	// Structured details via optional DetailedHealth method
	if dh, ok := h.mqPlugin.(interface{ DetailedHealth(context.Context) core.HealthStatus }); ok {
		detailed := dh.DetailedHealth(ctx)
		status.Details = detailed.Details
		if detailed.Status != "healthy" && status.Status == "healthy" {
			status.Status = detailed.Status
		}
	}

	status.ResponseTime = time.Since(start).Milliseconds()
	return status
}

// respondJSON responds with JSON data
func (h *HealthCheckHandler) respondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err.Error())
	}
}

// respondError responds with an error message
func (h *HealthCheckHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	code := "INTERNAL_SERVER_ERROR"
	if statusCode == http.StatusServiceUnavailable {
		code = "SERVICE_UNAVAILABLE"
	}
	(&APIError{Code: code, Message: message, Status: statusCode}).WriteHTTP(w)
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
