package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDLQHandler(t *testing.T) {
	t.Parallel()

	h := NewDLQHandler(nil, nil, nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.db != nil {
		t.Error("expected nil db")
	}
}

func TestDLQHandler_HandleListDLQEvents_NilDB(t *testing.T) {
	t.Parallel()

	h := NewDLQHandler(nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/dlq", nil)
	rec := httptest.NewRecorder()

	h.HandleListDLQEvents(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestDLQHandler_HandleReplayDLQEvents_NilDB(t *testing.T) {
	t.Parallel()

	h := NewDLQHandler(nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/dlq/replay", nil)
	rec := httptest.NewRecorder()

	h.HandleReplayDLQEvents(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestWriteJSONError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSONError(rec, http.StatusBadRequest, "test error")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["statusCode"] != float64(http.StatusBadRequest) {
		t.Errorf("expected status %d in body, got %v", http.StatusBadRequest, body["statusCode"])
	}
	if body["error"] != "DLQ_ERROR" {
		t.Errorf("expected error code DLQ_ERROR, got %v", body["error"])
	}
}

func TestWriteJSONError_InternalServerError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSONError(rec, http.StatusInternalServerError, "internal failure")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestWriteJSONResponse(t *testing.T) {
	t.Parallel()

	data := map[string]string{"key": "value"}
	rec := httptest.NewRecorder()
	writeJSONResponse(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["key"] != "value" {
		t.Errorf("expected body key=value, got %v", body)
	}
}

func TestWriteJSONResponse_Created(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSONResponse(rec, http.StatusCreated, map[string]int{"id": 42})

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestWriteJSONResponse_ArrayData(t *testing.T) {
	t.Parallel()

	data := []string{"a", "b", "c"}
	rec := httptest.NewRecorder()
	writeJSONResponse(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body []string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if len(body) != 3 {
		t.Errorf("expected 3 items, got %d", len(body))
	}
}
