package grpc

import (
	"context"
	"encoding/json"
	"testing"

	"chainpulse/pkg/plugins/api/core"
)

func skipGRPCPropertyLifecycleTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping gRPC property lifecycle test in short mode")
	}
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// userIDKey is the key for storing user ID in context.Context
	userIDKey contextKey = "user_id"
)

// Property 1: gRPC Request Abstraction Consistency
// For any gRPC request, converting it to Request abstraction
// SHALL preserve all properties.
func TestProperty_GRPCRequestAbstractionConsistency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		body    []byte
	}{
		{
			name:    "simple query request",
			method:  "POST",
			path:    "/api.Query/Execute",
			headers: map[string]string{"Authorization": "Bearer token"},
			body:    []byte(`{"query":"SELECT * FROM users"}`),
		},
		{
			name:   "request with multiple headers",
			method: "POST",
			path:   "/api.Admin/GetStatus",
			headers: map[string]string{
				"Authorization": "Bearer token",
				"X-Request-ID":  "req-123",
			},
			body: []byte(`{"action":"status"}`),
		},
		{
			name:    "empty body request",
			method:  "POST",
			path:    "/api.Service/Ping",
			headers: map[string]string{},
			body:    []byte("{}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create gRPC request
			req := NewGRPCRequest(tt.method, tt.path, tt.headers, tt.body, context.Background())

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

			if string(req.Body()) != string(tt.body) {
				t.Errorf("body mismatch: expected %s, got %s", string(tt.body), string(req.Body()))
			}
		})
	}
}

// Property 2: gRPC Response Abstraction Consistency
// For any gRPC response, converting it to Response abstraction
// SHALL preserve all properties.
func TestProperty_GRPCResponseAbstractionConsistency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		body    []byte
	}{
		{
			name:    "200 OK response",
			status:  200,
			headers: map[string]string{"X-Message-ID": "msg-123"},
			body:    []byte(`{"status":"ok"}`),
		},
		{
			name:    "201 Created response",
			status:  201,
			headers: map[string]string{"X-Resource-ID": "res-456"},
			body:    []byte(`{"id":1}`),
		},
		{
			name:    "400 Bad Request response",
			status:  400,
			headers: map[string]string{},
			body:    []byte(`{"error":"invalid request"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			resp := NewGRPCResponse()

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

// Property 3: gRPC Request Context Preservation
// For any gRPC request with context, the context SHALL be preserved.
func TestProperty_GRPCRequestContextPreservation(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), userIDKey, "user-123")
	req := NewGRPCRequest("POST", "/api.Service/Method", nil, []byte("{}"), ctx)

	// Verify context is preserved
	if req.Context().Value(userIDKey) != "user-123" {
		t.Error("context value not preserved")
	}
}

// Property 4: gRPC Response JSON Serialization Round Trip
// For any gRPC response, serializing to JSON and deserializing
// SHALL produce an equivalent response.
func TestProperty_GRPCResponseJSONRoundTrip(t *testing.T) {
	t.Parallel()
	original := NewGRPCResponse()
	original.SetStatus(201)
	original.SetHeader("X-Message-ID", "msg-123")
	original.SetHeader("X-Timestamp", "2024-01-10")
	original.SetBody([]byte(`{"id":1,"name":"test"}`))

	// Serialize to JSON
	data, err := original.ToJSON()
	if err != nil {
		t.Errorf("unexpected error serializing: %v", err)
		return
	}

	// Deserialize from JSON
	restored := NewGRPCResponse()
	err = restored.FromJSON(data)
	if err != nil {
		t.Errorf("unexpected error deserializing: %v", err)
		return
	}

	// Verify properties match
	if restored.Status() != original.Status() {
		t.Errorf("status mismatch: expected %d, got %d", original.Status(), restored.Status())
	}

	if string(restored.Body()) != string(original.Body()) {
		t.Errorf("body mismatch: expected %s, got %s", string(original.Body()), string(restored.Body()))
	}

	for key, value := range original.Headers() {
		if restored.Header(key) != value {
			t.Errorf("header mismatch for %s: expected %s, got %s", key, value, restored.Header(key))
		}
	}
}

// Property 5: gRPC Response Body Accumulation
// For any gRPC response, multiple Write() calls SHALL accumulate in the body.
func TestProperty_GRPCResponseBodyAccumulation(t *testing.T) {
	t.Parallel()
	resp := NewGRPCResponse()

	// Write multiple times
	_, _ = resp.Write([]byte(`{"event":"start"`))
	_, _ = resp.Write([]byte(`,"data":[1,2,3]}`))

	expected := `{"event":"start","data":[1,2,3]}`
	if string(resp.Body()) != expected {
		t.Errorf("expected %s, got %s", expected, string(resp.Body()))
	}
}

// Property 6: gRPC Response Header Immutability After Send
// For any gRPC response, headers SHALL NOT be modifiable after send.
func TestProperty_GRPCResponseHeaderImmutabilityAfterSend(t *testing.T) {
	t.Parallel()
	resp := NewGRPCResponse()

	resp.SetStatus(200)
	resp.SetHeader("X-Original", "value")
	if err := resp.Send(); err != nil {
		t.Fatalf("failed to send response: %v", err)
	}

	// Try to modify after send
	resp.SetStatus(500)
	resp.SetHeader("X-New", "value")

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.Header("X-New") != "" {
		t.Error("expected header not to be set after send")
	}
}

// Property 7: gRPC Plugin Lifecycle Management
// For any gRPC plugin, start/stop operations SHALL be idempotent and thread-safe.
func TestProperty_GRPCPluginLifecycleManagement(t *testing.T) {
	t.Parallel()
	skipGRPCPropertyLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9100, apiLayer)

	// Initial state
	if plugin.IsRunning() {
		t.Error("expected plugin to not be running initially")
	}

	// Start
	err := plugin.Start()
	if err != nil {
		t.Errorf("unexpected error on start: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running after start")
	}

	// Stop
	err = plugin.Stop()
	if err != nil {
		t.Errorf("unexpected error on stop: %v", err)
	}

	if plugin.IsRunning() {
		t.Error("expected plugin to not be running after stop")
	}

	// Verify can start again
	err = plugin.Start()
	if err != nil {
		t.Errorf("unexpected error on second start: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running after second start")
	}

	_ = plugin.Stop()
}

// Property 8: gRPC Response JSON Envelope Structure
// For any gRPC response, the JSON envelope SHALL contain status, headers, and body.
func TestProperty_GRPCResponseJSONEnvelopeStructure(t *testing.T) {
	t.Parallel()
	resp := NewGRPCResponse()

	resp.SetStatus(200)
	resp.SetHeader("X-Message-ID", "msg-123")
	resp.SetBody([]byte(`{"data":"test"}`))

	data, err := resp.ToJSON()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Verify envelope structure
	var envelope map[string]any
	err = json.Unmarshal(data, &envelope)
	if err != nil {
		t.Errorf("unexpected error unmarshaling envelope: %v", err)
		return
	}

	// Verify all required fields
	if _, ok := envelope["status"]; !ok {
		t.Error("expected 'status' field in envelope")
	}

	if _, ok := envelope["headers"]; !ok {
		t.Error("expected 'headers' field in envelope")
	}

	if _, ok := envelope["body"]; !ok {
		t.Error("expected 'body' field in envelope")
	}
}
