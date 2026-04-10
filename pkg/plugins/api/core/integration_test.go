package core

import (
	"context"
	"testing"
)

// TestIntegration_HTTPRequestFlow tests complete HTTP request flow
func TestIntegration_HTTPRequestFlow(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register HTTP handler
	httpHandler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte(`{"status":"ok"}`))
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolHTTP, httpHandler); err != nil {
		t.Fatalf("Failed to register HTTP handler: %v", err)
	}

	// Create HTTP request
	req := NewBaseRequest(context.Background(), "GET", "/api/users", map[string]string{
		"Content-Type": "application/json",
	}, nil)

	// Detect protocol
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP, got %v", protocol)
	}

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if string(resp.Body()) != `{"status":"ok"}` {
		t.Errorf("unexpected response body: %s", string(resp.Body()))
	}
}

// TestIntegration_WebSocketRequestFlow tests complete WebSocket request flow
func TestIntegration_WebSocketRequestFlow(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register WebSocket handler
	wsHandler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(101)
		resp.SetHeader("Upgrade", "websocket")
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolWebSocket, wsHandler); err != nil {
		t.Fatalf("failed to register WebSocket handler: %v", err)
	}

	// Create WebSocket request
	req := NewBaseRequest(context.Background(), "GET", "/ws", map[string]string{
		"Upgrade":    "websocket",
		"Connection": "Upgrade",
	}, nil)

	// Detect protocol
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolWebSocket {
		t.Errorf("expected ProtocolWebSocket, got %v", protocol)
	}

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status() != 101 {
		t.Errorf("expected status 101, got %d", resp.Status())
	}
}

// TestIntegration_GRPCRequestFlow tests complete gRPC request flow
func TestIntegration_GRPCRequestFlow(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register gRPC handler
	grpcHandler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetHeader("Content-Type", "application/grpc")
		resp.SetBody([]byte{0, 0, 0, 0, 5, 123, 34, 125})
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolGRPC, grpcHandler); err != nil {
		t.Fatalf("failed to register gRPC handler: %v", err)
	}

	// Create gRPC request
	req := NewBaseRequest(context.Background(), "POST", "/api.Service/Method", map[string]string{
		"Content-Type":  "application/grpc",
		"grpc-encoding": "gzip",
	}, []byte{0, 0, 0, 0, 2, 123, 125})

	// Detect protocol
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolGRPC {
		t.Errorf("expected ProtocolGRPC, got %v", protocol)
	}

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

// TestIntegration_GraphQLRequestFlow tests complete GraphQL request flow
func TestIntegration_GraphQLRequestFlow(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register GraphQL handler
	graphqlHandler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetHeader("Content-Type", "application/json")
		resp.SetBody([]byte(`{"data":{"users":[]}}`))
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolGraphQL, graphqlHandler); err != nil {
		t.Fatalf("failed to register GraphQL handler: %v", err)
	}

	// Create GraphQL request
	req := NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{
		"Content-Type": "application/json",
	}, []byte(`{"query":"{ users { id } }"}`))

	// Detect protocol
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected ProtocolGraphQL, got %v", protocol)
	}

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if string(resp.Body()) != `{"data":{"users":[]}}` {
		t.Errorf("unexpected response body: %s", string(resp.Body()))
	}
}

// TestIntegration_MultiProtocolDetection tests detecting multiple protocols
func TestIntegration_MultiProtocolDetection(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register all protocol handlers
	protocols := map[ProtocolType]HandlerFunc{
		ProtocolHTTP: func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetBody([]byte("HTTP"))
			return resp, nil
		},
		ProtocolWebSocket: func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(101)
			resp.SetBody([]byte("WebSocket"))
			return resp, nil
		},
		ProtocolGRPC: func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetBody([]byte("gRPC"))
			return resp, nil
		},
		ProtocolGraphQL: func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			resp.SetBody([]byte("GraphQL"))
			return resp, nil
		},
	}

	for protocol, handler := range protocols {
		if err := detector.RegisterHandler(protocol, handler); err != nil {
			t.Fatalf("failed to register handler for protocol %v: %v", protocol, err)
		}
	}

	// Test cases
	testCases := []struct {
		name     string
		req      Request
		expected ProtocolType
		body     string
	}{
		{
			name: "HTTP",
			req: NewBaseRequest(context.Background(), "GET", "/api", map[string]string{
				"Content-Type": "application/json",
			}, nil),
			expected: ProtocolHTTP,
			body:     "HTTP",
		},
		{
			name: "WebSocket",
			req: NewBaseRequest(context.Background(), "GET", "/ws", map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			}, nil),
			expected: ProtocolWebSocket,
			body:     "WebSocket",
		},
		{
			name: "gRPC",
			req: NewBaseRequest(context.Background(), "POST", "/api.Service/Method", map[string]string{
				"Content-Type": "application/grpc",
			}, nil),
			expected: ProtocolGRPC,
			body:     "gRPC",
		},
		{
			name: "GraphQL",
			req: NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{
				"Content-Type": "application/json",
			}, []byte(`{"query":"{}"}`)),
			expected: ProtocolGraphQL,
			body:     "GraphQL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			protocol := detector.DetectProtocol(tc.req)
			if protocol != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, protocol)
			}

			resp, err := detector.Route(tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(resp.Body()) != tc.body {
				t.Errorf("expected body %s, got %s", tc.body, string(resp.Body()))
			}
		})
	}
}

