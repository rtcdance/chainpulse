package core

import (
	"context"
	"testing"
)

// testContextKey is a custom type for test context keys
type testContextKey string

const (
	// testKey is a test context key
	testKey testContextKey = "key"
)

func TestNewBaseRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		body    []byte
		ctx     context.Context
	}{
		{
			name:    "basic request",
			method:  "GET",
			path:    "/api/users",
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(""),
			ctx:     context.Background(),
		},
		{
			name:    "request with nil headers",
			method:  "POST",
			path:    "/api/users",
			headers: nil,
			body:    []byte(`{"name":"test"}`),
			ctx:     context.Background(),
		},
		{
			name:    "request with nil context",
			method:  "PUT",
			path:    "/api/users/1",
			headers: map[string]string{},
			body:    []byte(`{"name":"updated"}`),
			ctx:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewBaseRequest(tt.ctx, tt.method, tt.path, tt.headers, tt.body)

			if req == nil {
				t.Fatal("expected request, got nil")
			}

			if req.Method() != tt.method {
				t.Errorf("expected method %s, got %s", tt.method, req.Method())
			}

			if req.Path() != tt.path {
				t.Errorf("expected path %s, got %s", tt.path, req.Path())
			}

			if req.Context() == nil {
				t.Error("expected context, got nil")
			}
		})
	}
}

func TestBaseRequestHeaders(t *testing.T) {
	t.Parallel()
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token",
	}

	req := NewBaseRequest(context.Background(), "GET", "/api/users", headers, []byte(""))

	// Test Headers()
	if len(req.Headers()) != len(headers) {
		t.Errorf("expected %d headers, got %d", len(headers), len(req.Headers()))
	}

	// Test Header()
	if req.Header("Content-Type") != "application/json" {
		t.Error("expected Content-Type header")
	}

	if req.Header("Authorization") != "Bearer token" {
		t.Error("expected Authorization header")
	}

	if req.Header("Non-Existent") != "" {
		t.Error("expected empty string for non-existent header")
	}
}

func TestBaseRequestBody(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name":"test"}`)
	req := NewBaseRequest(context.Background(), "POST", "/api/users", nil, body)

	if string(req.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(req.Body()))
	}
}

func TestBaseRequestQueryParams(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "GET", "/api/users", nil, []byte(""))

	// Set query params
	req.SetQuery(map[string]string{
		"page":  "1",
		"limit": "10",
	})

	if req.QueryParam("page") != "1" {
		t.Error("expected page=1")
	}

	if req.QueryParam("limit") != "10" {
		t.Error("expected limit=10")
	}

	if req.QueryParam("non-existent") != "" {
		t.Error("expected empty string for non-existent param")
	}

	query := req.Query()
	if len(query) != 2 {
		t.Errorf("expected 2 query params, got %d", len(query))
	}
	if query["page"] != "1" {
		t.Error("expected page=1 in Query()")
	}
}

func TestBaseRequestPathParams(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "GET", "/api/users/:id", nil, []byte(""))

	// Set path params
	req.SetPathParam("id", "123")

	if req.PathParam("id") != "123" {
		t.Error("expected id=123")
	}

	if req.PathParam("non-existent") != "" {
		t.Error("expected empty string for non-existent param")
	}
}

func TestBaseRequestContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), testKey, "value")
	req := NewBaseRequest(ctx, "GET", "/api/users", nil, []byte(""))

	if req.Context().Value(testKey) != "value" {
		t.Error("expected context value")
	}
}

func TestBaseRequestRuntimeMetricsStaged(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "GET", "/health", nil, nil)

	metrics := req.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-minimal" {
		t.Fatalf("expected request-minimal, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-staged" {
		t.Fatalf("expected request-staged, got %v", metrics["runtime_posture"])
	}
}

func TestBaseRequestRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "POST", "/api/users/123", map[string]string{
		"Content-Type": "application/json",
	}, []byte(`{"name":"alice"}`))
	req.SetQuery(map[string]string{"verbose": "true"})
	req.SetPathParam("id", "123")

	metrics := req.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-parameterized" {
		t.Fatalf("expected request-parameterized, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-ready" {
		t.Fatalf("expected request-ready, got %v", metrics["runtime_posture"])
	}
}

func TestBaseRequestRuntimeMetricsDegraded(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "", "", nil, nil)

	metrics := req.GetRuntimeMetrics()
	if metrics["runtime_posture"] != "request-degraded" {
		t.Fatalf("expected request-degraded, got %v", metrics["runtime_posture"])
	}
}

func TestBaseRequestImplementsInterface(t *testing.T) {
	t.Parallel()
	req := NewBaseRequest(context.Background(), "GET", "/api/users", nil, []byte(""))

	// Verify it implements Request interface
	var _ Request = req
}
