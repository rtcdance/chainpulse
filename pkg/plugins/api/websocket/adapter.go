package websocket

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

// Request adapts WebSocket message to core.Request interface.
type Request struct {
	httpReq *http.Request
	data    []byte
	headers map[string]string
}

// NewWebSocketRequest creates a new WebSocket request adapter
func NewWebSocketRequest(r *http.Request, data []byte) *Request {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &Request{
		httpReq: r,
		data:    data,
		headers: headers,
	}
}

// Method returns the HTTP method (always GET for WebSocket upgrade)
func (r *Request) Method() string {
	return r.httpReq.Method
}

// Path returns the request path
func (r *Request) Path() string {
	return r.httpReq.URL.Path
}

// Headers returns all request headers
func (r *Request) Headers() map[string]string {
	return r.headers
}

// Header returns a specific header value
func (r *Request) Header(key string) string {
	return r.headers[key]
}

// Body returns the message data
func (r *Request) Body() []byte {
	return r.data
}

// Context returns the request context
func (r *Request) Context() context.Context {
	return r.httpReq.Context()
}

// Query returns query parameters from the WebSocket upgrade request
func (r *Request) Query() map[string]string {
	query := make(map[string]string)
	for key, values := range r.httpReq.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}
	return query
}

// QueryParam returns a specific query parameter
func (r *Request) QueryParam(key string) string {
	return r.httpReq.URL.Query().Get(key)
}

// PathParam returns a path parameter (not directly available)
func (r *Request) PathParam(_ string) string {
	return ""
}

// Response adapts WebSocket connection to core.Response interface.
type Response struct {
	conn        *websocket.Conn
	status      int
	headers     map[string]string
	body        []byte
	messageSent bool
}

// NewWebSocketResponse creates a new WebSocket response adapter
func NewWebSocketResponse(conn *websocket.Conn) *Response {
	return &Response{
		conn:        conn,
		status:      200,
		headers:     make(map[string]string),
		body:        make([]byte, 0),
		messageSent: false,
	}
}

// SetStatus sets the response status (stored in headers for WebSocket)
func (r *Response) SetStatus(code int) {
	if !r.messageSent {
		r.status = code
	}
}

// Status returns the current status code
func (r *Response) Status() int {
	return r.status
}

// SetHeader sets a response header (stored for metadata)
func (r *Response) SetHeader(key, value string) {
	if !r.messageSent {
		r.headers[key] = value
	}
}

// Header returns a specific header value
func (r *Response) Header(key string) string {
	return r.headers[key]
}

// Headers returns all response headers
func (r *Response) Headers() map[string]string {
	return r.headers
}

// SetBody sets the response body
func (r *Response) SetBody(data []byte) {
	r.body = data
}

// Body returns the response body
func (r *Response) Body() []byte {
	return r.body
}

// Write writes data to the response body
func (r *Response) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}

// Send sends the response through WebSocket
func (r *Response) Send() error {
	r.messageSent = true

	// Create response envelope with status and headers
	envelope := map[string]interface{}{
		"status":  r.status,
		"headers": r.headers,
		"body":    string(r.body),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return r.conn.WriteMessage(websocket.TextMessage, data)
}

// IsMessageSent returns whether the message has been sent
func (r *Response) IsMessageSent() bool {
	return r.messageSent
}

// IsHeadersSent returns whether the response has been sent.
func (r *Response) IsHeadersSent() bool {
	return r.messageSent
}
