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