// TestIntegration_RequestResponseConsistency tests request/response consistency
func TestIntegration_RequestResponseConsistency(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register handler that echoes request info
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetHeader("X-Method", req.Method())
		resp.SetHeader("X-Path", req.Path())
		resp.SetBody(req.Body())
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolHTTP, handler); err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}

	// Create request
	body := []byte(`{"test":"data"}`)
	req := NewBaseRequest(context.Background(), "POST", "/api/test", map[string]string{
		"Content-Type": "application/json",
	}, body)

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.Header("X-Method") != "POST" {
		t.Errorf("expected method POST, got %s", resp.Header("X-Method"))
	}

	if resp.Header("X-Path") != "/api/test" {
		t.Errorf("expected path /api/test, got %s", resp.Header("X-Path"))
	}

	if string(resp.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(resp.Body()))
	}
}

// TestIntegration_ErrorHandling tests error handling across protocols
func TestIntegration_ErrorHandling(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register handler that returns error
	handler := HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(500)
		resp.SetBody([]byte("Internal Server Error"))
		return resp, nil
	})

	if err := detector.RegisterHandler(ProtocolHTTP, handler); err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}

	// Create request
	req := NewBaseRequest(context.Background(), "GET", "/api/error", map[string]string{}, nil)

	// Route request
	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status() != 500 {
		t.Errorf("expected status 500, got %d", resp.Status())
	}
}

// TestIntegration_ProtocolPriority tests protocol detection priority
func TestIntegration_ProtocolPriority(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register all handlers
	for i := 0; i < 4; i++ {
		if err := detector.RegisterHandler(ProtocolType(i), &MockHandler{}); err != nil {
			t.Fatalf("failed to register handler: %v", err)
		}
	}

	// Create request that could match multiple protocols
	req := NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{
		"Upgrade":      "websocket",
		"Connection":   "Upgrade",
		"Content-Type": "application/json",
	}, []byte(`{"query":"{}"}`))

	// GraphQL should be detected first (highest priority)
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected ProtocolGraphQL (highest priority), got %v", protocol)
	}
}

// TestIntegration_ConcurrentRequests tests handling concurrent requests
func TestIntegration_ConcurrentRequests(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()

	// Register handlers
	for i := 0; i < 4; i++ {
		if err := detector.RegisterHandler(ProtocolType(i), HandlerFunc(func(req Request) (Response, error) {
			resp := NewBaseResponse(nil)
			resp.SetStatus(200)
			return resp, nil
		})); err != nil {
			t.Fatalf("failed to register handler: %v", err)
		}
	}

	// Send concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
			resp, err := detector.Route(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if resp.Status() != 200 {
				t.Errorf("expected status 200, got %d", resp.Status())
			}

			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestIntegration_FullRequestLifecycle tests complete request lifecycle
func TestIntegration_FullRequestLifecycle(t *testing.T) {
	// Setup
	detector := NewProtocolDetector()
	apiLayer := NewAPILayer()

	// Register business logic handler
	apiLayer.RegisterHandlerFunc("/api/users", func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetHeader("Content-Type", "application/json")
		resp.SetBody([]byte(`[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`))
		return resp, nil
	})

	// Register protocol handler
	protocolHandler := HandlerFunc(func(req Request) (Response, error) {
		return apiLayer.Handle(req), nil
	})

	if err := detector.RegisterHandler(ProtocolHTTP, protocolHandler); err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}

	// Create request
	req := NewBaseRequest(context.Background(), "GET", "/api/users", map[string]string{
		"Content-Type": "application/json",
	}, nil)

	// Detect and route
	protocol := detector.DetectProtocol(req)
	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP, got %v", protocol)
	}

	resp, err := detector.Route(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	expectedBody := `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`
	if string(resp.Body()) != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, string(resp.Body()))
	}
}
