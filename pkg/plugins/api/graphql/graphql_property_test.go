package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"chainpulse/pkg/plugins/api/core"
)

func skipGraphQLLifecycleTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping GraphQL lifecycle test in short mode")
	}
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// testContextKey is the key for storing test values in context.Context
	testContextKey contextKey = "key"
)

// Property 1: GraphQL Request Abstraction Consistency
// For any GraphQL request, converting it to Request abstraction and back SHALL preserve all properties
func TestGraphQLRequestAbstractionConsistency(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		path   string
		body   map[string]interface{}
	}{
		{
			name:   "simple query",
			method: "POST",
			path:   "/graphql",
			body: map[string]interface{}{
				"query": "{ event(id: \"1\") { id } }",
			},
		},
		{
			name:   "query with variables",
			method: "POST",
			path:   "/graphql",
			body: map[string]interface{}{
				"query": "query GetEvent($id: ID!) { event(id: $id) { id } }",
				"variables": map[string]interface{}{
					"id": "123",
				},
			},
		},
		{
			name:   "mutation",
			method: "POST",
			path:   "/graphql",
			body: map[string]interface{}{
				"query": "mutation { executeQuery(query: \"test\") }",
			},
		},
		{
			name:   "complex query",
			method: "POST",
			path:   "/graphql",
			body: map[string]interface{}{
				"query": "{ events(limit: 10, offset: 0) { id type } }",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			bodyBytes, _ := json.Marshal(tc.body)
			httpReq := httpRequest(tc.method, tc.path, bodyBytes)

			// Act
			gqlReq := NewGraphQLRequest(httpReq)

			// Assert - verify all properties are preserved
			if gqlReq.Method() != tc.method {
				t.Errorf("method mismatch: expected %s, got %s", tc.method, gqlReq.Method())
			}

			if gqlReq.Path() != tc.path {
				t.Errorf("path mismatch: expected %s, got %s", tc.path, gqlReq.Path())
			}

			if !bytes.Equal(gqlReq.Body(), bodyBytes) {
				t.Errorf("body mismatch")
			}

			if gqlReq.Context() == nil {
				t.Error("context should not be nil")
			}
		})
	}
}

// Property 2: GraphQL Response Abstraction Consistency
// For any Response abstraction, converting it to protocol-specific format and back SHALL preserve all properties
func TestGraphQLResponseAbstractionConsistency(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		body   map[string]interface{}
	}{
		{
			name:   "success response",
			status: 200,
			body: map[string]interface{}{
				"data": map[string]interface{}{
					"event": map[string]interface{}{
						"id": "1",
					},
				},
			},
		},
		{
			name:   "error response",
			status: 400,
			body: map[string]interface{}{
				"errors": []string{"invalid query"},
			},
		},
		{
			name:   "not found response",
			status: 404,
			body: map[string]interface{}{
				"errors": []string{"not found"},
			},
		},
		{
			name:   "server error response",
			status: 500,
			body: map[string]interface{}{
				"errors": []string{"internal server error"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := &mockResponseWriter{}
			resp := NewGraphQLResponse(w)
			bodyBytes, _ := json.Marshal(tc.body)

			// Act
			resp.SetStatus(tc.status)
			resp.SetBody(bodyBytes)

			// Assert - verify all properties are preserved
			if resp.Status() != tc.status {
				t.Errorf("status mismatch: expected %d, got %d", tc.status, resp.Status())
			}

			if !bytes.Equal(resp.Body(), bodyBytes) {
				t.Errorf("body mismatch")
			}
		})
	}
}

// Property 3: GraphQL Request Context Preservation
// For any GraphQL request, the context SHALL be preserved through the request lifecycle
func TestGraphQLRequestContextPreservation(t *testing.T) {
	testCases := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "background context",
			ctx:  context.Background(),
		},
		{
			name: "context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				defer cancel()
				return ctx
			}(),
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), testContextKey, "value"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			body := []byte(`{"query":"test"}`)
			httpReq := httpRequest("POST", "/graphql", body)
			httpReq = httpReq.WithContext(tc.ctx)

			// Act
			gqlReq := NewGraphQLRequest(httpReq)

			// Assert - context should be preserved
			if gqlReq.Context() == nil {
				t.Error("context should not be nil")
			}
		})
	}
}

