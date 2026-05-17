package core

import (
	"fmt"
	"strings"
	"sync"
)

// ProtocolDetector detects incoming protocol and routes to appropriate handler
type ProtocolDetector struct {
	handlers map[ProtocolType]Handler
	mu       sync.RWMutex
}

// NewProtocolDetector creates a new protocol detector
func NewProtocolDetector() *ProtocolDetector {
	return &ProtocolDetector{
		handlers: make(map[ProtocolType]Handler),
	}
}

// RegisterHandler registers a handler for a specific protocol
func (pd *ProtocolDetector) RegisterHandler(protocol ProtocolType, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.handlers[protocol] = handler
	return nil
}

// DetectProtocol detects the protocol from a request
func (pd *ProtocolDetector) DetectProtocol(req Request) ProtocolType {
	if req == nil {
		return ProtocolUnknown
	}

	// Check for GraphQL
	if pd.isGraphQL(req) {
		return ProtocolGraphQL
	}

	// Check for WebSocket
	if pd.isWebSocket(req) {
		return ProtocolWebSocket
	}

	// Check for gRPC
	if pd.isGRPC(req) {
		return ProtocolGRPC
	}

	// Default to HTTP
	return ProtocolHTTP
}

// isGraphQL checks if request is GraphQL
func (pd *ProtocolDetector) isGraphQL(req Request) bool {
	path := req.Path()
	if strings.Contains(path, "graphql") {
		return true
	}

	contentType := req.Header("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body := string(req.Body())
		if strings.Contains(body, "query") || strings.Contains(body, "mutation") {
			return true
		}
	}

	return false
}

// isWebSocket checks if request is WebSocket
func (pd *ProtocolDetector) isWebSocket(req Request) bool {
	upgrade := req.Header("Upgrade")
	connection := req.Header("Connection")

	if strings.ToLower(upgrade) == "websocket" && strings.Contains(strings.ToLower(connection), "upgrade") {
		return true
	}

	path := req.Path()
	if strings.Contains(path, "ws") || strings.Contains(path, "websocket") {
		return true
	}

	return false
}

// isGRPC checks if request is gRPC
func (pd *ProtocolDetector) isGRPC(req Request) bool {
	contentType := req.Header("Content-Type")
	if strings.Contains(contentType, "application/grpc") {
		return true
	}

	// Check for gRPC method header
	if req.Header("grpc-encoding") != "" {
		return true
	}

	return false
}

// Route routes a request to the appropriate handler
func (pd *ProtocolDetector) Route(req Request) (Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	protocol := pd.DetectProtocol(req)
	handler := pd.getHandler(protocol)

	if handler == nil {
		return nil, fmt.Errorf("no handler registered for protocol: %v", protocol)
	}

	return handler.Handle(req)
}

// getHandler returns the handler for a specific protocol
func (pd *ProtocolDetector) getHandler(protocol ProtocolType) Handler {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	handler, ok := pd.handlers[protocol]
	if !ok {
		return nil
	}

	return handler
}

// GetProtocolName returns the name of a protocol
func GetProtocolName(protocol ProtocolType) string {
	switch protocol {
	case ProtocolHTTP:
		return "HTTP"
	case ProtocolWebSocket:
		return "WebSocket"
	case ProtocolGRPC:
		return "gRPC"
	case ProtocolGraphQL:
		return "GraphQL"
	default:
		return "Unknown"
	}
}

// GetRegisteredProtocols returns all registered protocols
func (pd *ProtocolDetector) GetRegisteredProtocols() []ProtocolType {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	protocols := make([]ProtocolType, 0, len(pd.handlers))
	for protocol := range pd.handlers {
		protocols = append(protocols, protocol)
	}

	return protocols
}

// IsProtocolSupported checks if a protocol is supported
func (pd *ProtocolDetector) IsProtocolSupported(protocol ProtocolType) bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	_, ok := pd.handlers[protocol]
	return ok
}

// GetSupportedProtocolCount returns the number of supported protocols
func (pd *ProtocolDetector) GetSupportedProtocolCount() int {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	return len(pd.handlers)
}

// GetMetrics returns protocol detection metrics
func (pd *ProtocolDetector) GetMetrics() map[string]any {
	pd.mu.RLock()
	protocols := make([]string, 0, len(pd.handlers))
	for protocol := range pd.handlers {
		protocols = append(protocols, GetProtocolName(protocol))
	}
	protocolCount := len(pd.handlers)
	pd.mu.RUnlock()

	httpSupported := pd.IsProtocolSupported(ProtocolHTTP)
	websocketSupported := pd.IsProtocolSupported(ProtocolWebSocket)
	grpcSupported := pd.IsProtocolSupported(ProtocolGRPC)
	graphqlSupported := pd.IsProtocolSupported(ProtocolGraphQL)
	coveragePosture := classifyDetectorCoveragePosture(protocolCount, httpSupported, websocketSupported, grpcSupported, graphqlSupported)
	runtimePosture := classifyDetectorRuntimePosture(protocolCount, httpSupported, websocketSupported, grpcSupported, graphqlSupported)

	return map[string]any{
		"supported_protocols": protocols,
		"protocol_count":      protocolCount,
		"coverage_posture":    coveragePosture,
		"runtime_posture":     runtimePosture,
		"reliability_hint":    buildDetectorReliabilityHint(coveragePosture, runtimePosture),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for protocol coverage and
// detector readiness on top of the raw supported-protocol metrics.
func (pd *ProtocolDetector) GetRuntimeMetrics() map[string]any {
	metrics := pd.GetMetrics()

	supportedProtocols, _ := metrics["supported_protocols"].([]string)
	protocolCount, _ := metrics["protocol_count"].(int)

	return map[string]any{
		"supported_protocols": supportedProtocols,
		"protocol_count":      protocolCount,
		"coverage_posture":    metrics["coverage_posture"],
		"runtime_posture":     metrics["runtime_posture"],
		"reliability_hint":    metrics["reliability_hint"],
	}
}

func classifyDetectorCoveragePosture(protocolCount int, httpSupported bool, websocketSupported bool, grpcSupported bool, graphqlSupported bool) string {
	if protocolCount == 0 {
		return "detector-empty"
	}
	if httpSupported && websocketSupported && grpcSupported && graphqlSupported {
		return "detector-full-stack"
	}
	if httpSupported && protocolCount == 1 {
		return "detector-http-only"
	}
	return "detector-partial"
}

func classifyDetectorRuntimePosture(protocolCount int, httpSupported bool, websocketSupported bool, grpcSupported bool, graphqlSupported bool) string {
	if protocolCount == 0 {
		return "detector-unobserved"
	}
	if !httpSupported {
		return "detector-degraded"
	}
	if websocketSupported || grpcSupported || graphqlSupported {
		return "detector-ready"
	}
	return "detector-watch"
}

func buildDetectorReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "detector-degraded":
		return "protocol detector is missing HTTP coverage; verify baseline handler wiring before relying on protocol routing"
	case runtimePosture == "detector-watch":
		return "protocol detector currently routes only baseline HTTP traffic; add more protocol handlers if broader coverage is expected"
	case coveragePosture == "detector-partial":
		return "protocol detector has partial multi-protocol coverage; continue observing whether supported protocols match deployment needs"
	case coveragePosture == "detector-full-stack":
		return "protocol detector has broad protocol coverage and is ready for multi-protocol routing"
	default:
		return "protocol detector has no registered protocol handlers yet"
	}
}
