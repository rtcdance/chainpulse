package core

import "context"

// Request defines the protocol-agnostic request interface
type Request interface {
	// Method returns the HTTP method (GET, POST, etc.) or equivalent
	Method() string

	// Path returns the request path
	Path() string

	// Headers returns all request headers
	Headers() map[string]string

	// Header returns a specific header value
	Header(key string) string

	// Body returns the request body as bytes
	Body() []byte

	// Context returns the request context
	Context() context.Context

	// Query returns query parameters
	Query() map[string]string

	// QueryParam returns a specific query parameter
	QueryParam(key string) string

	// PathParam returns a path parameter
	PathParam(key string) string
}

// BaseRequest provides a base implementation of Request interface
type BaseRequest struct {
	method     string
	path       string
	headers    map[string]string
	body       []byte
	ctx        context.Context
	query      map[string]string
	pathParams map[string]string
}

// NewBaseRequest creates a new base request
func NewBaseRequest(ctx context.Context, method, path string, headers map[string]string, body []byte) *BaseRequest {
	if headers == nil {
		headers = make(map[string]string)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &BaseRequest{
		method:     method,
		path:       path,
		headers:    headers,
		body:       body,
		ctx:        ctx,
		query:      make(map[string]string),
		pathParams: make(map[string]string),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for request readiness
// on top of path/method metadata and parameter/payload coverage.
func (r *BaseRequest) GetRuntimeMetrics() map[string]interface{} {
	headerCount := len(r.headers)
	queryCount := len(r.query)
	pathParamCount := len(r.pathParams)
	bodyBytes := len(r.body)

	coveragePosture := classifyBaseRequestCoveragePosture(headerCount, queryCount, pathParamCount, bodyBytes)
	runtimePosture := classifyBaseRequestRuntimePosture(r.method, r.path, coveragePosture)

	return map[string]interface{}{
		"method":           r.method,
		"path":             r.path,
		"header_count":     headerCount,
		"query_count":      queryCount,
		"path_param_count": pathParamCount,
		"body_bytes":       bodyBytes,
		"coverage_posture": coveragePosture,
		"runtime_posture":  runtimePosture,
		"reliability_hint": buildBaseRequestReliabilityHint(coveragePosture, runtimePosture),
	}
}

// Method returns the HTTP method
func (r *BaseRequest) Method() string {
	return r.method
}

// Path returns the request path
func (r *BaseRequest) Path() string {
	return r.path
}

// Headers returns all request headers
func (r *BaseRequest) Headers() map[string]string {
	return r.headers
}

// Header returns a specific header value
func (r *BaseRequest) Header(key string) string {
	return r.headers[key]
}

// Body returns the request body
func (r *BaseRequest) Body() []byte {
	return r.body
}

// Context returns the request context
func (r *BaseRequest) Context() context.Context {
	return r.ctx
}

// Query returns query parameters
func (r *BaseRequest) Query() map[string]string {
	return r.query
}

// QueryParam returns a specific query parameter
func (r *BaseRequest) QueryParam(key string) string {
	return r.query[key]
}

// PathParam returns a path parameter
func (r *BaseRequest) PathParam(key string) string {
	return r.pathParams[key]
}

// SetQuery sets query parameters
func (r *BaseRequest) SetQuery(query map[string]string) {
	r.query = query
}

// SetPathParam sets a path parameter
func (r *BaseRequest) SetPathParam(key, value string) {
	r.pathParams[key] = value
}

func classifyBaseRequestCoveragePosture(headerCount int, queryCount int, pathParamCount int, bodyBytes int) string {
	if headerCount == 0 && queryCount == 0 && pathParamCount == 0 && bodyBytes == 0 {
		return "request-minimal"
	}
	if bodyBytes > 0 || queryCount > 0 || pathParamCount > 0 {
		return "request-parameterized"
	}
	return "request-headered"
}

func classifyBaseRequestRuntimePosture(method string, path string, coveragePosture string) string {
	if method == "" || path == "" {
		return "request-degraded"
	}
	if coveragePosture == "request-minimal" {
		return "request-staged"
	}
	if coveragePosture == "request-parameterized" {
		return "request-ready"
	}
	return "request-watch"
}

func buildBaseRequestReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "request-degraded":
		return "request is missing method or path; verify routing metadata before relying on runtime handling"
	case runtimePosture == "request-staged":
		return "request has method/path but no headers or parameters yet; enrich metadata if downstream handlers depend on it"
	case runtimePosture == "request-ready":
		return "request has method/path with payload or parameters and is ready for handler processing"
	case coveragePosture == "request-headered":
		return "request currently carries headers only; verify whether query/path/body inputs are expected for this route"
	default:
		return "request runtime shape is partially populated; continue observing request construction"
	}
}
