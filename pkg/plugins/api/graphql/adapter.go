package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// GraphQLRequest adapts HTTP request to GraphQL request
type GraphQLRequest struct {
	httpReq    *http.Request
	body       []byte
	pathParams map[string]string
}

// NewGraphQLRequest creates a new GraphQL request adapter
func NewGraphQLRequest(httpReq *http.Request) *GraphQLRequest {
	body, _ := io.ReadAll(httpReq.Body)
	httpReq.Body = io.NopCloser(bytes.NewReader(body))

	return &GraphQLRequest{
		httpReq:    httpReq,
		body:       body,
		pathParams: make(map[string]string),
	}
}

// Method returns the HTTP method
func (r *GraphQLRequest) Method() string {
	return r.httpReq.Method
}

// Path returns the request path
func (r *GraphQLRequest) Path() string {
	return r.httpReq.URL.Path
}

// Body returns the request body
func (r *GraphQLRequest) Body() []byte {
	return r.body
}

// Context returns the request context
func (r *GraphQLRequest) Context() context.Context {
	return r.httpReq.Context()
}

// Header returns a header value
func (r *GraphQLRequest) Header(key string) string {
	return r.httpReq.Header.Get(key)
}

// Headers returns all headers
func (r *GraphQLRequest) Headers() map[string]string {
	headers := make(map[string]string)
	for key, values := range r.httpReq.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// Query returns query parameters
func (r *GraphQLRequest) Query() map[string]string {
	return parseQueryParams(r.httpReq.URL.RawQuery)
}

// QueryParam returns a query parameter value
func (r *GraphQLRequest) QueryParam(key string) string {
	return r.httpReq.URL.Query().Get(key)
}

// PathParam returns a path parameter value
func (r *GraphQLRequest) PathParam(key string) string {
	return r.pathParams[key]
}

// GetGraphQLQuery extracts the GraphQL query from the request body
func (r *GraphQLRequest) GetGraphQLQuery() (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(r.body, &payload); err != nil {
		return "", err
	}

	query, ok := payload["query"].(string)
	if !ok {
		return "", nil
	}

	return query, nil
}

// GetGraphQLVariables extracts the GraphQL variables from the request body
func (r *GraphQLRequest) GetGraphQLVariables() (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(r.body, &payload); err != nil {
		return nil, err
	}

	variables, ok := payload["variables"].(map[string]interface{})
	if !ok {
		return make(map[string]interface{}), nil
	}

	return variables, nil
}

// GraphQLResponse adapts HTTP response to GraphQL response
type GraphQLResponse struct {
	w       http.ResponseWriter
	status  int
	headers map[string]string
	body    bytes.Buffer
	sent    bool
	mu      sync.RWMutex
}

// NewGraphQLResponse creates a new GraphQL response adapter
func NewGraphQLResponse(w http.ResponseWriter) *GraphQLResponse {
	return &GraphQLResponse{
		w:       w,
		status:  200,
		headers: make(map[string]string),
	}
}

// Status returns the response status code
func (r *GraphQLResponse) Status() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

// SetStatus sets the response status code
func (r *GraphQLResponse) SetStatus(status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sent {
		r.status = status
	}
}

// Body returns the response body
func (r *GraphQLResponse) Body() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.body.Bytes()
}

// SetBody sets the response body
func (r *GraphQLResponse) SetBody(body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sent {
		r.body.Reset()
		r.body.Write(body)
	}
}

// Header returns a header value
func (r *GraphQLResponse) Header(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.headers[key]
}

// SetHeader sets a header value
func (r *GraphQLResponse) SetHeader(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sent {
		r.headers[key] = value
	}
}

// Write writes data to the response body
func (r *GraphQLResponse) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Write(data)
}

// Send sends the response
func (r *GraphQLResponse) Send() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sent {
		return nil
	}

	// Set default Content-Type if not already set
	if _, exists := r.headers["Content-Type"]; !exists {
		r.headers["Content-Type"] = "application/json"
	}

	// Write headers
	for key, value := range r.headers {
		r.w.Header().Set(key, value)
	}

	// Write status
	r.w.WriteHeader(r.status)

	// Write body
	_, _ = r.w.Write(r.body.Bytes())

	r.sent = true
	return nil
}

// SetGraphQLResult sets a GraphQL result with data and errors
func (r *GraphQLResponse) SetGraphQLResult(data map[string]interface{}, errors []error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sent {
		return nil
	}

	result := make(map[string]interface{})
	result["data"] = data

	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i := range errors {
			errorMessages[i] = "internal error"
		}
		result["errors"] = errorMessages
	}

	body, err := json.Marshal(result)
	if err != nil {
		return err
	}

	r.body.Reset()
	r.body.Write(body)
	r.headers["Content-Type"] = "application/json"

	return nil
}

// Headers returns all headers
func (r *GraphQLResponse) Headers() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.headers
}

// IsHeadersSent returns whether headers have been sent
func (r *GraphQLResponse) IsHeadersSent() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sent
}

// parseQueryParams parses query parameters from a query string
func parseQueryParams(query string) map[string]string {
	params := make(map[string]string)
	if query == "" {
		return params
	}

	pairs := bytes.Split([]byte(query), []byte("&"))
	for _, pair := range pairs {
		parts := bytes.Split(pair, []byte("="))
		if len(parts) == 2 {
			key := string(parts[0])
			value := string(parts[1])
			params[key] = value
		}
	}

	return params
}
