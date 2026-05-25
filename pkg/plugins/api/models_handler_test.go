package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestErrorCodeFromStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code     int
		expected string
	}{
		{400, "INVALID_REQUEST"},
		{401, "UNAUTHORIZED"},
		{403, "FORBIDDEN"},
		{404, "NOT_FOUND"},
		{429, "RATE_LIMIT_EXCEEDED"},
		{500, "INTERNAL_SERVER_ERROR"},
		{503, "SERVICE_UNAVAILABLE"},
		{418, "INTERNAL_SERVER_ERROR"}, // unknown code defaults to 500
		{200, "INTERNAL_SERVER_ERROR"},
		{0, "INTERNAL_SERVER_ERROR"},
	}

	for _, tt := range tests {
		result := errorCodeFromStatus(tt.code)
		if result != tt.expected {
			t.Errorf("errorCodeFromStatus(%d) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestModelsHandler_respondJSON(t *testing.T) {
	t.Parallel()
	h := &ModelsHandler{
		logger:  core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics: core.NewDefaultMetricsCollector(),
	}
	w := httptest.NewRecorder()
	h.respondJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected application/json content type")
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["hello"] != "world" {
		t.Errorf("expected hello=world, got %v", body)
	}
}

func TestModelsHandler_respondError(t *testing.T) {
	t.Parallel()
	h := &ModelsHandler{
		logger:  core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics: core.NewDefaultMetricsCollector(),
	}
	w := httptest.NewRecorder()
	h.respondError(w, http.StatusNotFound, "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	var body APIError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "resource not found" {
		t.Errorf("expected 'resource not found', got %q", body.Message)
	}
	if body.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %q", body.Code)
	}
}

func TestModelsHandler_respondError_Internal(t *testing.T) {
	t.Parallel()
	h := &ModelsHandler{
		logger:  core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics: core.NewDefaultMetricsCollector(),
	}
	w := httptest.NewRecorder()
	h.respondError(w, http.StatusInternalServerError, "internal error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestModelsHandler_Health(t *testing.T) {
	t.Parallel()
	h := &ModelsHandler{
		logger:      core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics:     core.NewDefaultMetricsCollector(),
		initialized: true,
	}
	health := h.Health(context.Background())
	if health.Status == "" {
		t.Error("expected non-empty health status")
	}
}

func TestModelsHandler_HandleModels_NotInitialized(t *testing.T) {
	t.Parallel()

	h := &ModelsHandler{
		logger:      core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics:     core.NewDefaultMetricsCollector(),
		initialized: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/models", nil)
	rec := httptest.NewRecorder()
	h.HandleModels(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when not initialized, got %d", rec.Code)
	}
}

func TestModelsHandler_HandleModels_Initialized(t *testing.T) {
	t.Parallel()

	h := &ModelsHandler{
		logger:      core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics:     core.NewDefaultMetricsCollector(),
		initialized: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/models", nil)
	rec := httptest.NewRecorder()
	h.HandleModels(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestModelsHandler_listModels(t *testing.T) {
	t.Parallel()

	h := &ModelsHandler{}
	models := h.listModels()
	if len(models) == 0 {
		t.Error("expected non-empty models list")
	}
	found := false
	for _, m := range models {
		if m.Name == "BlockchainEvent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected BlockchainEvent model in list")
	}
}

func TestModelsHandler_Health_NotInitialized(t *testing.T) {
	t.Parallel()
	h := &ModelsHandler{
		logger:      core.NewDefaultLoggerWithOutput(core.LogLevelInfo, io.Discard),
		metrics:     core.NewDefaultMetricsCollector(),
		initialized: false,
	}
	health := h.Health(context.Background())
	if health.Status == "" {
		t.Error("expected non-empty health status")
	}
}
