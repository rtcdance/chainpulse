package core

import (
	"context"
	"testing"
)

// TestProperty_AllProtocolsRoutingWorks tests that all protocols can route requests
func TestProperty_AllProtocolsRoutingWorks(t *testing.T) {
	// Feature: Multi-Protocol Integration, Property 1: All protocols can route requests successfully
	detector := NewProtocolDetector()

	// Register handlers for all protocols
	for i := 0; i < 4; i++ {
		protocol := ProtocolType(i)
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			return resp, nil
		})
		if err := detector.RegisterHandler(protocol, handler); err != nil {
			t.Fatalf("Failed to register handler for protocol %v: %v", protocol, err)
		}
	}

	// Test routing for each protocol
	testCases := []struct {
		path    string
		headers map[string]string
		body    []byte
	}{
		{"/api", map[string]string{}, nil},
		{"/ws", map[string]string{"Upgrade": "websocket"}, nil},
		{"/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
		{"/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
	}

	for _, tc := range testCases {
		req := NewBaseRequest("POST", tc.path, tc.headers, tc.body, context.Background())
		resp, err := detector.Route(req)

		if err != nil {
			t.Errorf("routing failed for path %s: %v", tc.path, err)
		}

		if resp == nil {
			t.Errorf("expected non-nil response for path %s", tc.path)
		}

		if resp.Status() != 200 {
			t.Errorf("expected status 200 for path %s, got %d", tc.path, resp.Status())
		}
	}
}

// TestProperty_ProtocolDetectionIsAccurate tests protocol detection accuracy
func TestProperty_ProtocolDetectionIsAccurate(t *testing.T) {
	// Feature: Protocol Detection, Property 2: Protocol detection is accurate for all protocols
	detector := NewProtocolDetector()

	testCases := []struct {
		name     string
		path     string
		headers  map[string]string
		body     []byte
		expected ProtocolType
	}{
		{
			name:     "HTTP GET",
			path:     "/api/users",
			headers:  map[string]string{"Content-Type": "application/json"},
			body:     nil,
			expected: ProtocolHTTP,
		},
		{
			name:     "WebSocket upgrade",
			path:     "/ws",
			headers:  map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"},
			body:     nil,
			expected: ProtocolWebSocket,
		},
		{
			name:     "gRPC call",
			path:     "/api.Service/Method",
			headers:  map[string]string{"Content-Type": "application/grpc"},
			body:     nil,
			expected: ProtocolGRPC,
		},
		{
			name:     "GraphQL query",
			path:     "/graphql",
			headers:  map[string]string{"Content-Type": "application/json"},
			body:     []byte(`{"query":"{ users { id } }"}`),
			expected: ProtocolGraphQL,
		},
	}

	for _, tc := range testCases {
		req := NewBaseRequest("POST", tc.path, tc.headers, tc.body, context.Background())
		protocol := detector.DetectProtocol(req)

		if protocol != tc.expected {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, protocol)
		}
	}
}

// TestProperty_RequestResponseIntegrity tests request/response integrity
func TestProperty_RequestResponseIntegrity(t *testing.T) {
	// Feature: Request/Response Integrity, Property 3: Request data is preserved through response
	detector := NewProtocolDetector()

	// Register handler that preserves request data
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetHeader("X-Method", req.Method())
		resp.SetHeader("X-Path", req.Path())
		resp.SetBody(req.Body())
		return resp, nil
	})

	_ = detector.RegisterHandler(ProtocolHTTP, handler)

	// Test with various request data
	testCases := []struct {
		method string
		path   string
		body   []byte
	}{
		{"GET", "/api/users", nil},
		{"POST", "/api/users", []byte(`{"name":"Alice"}`)},
		{"PUT", "/api/users/1", []byte(`{"name":"Bob"}`)},
		{"DELETE", "/api/users/1", nil},
	}

	for _, tc := range testCases {
		req := NewBaseRequest(tc.method, tc.path, map[string]string{}, tc.body, context.Background())
		resp, err := detector.Route(req)

		if err != nil {
			t.Errorf("routing failed: %v", err)
		}

		if resp.Header("X-Method") != tc.method {
			t.Errorf("method not preserved: expected %s, got %s", tc.method, resp.Header("X-Method"))
		}

		if resp.Header("X-Path") != tc.path {
			t.Errorf("path not preserved: expected %s, got %s", tc.path, resp.Header("X-Path"))
		}

		if string(resp.Body()) != string(tc.body) {
			t.Errorf("body not preserved: expected %s, got %s", string(tc.body), string(resp.Body()))
		}
	}
}

