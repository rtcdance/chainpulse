package core

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// testReporter adapts *testing.T to gopter.Reporter
type pbtReporter struct {
	t *testing.T
}

func (r *pbtReporter) ReportTestResult(name string, result *gopter.TestResult) {
	if !result.Passed() {
		r.t.Errorf("Property test failed: %s\n%s", name, result.Error)
	}
}

// Generators for protocol detection inputs

// genHTTPMethod generates arbitrary HTTP methods
var genHTTPMethod = gen.OneConstOf(
	"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT",
)

// genAnyPath generates arbitrary URL paths (unused but kept as potential future test utility)
var _ = gen.Rune().Map(func(r rune) string { return string(r) })

// genSlashPath generates paths starting with /
var genSlashPath = gen.SliceOf(gen.OneConstOf(
	rune('a'), rune('b'), rune('c'), rune('/'), rune('_'), rune('-'), rune('.'),
	rune('0'), rune('1'), rune('2'), rune('?'), rune('='),
)).Map(func(runes []rune) string {
	return "/" + string(runes)
})

// genHeaderMap generates arbitrary header maps
var genHeaderMap = gen.MapOf(
	gen.OneConstOf("Content-Type", "Upgrade", "Connection", "Accept", "Authorization", "X-Custom"),
	gen.OneConstOf(
		"application/json", "application/grpc", "text/plain",
		"websocket", "Upgrade", "keep-alive",
		"application/grpc-web", "application/octet-stream",
	),
)

// genBody generates arbitrary request bodies
var genBody = gen.SliceOf(gen.UInt8()).Map(func(b []byte) []byte {
	if len(b) > 200 {
		return b[:200]
	}
	return b
})

// genGraphQLPath generates paths containing "graphql"
var genGraphQLPath = gen.OneConstOf(
	"/graphql", "/api/graphql", "/v1/graphql", "/query/graphql",
)

// genWSPath generates paths containing websocket indicators
var genWSPath = gen.OneConstOf(
	"/ws", "/websocket", "/api/ws", "/stream/ws",
)

// validProtocols is the set of all valid ProtocolType values
var validProtocols = []ProtocolType{
	ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL, ProtocolUnknown,
}

func isValidProtocol(p ProtocolType) bool {
	for _, v := range validProtocols {
		if p == v {
			return true
		}
	}
	return false
}

// --- Property 1: For ANY request, detection always returns a valid protocol type ---
func TestPBT_DetectorAlwaysReturnsValidProtocol(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 200
	properties := gopter.NewProperties(parameters)

	properties.Property("detection always returns valid protocol", prop.ForAll(
		func(method string, path string, headers map[string]string, body []byte) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), method, path, headers, body)
			return isValidProtocol(pd.DetectProtocol(req))
		},
		genHTTPMethod,
		genSlashPath,
		genHeaderMap,
		genBody,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 2: Detection is deterministic (same request => same result) ---
func TestPBT_DetectionIsDeterministic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 200
	properties := gopter.NewProperties(parameters)

	properties.Property("detection is deterministic for any request", prop.ForAll(
		func(method string, path string, headers map[string]string, body []byte) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), method, path, headers, body)
			first := pd.DetectProtocol(req)
			for i := 0; i < 5; i++ {
				if pd.DetectProtocol(req) != first {
					return false
				}
			}
			return true
		},
		genHTTPMethod,
		genSlashPath,
		genHeaderMap,
		genBody,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 3: GraphQL path always detected as ProtocolGraphQL ---
func TestPBT_GraphQLPathDetectedCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("graphql path is always detected as ProtocolGraphQL", prop.ForAll(
		func(path string) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), "POST", path, map[string]string{}, nil)
			return pd.DetectProtocol(req) == ProtocolGraphQL
		},
		genGraphQLPath,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 4: WebSocket upgrade header always detected ---
func TestPBT_WebSocketHeaderDetectedCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Use paths that don't contain "ws" or "websocket" so only headers trigger detection
	genNonWSPath := gen.OneConstOf("/api", "/data", "/events", "/rpc", "/health")

	properties.Property("Upgrade+Connection headers detected as ProtocolWebSocket", prop.ForAll(
		func(path string) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), "GET", path,
				map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"}, nil)
			return pd.DetectProtocol(req) == ProtocolWebSocket
		},
		genNonWSPath,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 5: gRPC content-type always detected ---
func TestPBT_GRPCContentTypeDetectedCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	grpcContentTypes := gen.OneConstOf(
		"application/grpc", "application/grpc+proto", "application/grpc-web",
	)

	properties.Property("grpc content-type detected as ProtocolGRPC", prop.ForAll(
		func(path string, contentType string) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), "POST", path,
				map[string]string{"Content-Type": contentType}, nil)
			return pd.DetectProtocol(req) == ProtocolGRPC
		},
		genSlashPath,
		grpcContentTypes,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 6: Registration doesn't affect detection results ---
