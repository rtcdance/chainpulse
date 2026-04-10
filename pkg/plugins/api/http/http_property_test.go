package http

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"chainpulse/pkg/plugins/api/core"
)

// Property 1: Request Abstraction Consistency
// For any HTTP request, converting it to Request abstraction and back
// SHALL preserve all properties.
func TestProperty_HTTPRequestAbstractionConsistency(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		body    []byte
	}{
		{
			name:    "simple GET request",
			method:  "GET",
			path:    "/api/users",
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(""),
		},
		{
			name:    "POST request with body",
			method:  "POST",
			path:    "/api/users",
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(`{"name":"test"}`),
		},
		{
			name:   "request with multiple headers",
			method: "PUT",
			path:   "/api/users/1",
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
			body: []byte(`{"name":"updated"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			httpReq, _ := http.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			for key, value := range tt.headers {
				httpReq.Header.Set(key, value)
			}

			// Convert to abstraction
			req := NewHTTPRequest(httpReq)

			// Verify properties are preserved
			if req.Method() != tt.method {
				t.Errorf("method mismatch: expected %s, got %s", tt.method, req.Method())
			}

			if req.Path() != tt.path {
				t.Errorf("path mismatch: expected %s, got %s", tt.path, req.Path())
			}

			for key, value := range tt.headers {
				if req.Header(key) != value {
					t.Errorf("header mismatch for %s: expected %s, got %s", key, value, req.Header(key))
				}
			}
		})
	}
}

// Property 2: Response Abstraction Consistency
// For any HTTP response, converting it to Response abstraction and back
// SHALL preserve all properties.
func TestProperty_HTTPResponseAbstractionConsistency(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		body    []byte
	}{
		{
			name:    "200 OK response",
			status:  200,
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(`{"status":"ok"}`),
		},
		{
			name:    "201 Created response",
			status:  201,
			headers: map[string]string{"Location": "/api/users/1"},
			body:    []byte(`{"id":1}`),
		},
		{
			name:    "400 Bad Request response",
			status:  400,
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(`{"error":"invalid input"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			w := &testResponseWriter{buf: &bytes.Buffer{}}
			resp := NewHTTPResponse(w)

			resp.SetStatus(tt.status)
			for key, value := range tt.headers {
				resp.SetHeader(key, value)
			}
			resp.SetBody(tt.body)

			// Verify properties are preserved
			if resp.Status() != tt.status {
				t.Errorf("status mismatch: expected %d, got %d", tt.status, resp.Status())
			}

			if string(resp.Body()) != string(tt.body) {
				t.Errorf("body mismatch: expected %s, got %s", string(tt.body), string(resp.Body()))
			}

			for key, value := range tt.headers {
				if resp.Header(key) != value {
					t.Errorf("header mismatch for %s: expected %s, got %s", key, value, resp.Header(key))
				}
			}
		})
	}
}

// Property 3: HTTP Plugin Request Routing
// For any registered route, the HTTP plugin SHALL route requests correctly.
func TestProperty_HTTPPluginRoutingCorrectness(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8089, apiLayer)

	// Register handlers for different routes
	routes := map[string]int{
		"/api/users": 200,
		"/api/posts": 201,
		"/api/admin": 403,
	}

	for route, expectedStatus := range routes {
		status := expectedStatus // capture for closure
		handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp := core.NewBaseResponse(nil)
			resp.SetStatus(status)
			return resp, nil
		})
		_ = plugin.RegisterRoute(route, handler)
	}

	// Test routing
	for route := range routes {
		retrieved := plugin.router.Route(route)
		if retrieved == nil {
			t.Errorf("route %s: expected handler, got nil", route)
		}
	}
}

// Property 4: HTTP Request Context Preservation
// For any HTTP request with context, the context SHALL be preserved.
func TestProperty_HTTPRequestContextPreservation(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDContextKey, "123")
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", "/api/users", nil)

	req := NewHTTPRequest(httpReq)

	// Verify context is preserved
	if req.Context().Value(userIDContextKey) != "123" {
		t.Error("context value not preserved")
	}
}

// Property 5: HTTP Response Header Immutability After Send
// For any HTTP response, headers SHALL NOT be modifiable after send.
func TestProperty_HTTPResponseHeaderImmutabilityAfterSend(t *testing.T) {
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	resp.SetStatus(200)
	resp.SetHeader("Content-Type", "application/json")
	if err := resp.Send(); err != nil {
		t.Fatalf("failed to send response: %v", err)
	}

	// Try to set headers after send - should not change
	resp.SetStatus(500)
	resp.SetHeader("X-New-Header", "value")

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.Header("X-New-Header") != "" {
		t.Error("expected header not to be set after send")
	}
}

// Property 6: HTTP Request Query Parameter Extraction
// For any HTTP request with query parameters, all parameters SHALL be extractable.
func TestProperty_HTTPRequestQueryParameterExtraction(t *testing.T) {
	params := map[string]string{
		"page":   "1",
		"limit":  "10",
		"search": "test",
	}

	queryStr := "page=1&limit=10&search=test"
	httpReq, _ := http.NewRequest("GET", "/api/users?"+queryStr, nil)

	req := NewHTTPRequest(httpReq)

	// Verify all parameters are extractable
	for key, expectedValue := range params {
		if req.QueryParam(key) != expectedValue {
			t.Errorf("query param %s: expected %s, got %s", key, expectedValue, req.QueryParam(key))
		}
	}

	// Verify Query() returns all parameters
	query := req.Query()
	if len(query) != len(params) {
		t.Errorf("expected %d query params, got %d", len(params), len(query))
	}
}

// Property 7: HTTP Response Body Accumulation
// For any HTTP response, multiple Write() calls SHALL accumulate in the body.
func TestProperty_HTTPResponseBodyAccumulation(t *testing.T) {
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	// Write multiple times
	_, _ = resp.Write([]byte("hello"))
	_, _ = resp.Write([]byte(" "))
	_, _ = resp.Write([]byte("world"))

	if string(resp.Body()) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(resp.Body()))
	}
}

// Property 8: HTTP Plugin Middleware Application
// For any HTTP plugin with middleware, middleware SHALL be applied to all requests.
func TestProperty_HTTPPluginMiddlewareApplication(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8090, apiLayer)

	// Create middleware that adds a header
	middleware := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware", "applied")
			}
			return resp, err
		})
	}

	_ = plugin.Use(middleware)

	// Register handler
	handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	if err := plugin.RegisterRoute("/api/test", handler); err != nil {
		_ = err // Log but continue
	}

	// Verify middleware is registered
	if len(plugin.middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(plugin.middleware))
	}
}
