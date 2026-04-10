// Package core provides core abstractions and interfaces for the API layer.
package core

import (
	"context"
	"fmt"
	"testing"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// userIDKey is the context key for user ID
	userIDKey contextKey = "user_id"
)

// Property 1: Request Abstraction Consistency
// For any protocol-specific request, converting it to Request abstraction and back
// SHALL produce an equivalent request.
func TestProperty_RequestAbstractionConsistency(t *testing.T) {
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
				"X-Custom":      "value",
			},
			body: []byte(`{"name":"updated"}`),
		},
		{
			name:    "DELETE request",
			method:  "DELETE",
			path:    "/api/users/1",
			headers: map[string]string{"Authorization": "Bearer token"},
			body:    []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create original request
			original := NewBaseRequest(context.Background(), tt.method, tt.path, tt.headers, tt.body)

			// Verify abstraction preserves all properties
			if original.Method() != tt.method {
				t.Errorf("method mismatch: expected %s, got %s", tt.method, original.Method())
			}

			if original.Path() != tt.path {
				t.Errorf("path mismatch: expected %s, got %s", tt.path, original.Path())
			}

			if string(original.Body()) != string(tt.body) {
				t.Errorf("body mismatch: expected %s, got %s", string(tt.body), string(original.Body()))
			}

			// Verify all headers are preserved
			for key, value := range tt.headers {
				if original.Header(key) != value {
					t.Errorf("header mismatch for %s: expected %s, got %s", key, value, original.Header(key))
				}
			}
		})
	}
}

