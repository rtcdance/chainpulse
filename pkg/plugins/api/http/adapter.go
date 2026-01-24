package http

import (
	"context"
	"io"
	"net/http"
)

// HTTPRequest adapts http.Request to core.Request interface
type HTTPRequest struct {
	req *http.Request
}

// NewHTTPRequest creates a new HTTP request adapter
func NewHTTPRequest(r *http.Request) *HTTPRequest {
	return &HTTPRequest{req: r}
}

// Method returns the HTTP method
func (r *HTTPRequest) Method() string {
	return r.req.Method
}

// Path returns the request path
func (r *HTTPRequest) Path() string {
	return r.req.URL.Path
}

// Headers returns all request headers
func (r *HTTPRequest) Headers() map[string]string {
	headers := make(map[string]string)
	for key, values := range r.req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// Header returns a specific header value
func (r *HTTPRequest) Header(key string) string {
	return r.req.Header.Get(key)
}

// Body returns the request body
func (r *HTTPRequest) Body() []byte {
	if r.req.Body == nil {
		return []byte{}
	}
	body, _ := io.ReadAll(r.req.Body)
	return body
}

// Context returns the request context
func (r *HTTPRequest) Context() context.Context {
	return r.req.Context()
}

// Query returns query parameters
func (r *HTTPRequest) Query() map[string]string {
	query := make(map[string]string)
	for key, values := range r.req.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	return query
}

// QueryParam returns a specific query parameter
func (r *HTTPRequest) QueryParam(key string) string {
	return r.req.URL.Query().Get(key)
}

// PathParam returns a path parameter (not directly available in http.Request)
func (r *HTTPRequest) PathParam(_ string) string {
	return ""
}

// HTTPResponse adapts http.ResponseWriter to core.Response interface
type HTTPResponse struct {
	writer      http.ResponseWriter
	status      int
	headers     map[string]string
	body        []byte
	headersSent bool
}

// NewHTTPResponse creates a new HTTP response adapter
func NewHTTPResponse(w http.ResponseWriter) *HTTPResponse {
	return &HTTPResponse{
		writer:      w,
		status:      200,
		headers:     make(map[string]string),
		body:        make([]byte, 0),
		headersSent: false,
	}
}

// SetStatus sets the HTTP status code
func (r *HTTPResponse) SetStatus(code int) {
	if !r.headersSent {
		r.status = code
	}
}

// Status returns the current status code
func (r *HTTPResponse) Status() int {
	return r.status
}

// SetHeader sets a response header
func (r *HTTPResponse) SetHeader(key, value string) {
	if !r.headersSent {
		r.headers[key] = value
	}
}

// Header returns a specific header value
func (r *HTTPResponse) Header(key string) string {
	return r.headers[key]
}

// Headers returns all response headers
func (r *HTTPResponse) Headers() map[string]string {
	return r.headers
}

// SetBody sets the response body
func (r *HTTPResponse) SetBody(data []byte) {
	r.body = data
}

// Body returns the response body
func (r *HTTPResponse) Body() []byte {
	return r.body
}

// Write writes data to the response
func (r *HTTPResponse) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}

// Send sends the response
func (r *HTTPResponse) Send() error {
	r.headersSent = true
	for key, value := range r.headers {
		r.writer.Header().Set(key, value)
	}
	r.writer.WriteHeader(r.status)
	_, err := r.writer.Write(r.body)
	return err
}

// IsHeadersSent returns whether headers have been sent
func (r *HTTPResponse) IsHeadersSent() bool {
	return r.headersSent
}
