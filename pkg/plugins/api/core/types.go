package core

// ProtocolType represents the type of protocol
type ProtocolType int

const (
	ProtocolHTTP ProtocolType = iota
	ProtocolWebSocket
	ProtocolGRPC
	ProtocolGraphQL
	ProtocolUnknown
)

// HTTPMethod represents HTTP methods
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
	HEAD   HTTPMethod = "HEAD"
)

// StatusCode represents HTTP status codes
type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusCreated             StatusCode = 201
	StatusAccepted            StatusCode = 202
	StatusNoContent           StatusCode = 204
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusMethodNotAllowed    StatusCode = 405
	StatusConflict            StatusCode = 409
	StatusInternalServerError StatusCode = 500
	StatusNotImplemented      StatusCode = 501
	StatusServiceUnavailable  StatusCode = 503
)

// ContentType represents content types
type ContentType string

const (
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeXML       ContentType = "application/xml"
	ContentTypeText      ContentType = "text/plain"
	ContentTypeHTML      ContentType = "text/html"
	ContentTypeProtobuf  ContentType = "application/protobuf"
	ContentTypeGraphQL   ContentType = "application/graphql"
)



// RequestMetadata holds metadata about a request
type RequestMetadata struct {
	Protocol      ProtocolType
	ClientIP      string
	UserAgent     string
	RequestID     string
	Timestamp     int64
	ContentLength int64
}

// GetRuntimeMetrics returns a compact runtime surface for request metadata
// readiness on top of protocol identity and request attribution completeness.
func (m RequestMetadata) GetRuntimeMetrics() map[string]interface{} {
	protocolKnown := m.Protocol != ProtocolUnknown
	clientIdentified := m.ClientIP != ""
	requestTracked := m.RequestID != ""
	timestampSet := m.Timestamp > 0
	hasPayload := m.ContentLength > 0

	coveragePosture := classifyRequestMetadataCoveragePosture(protocolKnown, clientIdentified, requestTracked)
	runtimePosture := classifyRequestMetadataRuntimePosture(protocolKnown, timestampSet, requestTracked)

	return map[string]interface{}{
		"protocol":           m.Protocol,
		"client_ip_present":  clientIdentified,
		"user_agent_present": m.UserAgent != "",
		"request_id_present": requestTracked,
		"timestamp_set":      timestampSet,
		"content_length":     m.ContentLength,
		"payload_present":    hasPayload,
		"coverage_posture":   coveragePosture,
		"runtime_posture":    runtimePosture,
		"reliability_hint":   buildRequestMetadataReliabilityHint(coveragePosture, runtimePosture),
	}
}

// ResponseMetadata holds metadata about a response
type ResponseMetadata struct {
	Protocol      ProtocolType
	ContentLength int64
	Duration      int64 // milliseconds
	Timestamp     int64
}

// GetRuntimeMetrics returns a compact runtime surface for response metadata
// readiness on top of protocol identity and delivery timing completeness.
func (m ResponseMetadata) GetRuntimeMetrics() map[string]interface{} {
	protocolKnown := m.Protocol != ProtocolUnknown
	hasPayload := m.ContentLength > 0
	durationMeasured := m.Duration > 0
	timestampSet := m.Timestamp > 0

	coveragePosture := classifyResponseMetadataCoveragePosture(protocolKnown, hasPayload, durationMeasured)
	runtimePosture := classifyResponseMetadataRuntimePosture(protocolKnown, durationMeasured, timestampSet)

	return map[string]interface{}{
		"protocol":          m.Protocol,
		"content_length":    m.ContentLength,
		"payload_present":   hasPayload,
		"duration_ms":       m.Duration,
		"duration_measured": durationMeasured,
		"timestamp_set":     timestampSet,
		"coverage_posture":  coveragePosture,
		"runtime_posture":   runtimePosture,
		"reliability_hint":  buildResponseMetadataReliabilityHint(coveragePosture, runtimePosture),
	}
}

func classifyRequestMetadataCoveragePosture(protocolKnown bool, clientIdentified bool, requestTracked bool) string {
	if !protocolKnown {
		return "request-metadata-unconfigured"
	}
	if clientIdentified && requestTracked {
		return "request-metadata-attributed"
	}
	return "request-metadata-partial"
}

func classifyRequestMetadataRuntimePosture(protocolKnown bool, timestampSet bool, requestTracked bool) string {
	if !protocolKnown {
		return "request-metadata-unobserved"
	}
	if !timestampSet || !requestTracked {
		return "request-metadata-watch"
	}
	return "request-metadata-ready"
}

func buildRequestMetadataReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "request-metadata-unobserved":
		return "request metadata has unknown protocol; verify metadata initialization before relying on attribution"
	case runtimePosture == "request-metadata-watch":
		return "request metadata is missing request tracking or timestamp fields; verify observability wiring"
	case coveragePosture == "request-metadata-attributed":
		return "request metadata has protocol, client attribution, and request tracking ready for runtime observability"
	default:
		return "request metadata is partially populated; continue observing attribution coverage"
	}
}

func classifyResponseMetadataCoveragePosture(protocolKnown bool, hasPayload bool, durationMeasured bool) string {
	if !protocolKnown {
		return "response-metadata-unconfigured"
	}
	if hasPayload && durationMeasured {
		return "response-metadata-profiled"
	}
	return "response-metadata-partial"
}

func classifyResponseMetadataRuntimePosture(protocolKnown bool, durationMeasured bool, timestampSet bool) string {
	if !protocolKnown {
		return "response-metadata-unobserved"
	}
	if !durationMeasured || !timestampSet {
		return "response-metadata-watch"
	}
	return "response-metadata-ready"
}

func buildResponseMetadataReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "response-metadata-unobserved":
		return "response metadata has unknown protocol; verify protocol attribution before relying on response telemetry"
	case runtimePosture == "response-metadata-watch":
		return "response metadata is missing duration or timestamp fields; verify timing instrumentation"
	case coveragePosture == "response-metadata-profiled":
		return "response metadata has protocol, payload size, and duration measurement ready for runtime profiling"
	default:
		return "response metadata is partially populated; continue observing telemetry completeness"
	}
}