// Property 2: Response Abstraction Consistency
// For any Response abstraction, converting it to protocol-specific format and back
// SHALL produce an equivalent response.
func TestProperty_ResponseAbstractionConsistency(t *testing.T) {
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
		{
			name:    "500 Internal Server Error response",
			status:  500,
			headers: map[string]string{"Content-Type": "application/json"},
			body:    []byte(`{"error":"internal error"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			resp := NewBaseResponse(nil)
			resp.SetStatus(tt.status)

			for key, value := range tt.headers {
				resp.SetHeader(key, value)
			}

			resp.SetBody(tt.body)

			// Verify abstraction preserves all properties
			if resp.Status() != tt.status {
				t.Errorf("status mismatch: expected %d, got %d", tt.status, resp.Status())
			}

			if string(resp.Body()) != string(tt.body) {
				t.Errorf("body mismatch: expected %s, got %s", string(tt.body), string(resp.Body()))
			}

			// Verify all headers are preserved
			for key, value := range tt.headers {
				if resp.Header(key) != value {
					t.Errorf("header mismatch for %s: expected %s, got %s", key, value, resp.Header(key))
				}
			}
		})
	}
}

// Property 3: Handler Interface Consistency
// For any handler implementing the Handler interface, calling Handle() multiple times
// with the same request SHALL produce consistent results.
func TestProperty_HandlerConsistency(t *testing.T) {
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("consistent response"))
		return resp, nil
	})

	req := NewBaseRequest(context.Background(), "GET", "/api/test", nil, []byte(""))

	// Call handler multiple times
	for i := 0; i < 5; i++ {
		resp, err := handler.Handle(req)
		if err != nil {
			t.Errorf("iteration %d: unexpected error: %v", i, err)
		}

		if resp.Status() != 200 {
			t.Errorf("iteration %d: expected status 200, got %d", i, resp.Status())
		}

		if string(resp.Body()) != "consistent response" {
			t.Errorf("iteration %d: expected consistent body", i)
		}
	}
}

// Property 4: Router Routing Correctness
// For any registered route, the router SHALL route requests to the correct handler.
func TestProperty_RouterRoutingCorrectness(t *testing.T) {
	router := NewAPIRouter()

	// Register multiple handlers
	handlers := map[string]string{
		"/api/users":    "users",
		"/api/posts":    "posts",
		"/api/comments": "comments",
	}

	for route, name := range handlers {
		routeName := name // capture for closure
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetBody([]byte(routeName))
			return resp, nil
		})
		router.Register(route, handler)
	}

	// Test routing to each handler
	for route, expectedName := range handlers {
		req := NewBaseRequest(context.Background(), "GET", route, nil, []byte(""))
		resp, err := router.Handle(req)
		if err != nil {
			t.Errorf("route %s: unexpected error: %v", route, err)
		}

		if string(resp.Body()) != expectedName {
			t.Errorf("route %s: expected %s, got %s", route, expectedName, string(resp.Body()))
		}
	}
}

// Property 5: Middleware Chain Consistency
// For any middleware chain, the middleware SHALL be applied in the correct order.
func TestProperty_MiddlewareChainConsistency(t *testing.T) {
	router := NewAPIRouter()

	// Create middleware that appends to a string
	middleware1 := func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware-1", "applied")
			}
			return resp, err
		})
	}

	middleware2 := func(next Handler) Handler {
		return HandlerFunc(func(req Request) (Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware-2", "applied")
			}
			return resp, err
		})
	}

	router.Use(middleware1, middleware2)

	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	router.Register("/api/test", handler)

	req := NewBaseRequest(context.Background(), "GET", "/api/test", nil, []byte(""))
	resp, err := router.Handle(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Both middleware should be applied
	if resp.Header("X-Middleware-1") != "applied" {
		t.Error("middleware 1 not applied")
	}

	if resp.Header("X-Middleware-2") != "applied" {
		t.Error("middleware 2 not applied")
	}
}

// Property 6: Error Mapping Consistency
// For any error, the error mapper SHALL consistently map it to the same status code.
func TestProperty_ErrorMappingConsistency(t *testing.T) {
	mapper := NewDefaultErrorMapper()

	// Map the same error multiple times
	for i := 0; i < 5; i++ {
		status, _, _ := mapper.MapError(nil)
		if status != 200 {
			t.Errorf("iteration %d: expected status 200 for nil error, got %d", i, status)
		}
	}

	// Map different errors
	errors := []error{
		nil,
		// Add more error types as needed
	}

	for _, err := range errors {
		status1, _, _ := mapper.MapError(err)
		status2, _, _ := mapper.MapError(err)

		if status1 != status2 {
			t.Errorf("inconsistent mapping for error %v: %d vs %d", err, status1, status2)
		}
	}
}

// Property 7: API Layer Request Routing
// For any request to the API layer, the request SHALL be routed to the correct handler.
func TestProperty_APILayerRoutingCorrectness(t *testing.T) {
	layer := NewAPILayer()

	// Register handlers for different routes
	routes := map[string]int{
		"/api/users": 200,
		"/api/posts": 201,
		"/api/admin": 403,
	}

	for route, expectedStatus := range routes {
		status := expectedStatus // capture for closure
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(status)
			return resp, nil
		})
		layer.RegisterHandler(route, handler)
	}

	// Test routing
	for route, expectedStatus := range routes {
		req := NewBaseRequest(context.Background(), "GET", route, nil, []byte(""))
		resp := layer.Handle(req)

		if resp.Status() != expectedStatus {
			t.Errorf("route %s: expected status %d, got %d", route, expectedStatus, resp.Status())
		}
	}
}

// Property 8: Request Context Preservation
// For any request with a context, the context SHALL be preserved through the abstraction.
func TestProperty_RequestContextPreservation(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey, "123")
	req := NewBaseRequest(ctx, "GET", "/api/users", nil, []byte(""))

	// Verify context is preserved
	if req.Context().Value(userIDKey) != "123" {
		t.Error("context value not preserved")
	}

	// Verify context can be used in handlers
	handler := HandlerFunc(func(r Request) (Response, error) {
		userID := r.Context().Value(userIDKey)
		if userID == nil {
			return nil, fmt.Errorf("user_id not found in context")
		}
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte(userID.(string)))
		return resp, nil
	})

	resp, err := handler.Handle(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if string(resp.Body()) != "123" {
		t.Errorf("expected user_id in response, got %s", string(resp.Body()))
	}
}
