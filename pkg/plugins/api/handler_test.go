package api

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test-handler", "http://localhost:8080")
	if h.ID != "h1" {
		t.Errorf("expected ID h1, got %s", h.ID)
	}
	if h.Name != "test-handler" {
		t.Errorf("expected Name test-handler, got %s", h.Name)
	}
	if !h.Available {
		t.Error("expected Available to be true")
	}
	if h.Weight != 1 {
		t.Errorf("expected Weight 1, got %d", h.Weight)
	}
	if h.Metrics == nil {
		t.Error("expected Metrics to be initialized")
	}
}

func TestNewHandlerMetrics(t *testing.T) {
	t.Parallel()
	m := NewHandlerMetrics()
	if m.RequestCount != 0 {
		t.Errorf("expected 0 requests, got %d", m.RequestCount)
	}
}

func TestRequestHandler_RecordRequest_Success(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.RecordRequest(100*time.Millisecond, true)
	if h.Metrics.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", h.Metrics.RequestCount)
	}
	if h.Metrics.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", h.Metrics.SuccessCount)
	}
}

func TestRequestHandler_RecordRequest_Failure(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.RecordRequest(50*time.Millisecond, false)
	if h.Metrics.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", h.Metrics.ErrorCount)
	}
}

func TestRequestHandler_RecordError(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.RecordError("connection refused")
	if h.Metrics.LastError != "connection refused" {
		t.Errorf("expected 'connection refused', got %s", h.Metrics.LastError)
	}
	if h.Metrics.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", h.Metrics.ErrorCount)
	}
}

func TestRequestHandler_SetAvailable(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.SetAvailable(false)
	if h.IsAvailable() {
		t.Error("expected unavailable")
	}
	h.SetAvailable(true)
	if !h.IsAvailable() {
		t.Error("expected available")
	}
}

func TestRequestHandler_IsAvailable(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	if !h.IsAvailable() {
		t.Error("expected available by default")
	}
}

func TestRequestHandler_GetSuccessRate(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	rate := h.GetSuccessRate()
	if rate != 100.0 {
		t.Errorf("expected 100%% success rate with no requests, got %f", rate)
	}

	h.RecordRequest(100*time.Millisecond, true)
	h.RecordRequest(100*time.Millisecond, false)
	rate = h.GetSuccessRate()
	if rate != 50.0 {
		t.Errorf("expected 50%% success rate, got %f", rate)
	}
}

func TestRequestHandler_GetMetrics(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.RecordRequest(100*time.Millisecond, true)
	m := h.GetMetrics()
	if m.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", m.RequestCount)
	}
	m.RequestCount = 999
	if h.Metrics.RequestCount != 1 {
		t.Error("GetMetrics should return a copy")
	}
}

func TestRequestHandler_String(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	s := h.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestRequestHandler_SetHealthHTTPClient(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	custom := &http.Client{Timeout: 10 * time.Second}
	h.SetHealthHTTPClient(custom)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.healthClient != custom {
		t.Error("expected custom health client")
	}
}

func TestRequestHandler_SetHealthHTTPClient_Nil(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.SetHealthHTTPClient(nil)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.healthClient == nil {
		t.Error("expected default health client, got nil")
	}
}

func TestRequestHandler_SetHealthHeaders(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	headers := map[string]string{"Authorization": "Bearer token"}
	h.SetHealthHeaders(headers)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.healthHeaders["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization header, got %v", h.healthHeaders)
	}
}

func TestRequestHandler_SetHealthHeaders_Empty(t *testing.T) {
	t.Parallel()
	h := NewHandler("h1", "test", "http://localhost")
	h.SetHealthHeaders(map[string]string{})
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.healthHeaders != nil {
		t.Error("expected nil health headers")
	}
}
