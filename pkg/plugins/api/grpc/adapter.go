package grpc

import (
	"context"
	"encoding/json"
)

// GRPCRequest adapts gRPC request to core.Request interface
type GRPCRequest struct {
	method  string
	path    string
	headers map[string]string
	body    []byte
	ctx     context.Context
}

// NewGRPCRequest creates a new gRPC request adapter
func NewGRPCRequest(method, path string, headers map[string]string, body []byte, ctx context.Context) *GRPCRequest { //nolint:revive // ctx cannot be first param; method is the primary identifier
	if headers == nil {
		headers = make(map[string]string)
	}
	return &GRPCRequest{
		method:  method,
		path:    path,
		headers: headers,
		body:    body,
		ctx:     ctx,
	}
}

// Method returns the request method
func (r *GRPCRequest) Method() string {
	return r.method
}

// Path returns the request path
func (r *GRPCRequest) Path() string {
	return r.path
}

// Headers returns all request headers
func (r *GRPCRequest) Headers() map[string]string {
	return r.headers
}

// Header returns a specific header value
func (r *GRPCRequest) Header(key string) string {
	return r.headers[key]
}

// Body returns the request body
func (r *GRPCRequest) Body() []byte {
	return r.body
}

// Context returns the request context
func (r *GRPCRequest) Context() context.Context {
	return r.ctx
}

// Query returns query parameters (empty for gRPC)
func (r *GRPCRequest) Query() map[string]string {
	return make(map[string]string)
}

// QueryParam returns a specific query parameter (empty for gRPC)
func (r *GRPCRequest) QueryParam(key string) string {
	return ""
}

// PathParam returns a path parameter (empty for gRPC)
func (r *GRPCRequest) PathParam(key string) string {
	return ""
}

// GRPCResponse adapts gRPC response to core.Response interface
type GRPCResponse struct {
	status      int
	headers     map[string]string
	body        []byte
	messageSent bool
}

// NewGRPCResponse creates a new gRPC response adapter
func NewGRPCResponse() *GRPCResponse {
	return &GRPCResponse{
		status:      200,
		headers:     make(map[string]string),
		body:        make([]byte, 0),
		messageSent: false,
	}
}

// SetStatus sets the response status code
func (r *GRPCResponse) SetStatus(code int) {
	if !r.messageSent {
		r.status = code
	}
}

// Status returns the current status code
func (r *GRPCResponse) Status() int {
	return r.status
}

// SetHeader sets a response header
func (r *GRPCResponse) SetHeader(key, value string) {
	if !r.messageSent {
		r.headers[key] = value
	}
}

// Header returns a specific header value
func (r *GRPCResponse) Header(key string) string {
	return r.headers[key]
}

// Headers returns all response headers
func (r *GRPCResponse) Headers() map[string]string {
	return r.headers
}

// SetBody sets the response body
func (r *GRPCResponse) SetBody(data []byte) {
	r.body = data
}

// Body returns the response body
func (r *GRPCResponse) Body() []byte {
	return r.body
}

// Write writes data to the response body
func (r *GRPCResponse) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}

// Send sends the response
func (r *GRPCResponse) Send() error {
	r.messageSent = true
	return nil
}

// IsMessageSent returns whether the message has been sent
func (r *GRPCResponse) IsMessageSent() bool {
	return r.messageSent
}

// IsHeadersSent returns whether the response has been sent.
func (r *GRPCResponse) IsHeadersSent() bool {
	return r.messageSent
}

// ToJSON converts response to JSON for gRPC transport
func (r *GRPCResponse) ToJSON() ([]byte, error) {
	envelope := map[string]interface{}{
		"status":  r.status,
		"headers": r.headers,
		"body":    string(r.body),
	}
	return json.Marshal(envelope)
}

// FromJSON creates response from JSON
func (r *GRPCResponse) FromJSON(data []byte) error {
	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}

	if status, ok := envelope["status"].(float64); ok {
		r.status = int(status)
	}

	if headers, ok := envelope["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				r.headers[key] = strVal
			}
		}
	}

	if body, ok := envelope["body"].(string); ok {
		r.body = []byte(body)
	}

	return nil
}