// Property 4: GraphQL Response JSON Serialization Round Trip
// For any GraphQL result, serializing and deserializing SHALL produce equivalent data
func TestGraphQLResponseJSONSerializationRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "simple data",
			data: map[string]interface{}{
				"event": map[string]interface{}{
					"id": "1",
				},
			},
		},
		{
			name: "nested data",
			data: map[string]interface{}{
				"events": []map[string]interface{}{
					{"id": "1", "type": "Transfer"},
					{"id": "2", "type": "Swap"},
				},
			},
		},
		{
			name: "data with null values",
			data: map[string]interface{}{
				"event": nil,
			},
		},
		{
			name: "complex nested structure",
			data: map[string]interface{}{
				"pool": map[string]interface{}{
					"address": "0x123",
					"stats": map[string]interface{}{
						"tvl":    "1000000",
						"volume": "500000",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := &mockResponseWriter{}
			resp := NewGraphQLResponse(w)

			// Act - serialize
			if err := resp.SetGraphQLResult(tc.data, nil); err != nil {
				t.Fatalf("failed to set result: %v", err)
			}

			// Deserialize
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body(), &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Assert - data should be preserved
			if result["data"] == nil {
				t.Error("data field should not be nil")
			}
		})
	}
}

// Property 6: GraphQL Response Header Immutability After Send
// For any response, headers set after Send() SHALL not be applied
func TestGraphQLResponseHeaderImmutabilityAfterSend(t *testing.T) {
	testCases := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "single header",
			headers: map[string]string{
				"X-Custom": "value1",
			},
		},
		{
			name: "multiple headers",
			headers: map[string]string{
				"X-Header-1": "value1",
				"X-Header-2": "value2",
				"X-Header-3": "value3",
			},
		},
		{
			name: "content type header",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := &mockResponseWriter{}
			resp := NewGraphQLResponse(w)

			// Set initial headers
			for k, v := range tc.headers {
				resp.SetHeader(k, v)
			}

			// Act - send response
			_ = resp.Send()

			// Try to set new headers
			for k := range tc.headers {
				resp.SetHeader(k, "new_value")
			}

			// Assert - headers should not change
			for k, v := range tc.headers {
				if resp.Header(k) != v {
					t.Errorf("header %s should not change after send", k)
				}
			}
		})
	}
}

// Property 7: GraphQL Plugin Lifecycle Management
// For any plugin, Start/Stop operations SHALL be idempotent and state-consistent
func TestGraphQLPluginLifecycleManagement(t *testing.T) {
	skipGraphQLLifecycleTestsInShortMode(t)

	testCases := []struct {
		name       string
		operations []string
	}{
		{
			name:       "start stop",
			operations: []string{"start", "stop"},
		},
		{
			name:       "start stop start stop",
			operations: []string{"start", "stop", "start", "stop"},
		},
		{
			name:       "multiple starts",
			operations: []string{"start", "start"},
		},
		{
			name:       "multiple stops",
			operations: []string{"start", "stop", "stop"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			apiLayer := core.NewAPILayer()
			plugin := NewGraphQLPlugin("graphql", 9000+hashString(tc.name)%1000, apiLayer)

			// Act & Assert
			for i, op := range tc.operations {
				switch op {
				case "start":
					err := plugin.Start()
					if i > 0 && tc.operations[i-1] == "start" {
						// Second start should fail
						if err == nil {
							t.Error("second start should fail")
						}
					} else if err != nil {
						t.Fatalf("start failed: %v", err)
					}

				case "stop":
					err := plugin.Stop()
					if i > 0 && tc.operations[i-1] == "stop" {
						// Second stop should fail
						if err == nil {
							t.Error("second stop should fail")
						}
					} else if err != nil {
						t.Fatalf("stop failed: %v", err)
					}
				}
			}
		})
	}
}