// TestProperty_MultiProtocolConsistency tests consistency across protocols
func TestProperty_MultiProtocolConsistency(t *testing.T) {
	// Feature: Multi-Protocol Consistency, Property 4: All protocols handle requests consistently
	detector := NewProtocolDetector()

	// Register identical handlers for all protocols
	for i := 0; i < 4; i++ {
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetHeader("Content-Type", "application/json")
			resp.SetBody([]byte(`{"status":"ok"}`))
			return resp, nil
		})
		_ = detector.RegisterHandler(ProtocolType(i), handler)
	}

	// Send same request through different protocols
	protocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL}
	responses := make([]Response, len(protocols))

	for i, protocol := range protocols {
		var req Request
		switch protocol {
		case ProtocolHTTP:
			req = NewBaseRequest("GET", "/api", map[string]string{}, nil, context.Background())
		case ProtocolWebSocket:
			req = NewBaseRequest("GET", "/ws", map[string]string{"Upgrade": "websocket"}, nil, context.Background())
		case ProtocolGRPC:
			req = NewBaseRequest("POST", "/rpc", map[string]string{"Content-Type": "application/grpc"}, nil, context.Background())
		case ProtocolGraphQL:
			req = NewBaseRequest("POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`), context.Background())
		}

		resp, err := detector.Route(req)
		if err != nil {
			t.Errorf("routing failed for protocol %v: %v", protocol, err)
		}

		responses[i] = resp
	}

	// Verify all responses are consistent
	for i, resp := range responses {
		if resp.Status() != 200 {
			t.Errorf("protocol %v: expected status 200, got %d", protocols[i], resp.Status())
		}

		if string(resp.Body()) != `{"status":"ok"}` {
			t.Errorf("protocol %v: expected body {\"status\":\"ok\"}, got %s", protocols[i], string(resp.Body()))
		}
	}
}

// TestProperty_ProtocolIndependence tests protocol independence
func TestProperty_ProtocolIndependence(t *testing.T) {
	// Feature: Protocol Independence, Property 5: Protocols operate independently
	detector := NewProtocolDetector()

	// Register different handlers for each protocol
	handlers := map[ProtocolType]string{
		ProtocolHTTP:      "HTTP",
		ProtocolWebSocket: "WebSocket",
		ProtocolGRPC:      "gRPC",
		ProtocolGraphQL:   "GraphQL",
	}

	for protocol, name := range handlers {
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetBody([]byte(name))
			return resp, nil
		})
		if err := detector.RegisterHandler(protocol, handler); err != nil {
			t.Fatalf("Failed to register handler for protocol %v: %v", protocol, err)
		}
	}

	// Verify each protocol returns its own response
	testCases := []struct {
		path     string
		headers  map[string]string
		body     []byte
		expected string
	}{
		{"/api", map[string]string{}, nil, "HTTP"},
		{"/ws", map[string]string{"Upgrade": "websocket"}, nil, "WebSocket"},
		{"/rpc", map[string]string{"Content-Type": "application/grpc"}, nil, "gRPC"},
		{"/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`), "GraphQL"},
	}

	for _, tc := range testCases {
		req := NewBaseRequest("POST", tc.path, tc.headers, tc.body, context.Background())
		resp, err := detector.Route(req)

		if err != nil {
			t.Errorf("routing failed: %v", err)
		}

		if string(resp.Body()) != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, string(resp.Body()))
		}
	}
}

// TestProperty_ErrorHandlingConsistency tests error handling consistency
func TestProperty_ErrorHandlingConsistency(t *testing.T) {
	// Feature: Error Handling, Property 6: Error handling is consistent across protocols
	detector := NewProtocolDetector()

	// Register handlers that return errors
	for i := 0; i < 4; i++ {
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(500)
			resp.SetBody([]byte("Internal Server Error"))
			return resp, nil
		})
		_ = detector.RegisterHandler(ProtocolType(i), handler)
	}

	// Test error handling for each protocol
	testCases := []struct {
		path    string
		headers map[string]string
		body    []byte
	}{
		{"/api", map[string]string{}, nil},
		{"/ws", map[string]string{"Upgrade": "websocket"}, nil},
		{"/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
		{"/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
	}

	for _, tc := range testCases {
		req := NewBaseRequest("POST", tc.path, tc.headers, tc.body, context.Background())
		resp, err := detector.Route(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if resp.Status() != 500 {
			t.Errorf("expected status 500, got %d", resp.Status())
		}

		if string(resp.Body()) != "Internal Server Error" {
			t.Errorf("expected error message, got %s", string(resp.Body()))
		}
	}
}

// TestProperty_HeaderPreservation tests header preservation across protocols
func TestProperty_HeaderPreservation(t *testing.T) {
	// Feature: Header Handling, Property 7: Headers are preserved through routing
	detector := NewProtocolDetector()

	// Register handler that echoes headers
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)

		for key, value := range req.Headers() {
			resp.SetHeader("X-Echo-"+key, value)
		}

		return resp, nil
	})

	_ = detector.RegisterHandler(ProtocolHTTP, handler)

	// Create request with multiple headers
	headers := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
		"X-Custom":      "value",
	}

	req := NewBaseRequest("GET", "/api", headers, nil, context.Background())
	resp, err := detector.Route(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify headers are echoed back
	for key, value := range headers {
		echoKey := "X-Echo-" + key
		if resp.Header(echoKey) != value {
			t.Errorf("header %s not preserved: expected %s, got %s", key, value, resp.Header(echoKey))
		}
	}
}

