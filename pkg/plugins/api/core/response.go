package core

import "io"

// Response defines the protocol-agnostic response interface
type Response interface {
	// SetStatus sets the HTTP status code
	SetStatus(code int)

	// Status returns the current status code
	Status() int

	// SetHeader sets a response header
	SetHeader(key, value string)

	// Header returns a specific header value
	Header(key string) string

	// Headers returns all response headers
	Headers() map[string]string

	// SetBody sets the response body
	SetBody(data []byte)

	// Body returns the response body
	Body() []byte

	// Write writes data to the response
	Write(data []byte) (int, error)

	// Send sends the response (protocol-specific implementation)
	Send() error

	// IsHeadersSent returns whether headers have been sent
	IsHeadersSent() bool
}

// BaseResponse provides a base implementation of Response interface
type BaseResponse struct {
	status      int
	headers     map[string]string
	body        []byte
	headersSent bool
	writer      io.Writer
}

// NewBaseResponse creates a new base response
func NewBaseResponse(writer io.Writer) *BaseResponse {
	if writer == nil {
		writer = io.Discard
	}
	return &BaseResponse{
		status:      200,
		headers:     make(map[string]string),
		body:        make([]byte, 0),
		headersSent: false,
		writer:      writer,
	}
}

// GetRuntimeMetrics returns a compact runtime surface for response readiness
// on top of status, body/header coverage, send state, and writer presence.
func (r *BaseResponse) GetRuntimeMetrics() map[string]any {
	headerCount := len(r.headers)
	bodyBytes := len(r.body)
	writerConfigured := r.writer != nil

	coveragePosture := classifyBaseResponseCoveragePosture(headerCount, bodyBytes)
	runtimePosture := classifyBaseResponseRuntimePosture(r.headersSent, writerConfigured, bodyBytes)

	return map[string]any{
		"status":            r.status,
		"header_count":      headerCount,
		"body_bytes":        bodyBytes,
		"headers_sent":      r.headersSent,
		"writer_configured": writerConfigured,
		"coverage_posture":  coveragePosture,
		"runtime_posture":   runtimePosture,
		"reliability_hint":  buildBaseResponseReliabilityHint(coveragePosture, runtimePosture),
	}
}

// SetStatus sets the HTTP status code
func (r *BaseResponse) SetStatus(code int) {
	if !r.headersSent {
		r.status = code
	}
}

// Status returns the current status code
func (r *BaseResponse) Status() int {
	return r.status
}

// SetHeader sets a response header
func (r *BaseResponse) SetHeader(key, value string) {
	if !r.headersSent {
		r.headers[key] = value
	}
}

// Header returns a specific header value
func (r *BaseResponse) Header(key string) string {
	return r.headers[key]
}

// Headers returns all response headers
func (r *BaseResponse) Headers() map[string]string {
	return r.headers
}

// SetBody sets the response body
func (r *BaseResponse) SetBody(data []byte) {
	r.body = data
}

// Body returns the response body
func (r *BaseResponse) Body() []byte {
	return r.body
}

// Write writes data to the response
func (r *BaseResponse) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}

// Send sends the response
func (r *BaseResponse) Send() error {
	r.headersSent = true
	_, err := r.writer.Write(r.body)
	return err
}

// IsHeadersSent returns whether headers have been sent
func (r *BaseResponse) IsHeadersSent() bool {
	return r.headersSent
}

func classifyBaseResponseCoveragePosture(headerCount int, bodyBytes int) string {
	if headerCount == 0 && bodyBytes == 0 {
		return "response-empty"
	}
	if headerCount == 0 {
		return "response-body-only"
	}
	if bodyBytes == 0 {
		return "response-headers-only"
	}
	return "response-complete"
}

func classifyBaseResponseRuntimePosture(headersSent bool, writerConfigured bool, bodyBytes int) string {
	if !writerConfigured {
		return "response-degraded"
	}
	if headersSent {
		return "response-sent"
	}
	if bodyBytes == 0 {
		return "response-staged"
	}
	return "response-ready"
}

func buildBaseResponseReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "response-degraded":
		return "response writer is not configured; verify response sink wiring before relying on send behavior"
	case runtimePosture == "response-sent":
		return "response has already been sent; further status and header mutations will not apply"
	case coveragePosture == "response-empty":
		return "response has no headers or body yet; populate payload before treating it as ready"
	case coveragePosture == "response-complete" && runtimePosture == "response-ready":
		return "response has headers and body staged with a writer configured and is ready to send"
	default:
		return "response is partially staged; verify payload completeness before sending"
	}
}
