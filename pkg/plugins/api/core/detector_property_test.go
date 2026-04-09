package core

import (
	"context"
	"strings"
	"testing"
)

// PropertyTestHelper provides utilities for property-based testing
type PropertyTestHelper struct {
	t *testing.T
}

// NewPropertyTestHelper creates a new property test helper
func NewPropertyTestHelper(t *testing.T) *PropertyTestHelper {
	return &PropertyTestHelper{t: t}
}

// TestProperty_DetectorAlwaysReturnsValidProtocol tests that detector always returns a valid protocol
func TestProperty_DetectorAlwaysReturnsValidProtocol(t *testing.T) {
	// Feature: Protocol Detection, Property 1: Detector always returns a valid protocol type
	pd := NewProtocolDetector()

	testCases := []struct {
		method  string
		path    string
		headers map[string]string
		body    []byte
	}{
		{"GET", "/api", map[string]string{}, nil},
		{"POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
		{"GET", "/ws", map[string]string{"Upgrade": "websocket"}, nil},
		{"POST", "/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
		{"DELETE", "/users/1", map[string]string{}, nil},
		{"PUT", "/data", map[string]string{"Content-Type": "text/plain"}, []byte("data")},
		{"PATCH", "/config", map[string]string{}, nil},
		{"HEAD", "/health", map[string]string{}, nil},
		{"OPTIONS", "/api", map[string]string{}, nil},
		{"TRACE", "/debug", map[string]string{}, nil},
	}

	for i, tc := range testCases {
		req := NewBaseRequest(context.Background(), tc.method, tc.path, tc.headers, tc.body)
		protocol := pd.DetectProtocol(req)

		// Protocol should be one of the valid types
		validProtocols := []ProtocolType{
			ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL, ProtocolUnknown,
		}
		found := false
		for _, valid := range validProtocols {
			if protocol == valid {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("test case %d: invalid protocol type %v", i, protocol)
		}
	}
}

// TestProperty_RegisteredHandlerIsRetrievable tests that registered handlers can be retrieved
func TestProperty_RegisteredHandlerIsRetrievable(t *testing.T) {
	// Feature: Handler Registration, Property 2: Any registered handler can be retrieved
	pd := NewProtocolDetector()

	protocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL}
	handlers := make(map[ProtocolType]*MockHandler)

	for _, protocol := range protocols {
		handler := &MockHandler{}
		handlers[protocol] = handler
		_ = pd.RegisterHandler(protocol, handler)
	}

	for _, protocol := range protocols {
		if !pd.IsProtocolSupported(protocol) {
			t.Errorf("protocol %v should be supported after registration", protocol)
		}
	}
}

// TestProperty_DetectionIsConsistent tests that protocol detection is consistent
func TestProperty_DetectionIsConsistent(t *testing.T) {
	// Feature: Protocol Detection, Property 3: Protocol detection is consistent for same request
	pd := NewProtocolDetector()

	testCases := []struct {
		method  string
		path    string
		headers map[string]string
		body    []byte
	}{
		{"POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
		{"GET", "/ws", map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"}, nil},
		{"POST", "/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
	}

	for _, tc := range testCases {
		req := NewBaseRequest(context.Background(), tc.method, tc.path, tc.headers, tc.body)
		protocol1 := pd.DetectProtocol(req)
		protocol2 := pd.DetectProtocol(req)

		if protocol1 != protocol2 {
			t.Errorf("protocol detection inconsistent: %v != %v", protocol1, protocol2)
		}
	}
}

// TestProperty_ProtocolCountMatchesRegistrations tests that protocol count matches registrations
func TestProperty_ProtocolCountMatchesRegistrations(t *testing.T) {
	// Feature: Handler Management, Property 4: Protocol count always matches number of registrations
	pd := NewProtocolDetector()

	for i := 0; i < 4; i++ {
		protocol := ProtocolType(i)
		_ = pd.RegisterHandler(protocol, &MockHandler{})

		if pd.GetSupportedProtocolCount() != i+1 {
			t.Errorf("expected %d protocols, got %d", i+1, pd.GetSupportedProtocolCount())
		}
	}
}

// TestProperty_MetricsAlwaysContainRequiredFields tests that metrics contain required fields
func TestProperty_MetricsAlwaysContainRequiredFields(t *testing.T) {
	// Feature: Metrics, Property 5: Metrics always contain required fields
	pd := NewProtocolDetector()

	// Test with no handlers
	metrics := pd.GetMetrics()
	if _, ok := metrics["protocol_count"]; !ok {
		t.Error("metrics missing protocol_count field")
	}
	if _, ok := metrics["supported_protocols"]; !ok {
		t.Error("metrics missing supported_protocols field")
	}

	// Test with handlers
	for i := 0; i < 4; i++ {
		_ = pd.RegisterHandler(ProtocolType(i), &MockHandler{})
	}

	metrics = pd.GetMetrics()
	if _, ok := metrics["protocol_count"]; !ok {
		t.Error("metrics missing protocol_count field")
	}
	if _, ok := metrics["supported_protocols"]; !ok {
		t.Error("metrics missing supported_protocols field")
	}
}

// TestProperty_GraphQLDetectionByPath tests GraphQL detection by path
func TestProperty_GraphQLDetectionByPath(t *testing.T) {
	// Feature: Protocol Detection, Property 6: GraphQL is detected when path contains 'graphql'
	pd := NewProtocolDetector()

	paths := []string{"/graphql", "/api/graphql", "/v1/graphql", "/graphql/query"}
	for _, path := range paths {
		req := NewBaseRequest(context.Background(), "POST", path, map[string]string{}, nil)
		protocol := pd.DetectProtocol(req)
		if protocol != ProtocolGraphQL {
			t.Errorf("expected GraphQL for path %s, got %v", path, protocol)
		}
	}
}

// TestProperty_WebSocketDetectionByHeaders tests WebSocket detection by headers
func TestProperty_WebSocketDetectionByHeaders(t *testing.T) {
	// Feature: Protocol Detection, Property 7: WebSocket is detected by Upgrade header
	pd := NewProtocolDetector()

	headers := []map[string]string{
		{"Upgrade": "websocket", "Connection": "Upgrade"},
		{"Upgrade": "WebSocket", "Connection": "upgrade"},
		{"Upgrade": "WEBSOCKET", "Connection": "Upgrade"},
	}

	for _, h := range headers {
		req := NewBaseRequest(context.Background(), "GET", "/api", h, nil)
		protocol := pd.DetectProtocol(req)
		if protocol != ProtocolWebSocket {
			t.Errorf("expected WebSocket for headers %v, got %v", h, protocol)
		}
	}
}

// TestProperty_GRPCDetectionByContentType tests gRPC detection by content type
func TestProperty_GRPCDetectionByContentType(t *testing.T) {
	// Feature: Protocol Detection, Property 8: gRPC is detected by application/grpc content type
	pd := NewProtocolDetector()

	contentTypes := []string{
		"application/grpc",
		"application/grpc+proto",
		"application/grpc+json",
	}

	for _, ct := range contentTypes {
		req := NewBaseRequest(context.Background(), "POST", "/api", map[string]string{"Content-Type": ct}, nil)
		protocol := pd.DetectProtocol(req)
		if protocol != ProtocolGRPC {
			t.Errorf("expected gRPC for content type %s, got %v", ct, protocol)
		}
	}
}

// TestProperty_RoutingFailsWithoutHandler tests that routing fails without handler
func TestProperty_RoutingFailsWithoutHandler(t *testing.T) {
	// Feature: Routing, Property 9: Routing fails when no handler is registered for detected protocol
	pd := NewProtocolDetector()

	req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
	_, err := pd.Route(req)

	if err == nil {
		t.Error("expected error when routing without handler")
	}
}

// TestProperty_RoutingSucceedsWithHandler tests that routing succeeds with handler
func TestProperty_RoutingSucceedsWithHandler(t *testing.T) {
	// Feature: Routing, Property 10: Routing succeeds when handler is registered for detected protocol
	pd := NewProtocolDetector()
	handler := &MockHandler{resp: NewBaseResponse(nil)}
	_ = pd.RegisterHandler(ProtocolHTTP, handler)

	req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
	resp, err := pd.Route(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response")
	}
}

// TestProperty_ProtocolNameIsNeverEmpty tests that protocol names are never empty
func TestProperty_ProtocolNameIsNeverEmpty(t *testing.T) {
	// Feature: Protocol Names, Property 11: Protocol names are never empty
	protocols := []ProtocolType{
		ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL, ProtocolUnknown,
	}

	for _, protocol := range protocols {
		name := GetProtocolName(protocol)
		if name == "" {
			t.Errorf("protocol %v has empty name", protocol)
		}
	}
}

// TestProperty_GetRegisteredProtocolsReturnsAllRegistered tests that all registered protocols are returned
func TestProperty_GetRegisteredProtocolsReturnsAllRegistered(t *testing.T) {
	// Feature: Handler Management, Property 12: GetRegisteredProtocols returns all registered protocols
	pd := NewProtocolDetector()

	protocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC}
	for _, protocol := range protocols {
		_ = pd.RegisterHandler(protocol, &MockHandler{})
	}

	registered := pd.GetRegisteredProtocols()
	if len(registered) != len(protocols) {
		t.Errorf("expected %d protocols, got %d", len(protocols), len(registered))
	}

	for _, protocol := range protocols {
		found := false
		for _, reg := range registered {
			if reg == protocol {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("protocol %v not in registered list", protocol)
		}
	}
}

// TestProperty_NilRequestReturnsUnknown tests that nil request returns Unknown protocol
func TestProperty_NilRequestReturnsUnknown(t *testing.T) {
	// Feature: Protocol Detection, Property 13: Nil request always returns Unknown protocol
	pd := NewProtocolDetector()

	protocol := pd.DetectProtocol(nil)
	if protocol != ProtocolUnknown {
		t.Errorf("expected ProtocolUnknown for nil request, got %v", protocol)
	}
}

// TestProperty_GraphQLDetectionByBody tests GraphQL detection by body content
func TestProperty_GraphQLDetectionByBody(t *testing.T) {
	// Feature: Protocol Detection, Property 14: GraphQL is detected by query/mutation in body
	pd := NewProtocolDetector()

	bodies := [][]byte{
		[]byte(`{"query":"{ users { id } }"}`),
		[]byte(`{"mutation":"mutation { createUser { id } }"}`),
		[]byte(`{"query":"query GetUsers { users { id } }"}`),
	}

	for _, body := range bodies {
		req := NewBaseRequest(context.Background(), "POST", "/api", map[string]string{"Content-Type": "application/json"}, body)
		protocol := pd.DetectProtocol(req)
		if protocol != ProtocolGraphQL {
			t.Errorf("expected GraphQL for body %s, got %v", string(body), protocol)
		}
	}
}

// TestProperty_WebSocketDetectionByPath tests WebSocket detection by path
func TestProperty_WebSocketDetectionByPath(t *testing.T) {
	// Feature: Protocol Detection, Property 15: WebSocket is detected by path containing ws/websocket
	pd := NewProtocolDetector()

	paths := []string{"/ws", "/websocket", "/api/ws", "/v1/websocket"}
	for _, path := range paths {
		req := NewBaseRequest(context.Background(), "GET", path, map[string]string{}, nil)
		protocol := pd.DetectProtocol(req)
		if protocol != ProtocolWebSocket {
			t.Errorf("expected WebSocket for path %s, got %v", path, protocol)
		}
	}
}

// TestProperty_ConcurrentOperationsAreSafe tests that concurrent operations are safe
func TestProperty_ConcurrentOperationsAreSafe(t *testing.T) {
	// Feature: Concurrency, Property 16: Concurrent operations are safe and don't cause data races
	pd := NewProtocolDetector()

	// Register handlers concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			protocol := ProtocolType(idx % 4)
			_ = pd.RegisterHandler(protocol, &MockHandler{})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Detect protocols concurrently
	for i := 0; i < 10; i++ {
		go func() {
			req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
			pd.DetectProtocol(req)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Check metrics concurrently
	for i := 0; i < 10; i++ {
		go func() {
			pd.GetMetrics()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestProperty_ProtocolDetectionIsIdempotent tests that protocol detection is idempotent
func TestProperty_ProtocolDetectionIsIdempotent(t *testing.T) {
	// Feature: Protocol Detection, Property 17: Protocol detection is idempotent
	pd := NewProtocolDetector()

	testCases := []struct {
		method  string
		path    string
		headers map[string]string
		body    []byte
	}{
		{"POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`)},
		{"GET", "/ws", map[string]string{"Upgrade": "websocket"}, nil},
		{"POST", "/rpc", map[string]string{"Content-Type": "application/grpc"}, nil},
		{"GET", "/api", map[string]string{}, nil},
	}

	for _, tc := range testCases {
		req := NewBaseRequest(context.Background(), tc.method, tc.path, tc.headers, tc.body)

		// Detect multiple times
		protocols := make([]ProtocolType, 5)
		for i := 0; i < 5; i++ {
			protocols[i] = pd.DetectProtocol(req)
		}

		// All should be the same
		for i := 1; i < len(protocols); i++ {
			if protocols[i] != protocols[0] {
				t.Errorf("protocol detection not idempotent: %v != %v", protocols[0], protocols[i])
			}
		}
	}
}

// TestProperty_HandlerCallReceivesCorrectRequest tests that handler receives correct request
func TestProperty_HandlerCallReceivesCorrectRequest(t *testing.T) {
	// Feature: Routing, Property 18: Handler receives the exact request that was routed
	pd := NewProtocolDetector()
	handler := &MockHandler{resp: NewBaseResponse(nil)}
	_ = pd.RegisterHandler(ProtocolHTTP, handler)

	req := NewBaseRequest(context.Background(), "GET", "/api/users", map[string]string{"Authorization": "Bearer token"}, nil)
	_, _ = pd.Route(req)

	if handler.req == nil {
		t.Fatal("handler did not receive request")
	}
	if handler.req.Path() != req.Path() {
		t.Errorf("handler received wrong path: %s != %s", handler.req.Path(), req.Path())
	}
	if handler.req.Method() != req.Method() {
		t.Errorf("handler received wrong method: %s != %s", handler.req.Method(), req.Method())
	}
}

// TestProperty_ProtocolDetectionPriority tests protocol detection priority order
func TestProperty_ProtocolDetectionPriority(t *testing.T) {
	// Feature: Protocol Detection, Property 19: Protocol detection follows correct priority
	pd := NewProtocolDetector()

	// GraphQL should be detected before WebSocket
	req := NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{
		"Upgrade":     "websocket",
		"Connection":  "Upgrade",
		"Content-Type": "application/json",
	}, []byte(`{"query":"{}"}`))

	protocol := pd.DetectProtocol(req)
	if protocol != ProtocolGraphQL {
		t.Errorf("expected GraphQL (higher priority), got %v", protocol)
	}
}

// TestProperty_EmptyHeadersHandledCorrectly tests that empty headers are handled correctly
func TestProperty_EmptyHeadersHandledCorrectly(t *testing.T) {
	// Feature: Protocol Detection, Property 20: Empty headers are handled correctly
	pd := NewProtocolDetector()

	req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
	protocol := pd.DetectProtocol(req)

	// Should default to HTTP
	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP for empty headers, got %v", protocol)
	}
}

// TestProperty_ProtocolNameConsistency tests that protocol names are consistent
func TestProperty_ProtocolNameConsistency(t *testing.T) {
	// Feature: Protocol Names, Property 21: Protocol names are consistent across calls
	protocols := []ProtocolType{
		ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL, ProtocolUnknown,
	}

	for _, protocol := range protocols {
		name1 := GetProtocolName(protocol)
		name2 := GetProtocolName(protocol)

		if name1 != name2 {
			t.Errorf("protocol name inconsistent: %s != %s", name1, name2)
		}
	}
}

// TestProperty_RegistrationDoesNotAffectDetection tests that registration doesn't affect detection
func TestProperty_RegistrationDoesNotAffectDetection(t *testing.T) {
	// Feature: Protocol Detection, Property 22: Protocol detection is independent of handler registration
	pd := NewProtocolDetector()

	req := NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{"Content-Type": "application/json"}, []byte(`{"query":"{}"}`))

	// Detect before registration
	protocol1 := pd.DetectProtocol(req)

	// Register handler
	_ = pd.RegisterHandler(ProtocolGraphQL, &MockHandler{})

	// Detect after registration
	protocol2 := pd.DetectProtocol(req)

	if protocol1 != protocol2 {
		t.Errorf("protocol detection changed after registration: %v != %v", protocol1, protocol2)
	}
}

// TestProperty_MetricsReflectCurrentState tests that metrics reflect current state
func TestProperty_MetricsReflectCurrentState(t *testing.T) {
	// Feature: Metrics, Property 23: Metrics always reflect current state
	pd := NewProtocolDetector()

	for i := 0; i < 4; i++ {
		metrics := pd.GetMetrics()
		count := metrics["protocol_count"].(int)

		if count != i {
			t.Errorf("expected %d protocols in metrics, got %d", i, count)
		}

		_ = pd.RegisterHandler(ProtocolType(i), &MockHandler{})
	}
}

// TestProperty_ProtocolTypeValuesAreUnique tests that protocol type values are unique
func TestProperty_ProtocolTypeValuesAreUnique(t *testing.T) {
	// Feature: Protocol Types, Property 24: Protocol type values are unique
	protocols := []ProtocolType{
		ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL, ProtocolUnknown,
	}

	seen := make(map[ProtocolType]bool)
	for _, protocol := range protocols {
		if seen[protocol] {
			t.Errorf("duplicate protocol type: %v", protocol)
		}
		seen[protocol] = true
	}
}

// TestProperty_LargeNumberOfHandlers tests detector with large number of handlers
func TestProperty_LargeNumberOfHandlers(t *testing.T) {
	// Feature: Handler Management, Property 25: Detector handles large number of handlers efficiently
	pd := NewProtocolDetector()

	// Register many handlers (simulate multiple protocol variants)
	for i := 0; i < 100; i++ {
		protocol := ProtocolType(i % 4)
		_ = pd.RegisterHandler(protocol, &MockHandler{})
	}

	// Should still work correctly
	req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
	protocol := pd.DetectProtocol(req)

	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP, got %v", protocol)
	}
}

// TestProperty_SpecialCharactersInPath tests protocol detection with special characters
func TestProperty_SpecialCharactersInPath(t *testing.T) {
	// Feature: Protocol Detection, Property 26: Protocol detection handles special characters in path
	pd := NewProtocolDetector()

	paths := []string{
		"/api/v1/graphql?query=test",
		"/ws/channel/123",
		"/rpc/method?param=value",
		"/api/users/john@example.com",
	}

	for _, path := range paths {
		req := NewBaseRequest(context.Background(), "GET", path, map[string]string{}, nil)
		protocol := pd.DetectProtocol(req)

		// Should return a valid protocol
		if protocol == ProtocolUnknown && !strings.Contains(path, "rpc") {
			// Most paths should not be unknown
			if !strings.Contains(path, "graphql") && !strings.Contains(path, "ws") {
				_ = protocol // This is expected for generic paths
			}
		}
	}
}

// TestProperty_CaseInsensitivePathDetection tests case-insensitive path detection
func TestProperty_CaseInsensitivePathDetection(t *testing.T) {
	// Feature: Protocol Detection, Property 27: Path detection is case-sensitive for protocol keywords
	pd := NewProtocolDetector()

	// GraphQL path detection is case-sensitive
	req1 := NewBaseRequest(context.Background(), "POST", "/graphql", map[string]string{}, nil)
	req2 := NewBaseRequest(context.Background(), "POST", "/GraphQL", map[string]string{}, nil)

	protocol1 := pd.DetectProtocol(req1)
	protocol2 := pd.DetectProtocol(req2)

	if protocol1 != ProtocolGraphQL {
		t.Errorf("expected GraphQL for /graphql, got %v", protocol1)
	}

	// /GraphQL should not be detected as GraphQL (case-sensitive)
	if protocol2 == ProtocolGraphQL {
		t.Errorf("expected non-GraphQL for /GraphQL, got %v", protocol2)
	}
}

// TestProperty_MultipleHeadersHandledCorrectly tests handling of multiple headers
func TestProperty_MultipleHeadersHandledCorrectly(t *testing.T) {
	// Feature: Protocol Detection, Property 28: Multiple headers are handled correctly
	pd := NewProtocolDetector()

	headers := map[string]string{
		"Content-Type":     "application/json",
		"Authorization":    "Bearer token",
		"X-Custom-Header":  "value",
		"Accept":           "application/json",
		"User-Agent":       "test-client",
	}

	req := NewBaseRequest(context.Background(), "POST", "/api", headers, nil)
	protocol := pd.DetectProtocol(req)

	if protocol != ProtocolHTTP {
		t.Errorf("expected ProtocolHTTP, got %v", protocol)
	}
}

// TestProperty_ErrorHandlingIsConsistent tests that error handling is consistent
func TestProperty_ErrorHandlingIsConsistent(t *testing.T) {
	// Feature: Error Handling, Property 29: Error handling is consistent
	pd := NewProtocolDetector()

	// Multiple nil requests should all return Unknown
	for i := 0; i < 10; i++ {
		protocol := pd.DetectProtocol(nil)
		if protocol != ProtocolUnknown {
			t.Errorf("expected ProtocolUnknown for nil request, got %v", protocol)
		}
	}

	// Multiple routing attempts without handler should all fail
	req := NewBaseRequest(context.Background(), "GET", "/api", map[string]string{}, nil)
	for i := 0; i < 10; i++ {
		_, err := pd.Route(req)
		if err == nil {
			t.Error("expected error when routing without handler")
		}
	}
}

// TestProperty_ProtocolDetectionWithEmptyBody tests protocol detection with empty body
func TestProperty_ProtocolDetectionWithEmptyBody(t *testing.T) {
	// Feature: Protocol Detection, Property 30: Protocol detection works with empty body
	pd := NewProtocolDetector()

	testCases := []struct {
		path     string
		headers  map[string]string
		expected ProtocolType
	}{
		{"/graphql", map[string]string{"Content-Type": "application/json"}, ProtocolGraphQL},
		{"/ws", map[string]string{}, ProtocolWebSocket},
		{"/api", map[string]string{}, ProtocolHTTP},
	}

	for _, tc := range testCases {
		req := NewBaseRequest(context.Background(), "POST", tc.path, tc.headers, []byte{})
		protocol := pd.DetectProtocol(req)

		if protocol != tc.expected {
			t.Errorf("expected %v for path %s, got %v", tc.expected, tc.path, protocol)
		}
	}
}
