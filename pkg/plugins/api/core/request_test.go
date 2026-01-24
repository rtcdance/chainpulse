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
			req := NewBaseRequest(tt.method, tt.path, tt.headers, tt.body, tt.ctx)

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
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token",
	}

	req := NewBaseRequest("GET", "/api/users", headers, []byte(""), context.Background())

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
	body := []byte(`{"name":"test"}`)
	req := NewBaseRequest("POST", "/api/users", nil, body, context.Background())

	if string(req.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(req.Body()))
	}
}

func TestBaseRequestQueryParams(t *testing.T) {
	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())

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
}

func TestBaseRequestPathParams(t *testing.T) {
	req := NewBaseRequest("GET", "/api/users/:id", nil, []byte(""), context.Background())

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
	ctx := context.WithValue(context.Background(), testKey, "value")
	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), ctx)

	if req.Context().Value(testKey) != "value" {
		t.Error("expected context value")
	}
}

func TestBaseRequestImplementsInterface(t *testing.T) {
	req := NewBaseRequest("GET", "/api/users", nil, []byte(""), context.Background())

	// Verify it implements Request interface
	var _ Request = req
}
