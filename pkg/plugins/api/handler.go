package api

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure atomic types are properly imported
var _ atomic.Int64

// RequestHandler represents a request handler instance
type RequestHandler struct {
	ID            string
	Name          string
	Endpoint      string
	Available     bool
	Weight        int
	Metrics       *HandlerMetrics
	CreatedAt     time.Time
	UpdatedAt     time.Time
	healthClient  *http.Client
	healthHeaders map[string]string
	mu            sync.RWMutex
	lastCheckAt   time.Time
	checkInterval time.Duration
}

// HandlerMetrics represents handler metrics
type HandlerMetrics struct {
	RequestCount  int64
	SuccessCount  int64
	ErrorCount    int64
	AvgLatency    int64
	LastError     string
	LastErrorTime time.Time
	mu            sync.RWMutex
}

// NewHandler creates a new handler
func NewHandler(id, name, endpoint string) *RequestHandler {
	return &RequestHandler{
		ID:        id,
		Name:      name,
		Endpoint:  endpoint,
		Available: true,
		Weight:    1,
		Metrics:   NewHandlerMetrics(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		healthClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		checkInterval: 30 * time.Second,
	}
}

// NewHandlerMetrics creates new handler metrics
func NewHandlerMetrics() *HandlerMetrics {
	return &HandlerMetrics{
		RequestCount: 0,
		SuccessCount: 0,
		ErrorCount:   0,
		AvgLatency:   0,
	}
}

// RecordRequest records a request
func (h *RequestHandler) RecordRequest(duration time.Duration, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Metrics.mu.Lock()
	defer h.Metrics.mu.Unlock()

	atomic.AddInt64(&h.Metrics.RequestCount, 1)

	if success {
		atomic.AddInt64(&h.Metrics.SuccessCount, 1)
	} else {
		atomic.AddInt64(&h.Metrics.ErrorCount, 1)
	}

	// Update average latency
	totalLatency := h.Metrics.AvgLatency * (atomic.LoadInt64(&h.Metrics.RequestCount) - 1)
	totalLatency += duration.Milliseconds()
	h.Metrics.AvgLatency = totalLatency / atomic.LoadInt64(&h.Metrics.RequestCount)

	h.UpdatedAt = time.Now()
}

// RecordError records an error
func (h *RequestHandler) RecordError(errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Metrics.mu.Lock()
	defer h.Metrics.mu.Unlock()

	h.Metrics.LastError = errMsg
	h.Metrics.LastErrorTime = time.Now()
	atomic.AddInt64(&h.Metrics.ErrorCount, 1)

	h.UpdatedAt = time.Now()
}

// CheckHealth checks if the handler is healthy
func (h *RequestHandler) CheckHealth() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if we need to perform a health check
	if time.Since(h.lastCheckAt) < h.checkInterval {
		return h.Available
	}

	client := h.healthClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequest(http.MethodGet, h.Endpoint+"/health", nil)
	if err != nil {
		h.Available = false
		h.lastCheckAt = time.Now()
		return false
	}
	for key, value := range h.healthHeaders {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		h.Available = false
		h.lastCheckAt = time.Now()
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	h.Available = resp.StatusCode >= 200 && resp.StatusCode < 300
	h.lastCheckAt = time.Now()
	return h.Available
}

// SetHealthHTTPClient overrides the HTTP client used for handler health checks.
//
//nolint:wsl // Setter keeps the health client wiring explicit and simple.
func (h *RequestHandler) SetHealthHTTPClient(client *http.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client == nil {
		h.healthClient = &http.Client{Timeout: 5 * time.Second}

		return
	}
	h.healthClient = client
}

// SetHealthHeaders configures static headers used during handler health checks.
func (h *RequestHandler) SetHealthHeaders(headers map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(headers) == 0 {
		h.healthHeaders = nil
		return
	}

	copied := make(map[string]string, len(headers))
	for key, value := range headers {
		copied[key] = value
	}
	h.healthHeaders = copied
}

// SetAvailable sets the availability status
func (h *RequestHandler) SetAvailable(available bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Available = available
	h.UpdatedAt = time.Now()
}

// IsAvailable returns the availability status
func (h *RequestHandler) IsAvailable() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.Available
}

// GetMetrics returns a copy of the metrics
func (h *RequestHandler) GetMetrics() HandlerMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.Metrics.mu.RLock()
	defer h.Metrics.mu.RUnlock()

	return HandlerMetrics{
		RequestCount:  atomic.LoadInt64(&h.Metrics.RequestCount),
		SuccessCount:  atomic.LoadInt64(&h.Metrics.SuccessCount),
		ErrorCount:    atomic.LoadInt64(&h.Metrics.ErrorCount),
		AvgLatency:    h.Metrics.AvgLatency,
		LastError:     h.Metrics.LastError,
		LastErrorTime: h.Metrics.LastErrorTime,
	}
}

// GetSuccessRate returns the success rate as a percentage
func (h *RequestHandler) GetSuccessRate() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.Metrics.mu.RLock()
	defer h.Metrics.mu.RUnlock()

	totalRequests := atomic.LoadInt64(&h.Metrics.RequestCount)
	if totalRequests == 0 {
		return 100.0
	}

	successCount := atomic.LoadInt64(&h.Metrics.SuccessCount)
	return (float64(successCount) / float64(totalRequests)) * 100.0
}

// String returns a string representation of the handler
func (h *RequestHandler) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return fmt.Sprintf("Handler{ID: %s, Name: %s, Endpoint: %s, Available: %v, Weight: %d}",
		h.ID, h.Name, h.Endpoint, h.Available, h.Weight)
}