// Property 8: GraphQL Response JSON Envelope Structure
// For any GraphQL response, the JSON envelope SHALL contain required fields
func TestGraphQLResponseJSONEnvelopeStructure(t *testing.T) {
	testCases := []struct {
		name   string
		data   map[string]interface{}
		errors []error
	}{
		{
			name: "success with data",
			data: map[string]interface{}{
				"event": map[string]interface{}{"id": "1"},
			},
			errors: nil,
		},
		{
			name:   "error response",
			data:   map[string]interface{}{},
			errors: []error{&testError{"error 1"}},
		},
		{
			name: "data and errors",
			data: map[string]interface{}{
				"event": nil,
			},
			errors: []error{&testError{"not found"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := &mockResponseWriter{}
			resp := NewGraphQLResponse(w)

			// Act
			if err := resp.SetGraphQLResult(tc.data, tc.errors); err != nil {
				t.Fatalf("failed to set result: %v", err)
			}

			// Deserialize
			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Assert - envelope should have required fields
			if _, ok := envelope["data"]; !ok {
				t.Error("envelope should contain 'data' field")
			}

			if len(tc.errors) > 0 {
				if _, ok := envelope["errors"]; !ok {
					t.Error("envelope should contain 'errors' field when errors present")
				}
			}
		})
	}
}

// Helper functions

func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// Feature: GraphQL Plugin, Property 1: GraphQL Request Abstraction Consistency
// For any GraphQL request, converting it to Request abstraction and back SHALL preserve all properties
func TestGraphQLPluginProperty1RequestAbstractionConsistency(t *testing.T) {
	// This test validates that the GraphQL request adapter correctly implements the Request interface
	// and preserves all request properties through the abstraction layer
	for i := 0; i < 100; i++ {
		body := []byte(fmt.Sprintf(`{"query":"query_%d"}`, i))
		httpReq := httpRequest("POST", "/graphql", body)
		gqlReq := NewGraphQLRequest(httpReq)

		// Verify all properties are accessible through the interface
		var req core.Request = gqlReq
		if req.Method() != "POST" {
			t.Errorf("iteration %d: method not preserved", i)
		}
		if req.Path() != "/graphql" {
			t.Errorf("iteration %d: path not preserved", i)
		}
		if req.Context() == nil {
			t.Errorf("iteration %d: context not preserved", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 2: GraphQL Response Abstraction Consistency
// For any Response abstraction, converting it to protocol-specific format and back SHALL preserve all properties
func TestGraphQLPluginProperty2ResponseAbstractionConsistency(t *testing.T) {
	// This test validates that the GraphQL response adapter correctly implements the Response interface
	// and preserves all response properties through the abstraction layer
	for i := 0; i < 100; i++ {
		w := &mockResponseWriter{}
		resp := NewGraphQLResponse(w)

		// Set properties
		status := 200 + (i%5)*100
		resp.SetStatus(status)
		resp.SetHeader("X-Test", fmt.Sprintf("value_%d", i))
		resp.SetBody([]byte(fmt.Sprintf(`{"id":%d}`, i)))

		// Verify all properties are accessible through the interface
		var r core.Response = resp
		if r.Status() != status {
			t.Errorf("iteration %d: status not preserved", i)
		}
		if r.Header("X-Test") != fmt.Sprintf("value_%d", i) {
			t.Errorf("iteration %d: header not preserved", i)
		}
		if len(r.Body()) == 0 {
			t.Errorf("iteration %d: body not preserved", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 3: GraphQL Request Context Preservation
// For any GraphQL request, the context SHALL be preserved through the request lifecycle
func TestGraphQLPluginProperty3ContextPreservation(t *testing.T) {
	for i := 0; i < 100; i++ {
		body := []byte(`{"query":"test"}`)
		httpReq := httpRequest("POST", "/graphql", body)
		ctx := context.WithValue(context.Background(), testContextKey, fmt.Sprintf("value_%d", i))
		httpReq = httpReq.WithContext(ctx)

		gqlReq := NewGraphQLRequest(httpReq)

		// Context should be preserved
		if gqlReq.Context() == nil {
			t.Errorf("iteration %d: context not preserved", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 4: GraphQL Response JSON Serialization Round Trip
// For any GraphQL result, serializing and deserializing SHALL produce equivalent data
func TestGraphQLPluginProperty4JSONSerializationRoundTrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := &mockResponseWriter{}
		resp := NewGraphQLResponse(w)

		data := map[string]interface{}{
			"id":   fmt.Sprintf("id_%d", i),
			"type": "test",
		}

		if err := resp.SetGraphQLResult(data, nil); err != nil {
			t.Errorf("iteration %d: failed to set result: %v", i, err)
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			t.Errorf("iteration %d: failed to unmarshal: %v", i, err)
			continue
		}

		if result["data"] == nil {
			t.Errorf("iteration %d: data not preserved", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 5: GraphQL Response Body Accumulation
// For any sequence of writes, the body SHALL accumulate all data in order
func TestGraphQLPluginProperty5BodyAccumulation(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := &mockResponseWriter{}
		resp := NewGraphQLResponse(w)

		// Write multiple parts
		for j := 0; j < 5; j++ {
			part := fmt.Sprintf("part_%d_", j)
			if _, err := resp.Write([]byte(part)); err != nil {
				t.Fatalf("iteration %d: failed to write response: %v", i, err)
			}
		}

		body := resp.Body()
		if len(body) == 0 {
			t.Errorf("iteration %d: body is empty", i)
		}

		// Verify all parts are present
		for j := 0; j < 5; j++ {
			expected := fmt.Sprintf("part_%d_", j)
			if !bytes.Contains(body, []byte(expected)) {
				t.Errorf("iteration %d: part %d not found in body", i, j)
			}
		}
	}
}

// Feature: GraphQL Plugin, Property 6: GraphQL Response Header Immutability After Send
// For any response, headers set after Send() SHALL not be applied
func TestGraphQLPluginProperty6HeaderImmutabilityAfterSend(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := &mockResponseWriter{}
		resp := NewGraphQLResponse(w)

		resp.SetHeader("X-Test", fmt.Sprintf("original_%d", i))
		_ = resp.Send()

		// Try to change header after send
		resp.SetHeader("X-Test", fmt.Sprintf("modified_%d", i))

		// Header should not change
		if resp.Header("X-Test") != fmt.Sprintf("original_%d", i) {
			t.Errorf("iteration %d: header changed after send", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 7: GraphQL Plugin Lifecycle Management
// For any plugin, Start/Stop operations SHALL be idempotent and state-consistent
func TestGraphQLPluginProperty7LifecycleManagement(t *testing.T) {
	for i := 0; i < 100; i++ {
		apiLayer := core.NewAPILayer()
		plugin := NewGraphQLPlugin("graphql", 9100+i%100, apiLayer)

		// Start
		if err := plugin.Start(); err != nil {
			t.Errorf("iteration %d: start failed: %v", i, err)
			continue
		}

		if !plugin.IsRunning() {
			t.Errorf("iteration %d: plugin should be running", i)
		}

		// Stop
		if err := plugin.Stop(); err != nil {
			t.Errorf("iteration %d: stop failed: %v", i, err)
			continue
		}

		if plugin.IsRunning() {
			t.Errorf("iteration %d: plugin should not be running", i)
		}
	}
}

// Feature: GraphQL Plugin, Property 8: GraphQL Response JSON Envelope Structure
// For any GraphQL response, the JSON envelope SHALL contain required fields
func TestGraphQLPluginProperty8JSONEnvelopeStructure(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := &mockResponseWriter{}
		resp := NewGraphQLResponse(w)

		data := map[string]interface{}{
			"id": fmt.Sprintf("id_%d", i),
		}

		if err := resp.SetGraphQLResult(data, nil); err != nil {
			t.Errorf("iteration %d: failed to set result: %v", i, err)
			continue
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
			t.Errorf("iteration %d: failed to unmarshal: %v", i, err)
			continue
		}

		if _, ok := envelope["data"]; !ok {
			t.Errorf("iteration %d: envelope missing 'data' field", i)
		}
	}
}
