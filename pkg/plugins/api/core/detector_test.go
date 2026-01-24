package core

import (
	"context"
	"testing"
)

// MockHandler is a mock implementation of Handler for testing
type MockHandler struct {
	called bool
	req    Request
	resp   Response
	err    error
}

func (m *MockHandler) Handle(req Request) (Response, error) {
	m.called = true
	m.req = req
	return m.resp, m.err
}

// TestNewProtocolDetector tests creating a new protocol detector
func TestNewProtocolDetector(t *testing.T) {
	pd := NewProtocolDetector()
	if pd == nil {
		t.Fatal("expected non-nil protocol detector")
	}
	if len(pd.handlers) != 0 {
		t.Errorf("expected empty handlers, got %d", len(pd.handlers))
	}
}

// TestRegisterHandler tests registering a handler
func TestRegisterHandler(t *testing.T) {
	pd := NewProtocolDetector()
	handler := &MockHandler{}

	err := pd.RegisterHandler(ProtocolHTTP, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !pd.IsProtocolSupported(ProtocolHTTP) {
		t.Error("expected HTTP protocol to be supported")
	}
}

// TestRegisterHandlerNil tests registering a nil handler
func TestRegisterHandlerNil(t *testing.T) {
	pd := NewProtocolDetector()

	err := pd.RegisterHandler(ProtocolHTTP, nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

// TestRegisterMultipleHandlers tests registering multiple handlers
func TestRegisterMultipleHandlers(t *testing.T) {
	pd := NewProtocolDetector()
	handlers := map[ProtocolType]*MockHandler{
		ProtocolHTTP:      {},
		ProtocolWebSocket: {},
		ProtocolGRPC:      {},
		ProtocolGraphQL:   {},
	}

	for protocol, handler := range handlers {
		err := pd.RegisterHandler(protocol, handler)
		if err != nil {
			t.Fatalf("unexpected error registering %v: %v", protocol, err)
		}
	}

	if pd.GetSupportedProtocolCount() != 4 {
		t.Errorf("expected 4 supported protocols, got %d", pd.GetSupportedProtocolCount())
	}
}

// TestDetectProtocolHTTP tests detecting HTTP protocol
func TestDetectProtocolHTTP(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("GET", "/api/users", map[string]string{
		"Content-Type": "application/json",
	}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP, got %v", protocol)
	}
}

// TestDetectProtocolWebSocket tests detecting WebSocket protocol
func TestDetectProtocolWebSocket(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("GET", "/ws", map[string]string{
		"Upgrade":     "websocket",
		"Connection":  "Upgrade",
		"Content-Type": "application/json",
	}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolWebSocket {
		t.Errorf("expected ProtocolWebSocket, got %v", protocol)
	}
}

// TestDetectProtocolWebSocketPath tests detecting WebSocket by path
func TestDetectProtocolWebSocketPath(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("GET", "/websocket", map[string]string{}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolWebSocket {
		t.Errorf("expected ProtocolWebSocket, got %v", protocol)
	}
}

// TestDetectProtocolGRPC tests detecting gRPC protocol
func TestDetectProtocolGRPC(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("POST", "/api.Service/Method", map[string]string{
		"Content-Type":   "application/grpc",
		"grpc-encoding":  "gzip",
	}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGRPC {
		t.Errorf("expected ProtocolGRPC, got %v", protocol)
	}
}

// TestDetectProtocolGRPCEncoding tests detecting gRPC by encoding header
func TestDetectProtocolGRPCEncoding(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("POST", "/api.Service/Method", map[string]string{
		"grpc-encoding": "gzip",
	}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGRPC {
		t.Errorf("expected ProtocolGRPC, got %v", protocol)
	}
}

// TestDetectProtocolGraphQL tests detecting GraphQL protocol
func TestDetectProtocolGraphQL(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("POST", "/graphql", map[string]string{
		"Content-Type": "application/json",
	}, []byte(`{"query":"{ users { id } }"}`), context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected ProtocolGraphQL, got %v", protocol)
	}
}

// TestDetectProtocolGraphQLMutation tests detecting GraphQL mutation
func TestDetectProtocolGraphQLMutation(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("POST", "/api", map[string]string{
		"Content-Type": "application/json",
	}, []byte(`{"mutation":"mutation { createUser(name: \"John\") { id } }"}`), context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected ProtocolGraphQL, got %v", protocol)
	}
}

// TestDetectProtocolNilRequest tests detecting protocol with nil request
func TestDetectProtocolNilRequest(t *testing.T) {
	pd := NewProtocolDetector()
	protocol := pd.DetectProtocol(nil)
	if protocol != ProtocolUnknown {
		t.Errorf("expected ProtocolUnknown for nil request, got %v", protocol)
	}
}

// TestRoute tests routing a request to a handler
func TestRoute(t *testing.T) {
	pd := NewProtocolDetector()
	handler := &MockHandler{
		resp: NewBaseResponse(nil),
	}

	err := pd.RegisterHandler(ProtocolHTTP, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := NewBaseRequest("GET", "/api/users", map[string]string{}, nil, context.Background())
	resp, err := pd.Route(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !handler.called {
		t.Error("expected handler to be called")
	}
}

// TestRouteNilRequest tests routing with nil request
func TestRouteNilRequest(t *testing.T) {
	pd := NewProtocolDetector()
	_, err := pd.Route(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// TestRouteNoHandler tests routing when no handler is registered
func TestRouteNoHandler(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("GET", "/api/users", map[string]string{}, nil, context.Background())

	_, err := pd.Route(req)
	if err == nil {
		t.Fatal("expected error when no handler registered")
	}
}

// TestGetProtocolName tests getting protocol names
func TestGetProtocolName(t *testing.T) {
	tests := []struct {
		protocol ProtocolType
		expected string
	}{
		{ProtocolHTTP, "HTTP"},
		{ProtocolWebSocket, "WebSocket"},
		{ProtocolGRPC, "gRPC"},
		{ProtocolGraphQL, "GraphQL"},
		{ProtocolUnknown, "Unknown"},
	}

	for _, tt := range tests {
		name := GetProtocolName(tt.protocol)
		if name != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, name)
		}
	}
}

// TestGetRegisteredProtocols tests getting registered protocols
func TestGetRegisteredProtocols(t *testing.T) {
	pd := NewProtocolDetector()
	protocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC}

	for _, protocol := range protocols {
		_ = pd.RegisterHandler(protocol, &MockHandler{})
	}

	registered := pd.GetRegisteredProtocols()
	if len(registered) != len(protocols) {
		t.Errorf("expected %d protocols, got %d", len(protocols), len(registered))
	}
}

// TestIsProtocolSupported tests checking if protocol is supported
func TestIsProtocolSupported(t *testing.T) {
	pd := NewProtocolDetector()
	if err := pd.RegisterHandler(ProtocolHTTP, &MockHandler{}); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	if !pd.IsProtocolSupported(ProtocolHTTP) {
		t.Error("expected HTTP to be supported")
	}
	if pd.IsProtocolSupported(ProtocolWebSocket) {
		t.Error("expected WebSocket to not be supported")
	}
}

// TestGetSupportedProtocolCount tests getting supported protocol count
func TestGetSupportedProtocolCount(t *testing.T) {
	pd := NewProtocolDetector()
	if pd.GetSupportedProtocolCount() != 0 {
		t.Errorf("expected 0 protocols, got %d", pd.GetSupportedProtocolCount())
	}

	if err := pd.RegisterHandler(ProtocolHTTP, &MockHandler{}); err != nil {
		t.Fatalf("Failed to register HTTP handler: %v", err)
	}
	if pd.GetSupportedProtocolCount() != 1 {
		t.Errorf("expected 1 protocol, got %d", pd.GetSupportedProtocolCount())
	}

	if err := pd.RegisterHandler(ProtocolWebSocket, &MockHandler{}); err != nil {
		t.Fatalf("Failed to register WebSocket handler: %v", err)
	}
	if pd.GetSupportedProtocolCount() != 2 {
		t.Errorf("expected 2 protocols, got %d", pd.GetSupportedProtocolCount())
	}
}

// TestGetMetrics tests getting metrics
func TestGetMetrics(t *testing.T) {
	pd := NewProtocolDetector()
	if err := pd.RegisterHandler(ProtocolHTTP, &MockHandler{}); err != nil {
		t.Fatalf("failed to register HTTP handler: %v", err)
	}
	if err := pd.RegisterHandler(ProtocolWebSocket, &MockHandler{}); err != nil {
		t.Fatalf("failed to register WebSocket handler: %v", err)
	}

	metrics := pd.GetMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	count, ok := metrics["protocol_count"]
	if !ok {
		t.Fatal("expected protocol_count in metrics")
	}
	if count != 2 {
		t.Errorf("expected 2 protocols, got %v", count)
	}

	protocols, ok := metrics["supported_protocols"]
	if !ok {
		t.Fatal("expected supported_protocols in metrics")
	}
	if protocols == nil {
		t.Fatal("expected non-nil supported_protocols")
	}
}

// TestDetectProtocolPriority tests protocol detection priority
func TestDetectProtocolPriority(t *testing.T) {
	// GraphQL should be detected before WebSocket
	pd := NewProtocolDetector()
	req := NewBaseRequest("POST", "/graphql", map[string]string{
		"Content-Type": "application/json",
		"Upgrade":      "websocket",
		"Connection":   "Upgrade",
	}, []byte(`{"query":"{ users { id } }"}`), context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected ProtocolGraphQL, got %v", protocol)
	}
}

// TestDetectProtocolCaseInsensitive tests case-insensitive protocol detection
func TestDetectProtocolCaseInsensitive(t *testing.T) {
	pd := NewProtocolDetector()
	req := NewBaseRequest("GET", "/ws", map[string]string{
		"Upgrade":     "WebSocket",
		"Connection":  "upgrade",
	}, nil, context.Background())

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolWebSocket {
		t.Errorf("expected ProtocolWebSocket, got %v", protocol)
	}
}

// TestConcurrentRegistration tests concurrent handler registration
func TestConcurrentRegistration(t *testing.T) {
	pd := NewProtocolDetector()
	done := make(chan bool, 4)

	for i := 0; i < 4; i++ {
		go func(protocol ProtocolType) {
			if err := pd.RegisterHandler(protocol, &MockHandler{}); err != nil {
				t.Logf("failed to register handler for protocol %v: %v", protocol, err)
			}
			done <- true
		}(ProtocolType(i))
	}

	for i := 0; i < 4; i++ {
		<-done
	}

	if pd.GetSupportedProtocolCount() != 4 {
		t.Errorf("expected 4 protocols, got %d", pd.GetSupportedProtocolCount())
	}
}

// TestConcurrentDetection tests concurrent protocol detection
func TestConcurrentDetection(t *testing.T) {
	pd := NewProtocolDetector()
	if err := pd.RegisterHandler(ProtocolHTTP, &MockHandler{}); err != nil {
		t.Fatalf("failed to register handler: %v", err)
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := NewBaseRequest("GET", "/api", map[string]string{}, nil, context.Background())
			protocol := pd.DetectProtocol(req)
			if protocol != ProtocolHTTP {
				t.Errorf("expected ProtocolHTTP, got %v", protocol)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