// TestProperty_ProtocolDetectionStability tests detection stability
func TestProperty_ProtocolDetectionStability(t *testing.T) {
	// Feature: Protocol Detection, Property 8: Protocol detection is stable and repeatable
	detector := NewProtocolDetector()

	testCases := []struct {
		path    string
		headers map[string]string
		body    []byte
	}{
		{"/api", map[string]string{}, nil},
		{"/ws", map[string]string{"Upgrade": "websocket"}, nil},
		{"/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
		{"/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
	}

	for _, tc := range testCases {
		req := NewBaseRequest("POST", tc.path, tc.headers, tc.body, context.Background())

		// Detect multiple times
		protocols := make([]ProtocolType, 5)
		for i := 0; i < 5; i++ {
			protocols[i] = detector.DetectProtocol(req)
		}

		// All detections should be the same
		for i := 1; i < len(protocols); i++ {
			if protocols[i] != protocols[0] {
				t.Errorf("detection not stable: %v != %v", protocols[0], protocols[i])
			}
		}
	}
}

// TestProperty_ConcurrentMultiProtocolRouting tests concurrent routing across protocols
func TestProperty_ConcurrentMultiProtocolRouting(t *testing.T) {
	// Feature: Concurrency, Property 9: Concurrent multi-protocol routing is safe
	detector := NewProtocolDetector()

	// Register handlers for all protocols
	for i := 0; i < 4; i++ {
		handler := HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			return resp, nil
		})
		if err := detector.RegisterHandler(ProtocolType(i), handler); err != nil {
			t.Logf("failed to register handler for protocol %d: %v", i, err)
		}
	}

	// Send concurrent requests across all protocols
	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func(idx int) {
			protocol := ProtocolType(idx % 4)

			var req Request
			switch protocol {
			case ProtocolHTTP:
				req = NewBaseRequest("GET", "/api", map[string]string{}, nil, context.Background())
			case ProtocolWebSocket:
				req = NewBaseRequest("GET", "/ws", map[string]string{"Upgrade": "websocket"}, nil, context.Background())
			case ProtocolGRPC:
				req = NewBaseRequest("POST", "/rpc", map[string]string{"Content-Type": "application/grpc"}, nil, context.Background())
			case ProtocolGraphQL:
				req = NewBaseRequest("POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`), context.Background())
			}

			resp, err := detector.Route(req)
			if err != nil {
				t.Errorf("routing failed: %v", err)
			}

			if resp.Status() != 200 {
				t.Errorf("expected status 200, got %d", resp.Status())
			}

			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestProperty_ProtocolMetricsAccuracy tests metrics accuracy
func TestProperty_ProtocolMetricsAccuracy(t *testing.T) {
	// Feature: Metrics, Property 10: Protocol metrics are accurate
	detector := NewProtocolDetector()

	// Register handlers
	for i := 0; i < 4; i++ {
		if err := detector.RegisterHandler(ProtocolType(i), &MockHandler{}); err != nil {
			t.Logf("failed to register handler for protocol %d: %v", i, err)
		}
	}

	// Get metrics
	metrics := detector.GetMetrics()

	// Verify metrics
	count, ok := metrics["protocol_count"]
	if !ok {
		t.Fatal("protocol_count not in metrics")
	}

	if count != 4 {
		t.Errorf("expected 4 protocols, got %v", count)
	}

	protocols, ok := metrics["supported_protocols"]
	if !ok {
		t.Fatal("supported_protocols not in metrics")
	}

	if protocols == nil {
		t.Fatal("supported_protocols is nil")
	}
}
