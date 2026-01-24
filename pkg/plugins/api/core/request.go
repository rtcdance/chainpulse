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
func NewBaseRequest(method, path string, headers map[string]string, body []byte, ctx context.Context) *BaseRequest {
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