func TestPBT_RegistrationDoesNotAffectDetection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("handler registration doesn't change detection", prop.ForAll(
		func(method string, path string, headers map[string]string, body []byte) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), method, path, headers, body)
			before := pd.DetectProtocol(req)

			_ = pd.RegisterHandler(ProtocolHTTP, &MockHandler{})
			_ = pd.RegisterHandler(ProtocolWebSocket, &MockHandler{})
			_ = pd.RegisterHandler(ProtocolGRPC, &MockHandler{})
			_ = pd.RegisterHandler(ProtocolGraphQL, &MockHandler{})

			after := pd.DetectProtocol(req)
			return before == after
		},
		genHTTPMethod,
		genSlashPath,
		genHeaderMap,
		genBody,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 7: Request abstraction preserves all fields ---
func TestPBT_RequestAbstractionPreservesFields(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 200
	properties := gopter.NewProperties(parameters)

	properties.Property("BaseRequest preserves method, path, headers, body", prop.ForAll(
		func(method string, path string, headers map[string]string, body []byte) bool {
			req := NewBaseRequest(context.Background(), method, path, headers, body)

			if req.Method() != method {
				return false
			}
			if req.Path() != path {
				return false
			}
			// Check body content match (nil body becomes empty)
			reqBody := req.Body()
			if body == nil {
				if len(reqBody) != 0 {
					return false
				}
			} else {
				if string(reqBody) != string(body) {
					return false
				}
			}
			// Check all headers present
			for k, v := range headers {
				if req.Header(k) != v {
					return false
				}
			}
			return true
		},
		genHTTPMethod,
		genSlashPath,
		genHeaderMap,
		genBody,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 8: Registered protocol count matches registrations ---
func TestPBT_ProtocolCountMatchesRegistrations(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generate a random number of registrations (0-4)
	genRegCount := gen.IntRange(0, 4)
	allProtocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL}

	properties.Property("protocol count matches number of registrations", prop.ForAll(
		func(count int) bool {
			pd := NewProtocolDetector()
			for i := 0; i < count; i++ {
				_ = pd.RegisterHandler(allProtocols[i], &MockHandler{})
			}
			return pd.GetSupportedProtocolCount() == count
		},
		genRegCount,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 9: GraphQL body with "query" key is always detected ---
func TestPBT_GraphQLBodyDetectedCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genGraphQLBody := gen.OneConstOf(
		`{"query":"{ users { id } }"}`,
		`{"mutation":"{ addUser }"}`,
		`{"query":"subscription { events }"}`,
	)

	properties.Property("body with query/mutation key detected as ProtocolGraphQL", prop.ForAll(
		func(path string, body string) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), "POST", path,
				map[string]string{"Content-Type": "application/json"}, []byte(body))
			return pd.DetectProtocol(req) == ProtocolGraphQL
		},
		genSlashPath,
		genGraphQLBody,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 10: Path with /ws always detected as WebSocket ---
func TestPBT_WSPathDetectedCorrectly(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("/ws path detected as ProtocolWebSocket", prop.ForAll(
		func(path string) bool {
			pd := NewProtocolDetector()
			req := NewBaseRequest(context.Background(), "GET", path, map[string]string{}, nil)
			return pd.DetectProtocol(req) == ProtocolWebSocket
		},
		genWSPath,
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 11: Registered protocols list contains all registered types ---
func TestPBT_RegisteredProtocolsListComplete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	allProtocols := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL}

	properties.Property("GetRegisteredProtocols contains all registered types", prop.ForAll(
		func(count int) bool {
			pd := NewProtocolDetector()
			registered := make(map[ProtocolType]bool)
			for i := 0; i < count; i++ {
				p := allProtocols[i]
				_ = pd.RegisterHandler(p, &MockHandler{})
				registered[p] = true
			}

			for _, p := range pd.GetRegisteredProtocols() {
				if !registered[p] {
					return false
				}
			}
			return len(pd.GetRegisteredProtocols()) == count
		},
		gen.IntRange(0, 4),
	))

	properties.Run(&pbtReporter{t: t})
}

// --- Property 12: Concurrent operations don't crash ---
func TestPBT_ConcurrentOperationsAreSafe(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	protocolTypes := []ProtocolType{ProtocolHTTP, ProtocolWebSocket, ProtocolGRPC, ProtocolGraphQL}

	properties.Property("concurrent register/detect/metrics don't panic", prop.ForAll(
		func(numOps int) bool {
			pd := NewProtocolDetector()
			done := make(chan bool, numOps*3)

			// Concurrent registrations
			for i := 0; i < numOps; i++ {
				go func(idx int) {
					defer func() { done <- true }()
					_ = pd.RegisterHandler(protocolTypes[idx%4], &MockHandler{})
				}(i)
			}

			// Concurrent detections
			for i := 0; i < numOps; i++ {
				go func(idx int) {
					defer func() { done <- true }()
					req := NewBaseRequest(context.Background(), "GET", "/test", nil, nil)
					_ = pd.DetectProtocol(req)
				}(i)
			}

			// Concurrent metrics reads
			for i := 0; i < numOps; i++ {
				go func() {
					defer func() { done <- true }()
					_ = pd.GetMetrics()
				}()
			}

			// Wait for all goroutines
			for i := 0; i < numOps*3; i++ {
				<-done
			}
			return true
		},
		gen.IntRange(1, 20),
	))

	properties.Run(&pbtReporter{t: t})
}
