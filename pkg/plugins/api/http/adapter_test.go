package http

import (
	"bytes"
	"context"
	"net/http"
	"testing"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// userIDContextKey is the key for storing user ID in context.Context
	userIDContextKey contextKey = "user_id"
)

func TestNewHTTPRequest(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	httpReq := NewHTTPRequest(req)

	if httpReq == nil {
		t.Fatal("expected request, got nil")
	}

	if httpReq.Method() != "GET" {
		t.Errorf("expected method GET, got %s", httpReq.Method())
	}
}

func TestHTTPRequestMethod(t *testing.T) {
	t.Parallel()
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req, _ := http.NewRequest(method, "/api/test", nil)
		httpReq := NewHTTPRequest(req)

		if httpReq.Method() != method {
			t.Errorf("expected method %s, got %s", method, httpReq.Method())
		}
	}
}

func TestHTTPRequestPath(t *testing.T) {
	t.Parallel()
	paths := []string{"/api/users", "/api/posts/1", "/health"}

	for _, path := range paths {
		req, _ := http.NewRequest("GET", path, nil)
		httpReq := NewHTTPRequest(req)

		if httpReq.Path() != path {
			t.Errorf("expected path %s, got %s", path, httpReq.Path())
		}
	}
}

func TestHTTPRequestHeaders(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	httpReq := NewHTTPRequest(req)

	if httpReq.Header("Content-Type") != "application/json" {
		t.Error("expected Content-Type header")
	}

	if httpReq.Header("Authorization") != "Bearer token" {
		t.Error("expected Authorization header")
	}

	headers := httpReq.Headers()
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
}

func TestHTTPRequestBody(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name":"test"}`)
	req, _ := http.NewRequest("POST", "/api/users", bytes.NewReader(body))

	httpReq := NewHTTPRequest(req)

	if string(httpReq.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(httpReq.Body()))
	}
}

func TestHTTPRequestContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), userIDContextKey, "123")
	req, _ := http.NewRequestWithContext(ctx, "GET", "/api/test", nil)

	httpReq := NewHTTPRequest(req)

	if httpReq.Context().Value(userIDContextKey) != "123" {
		t.Error("expected context value")
	}
}

func TestHTTPRequestPathParam(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("GET", "/api/users/123", nil)
	httpReq := NewHTTPRequest(req)

	if httpReq.PathParam("id") != "" {
		t.Error("expected empty path param")
	}
}

func TestHTTPRequestQuery(t *testing.T) {
	t.Parallel()
	req, _ := http.NewRequest("GET", "/api/users?page=1&limit=10", nil)
	httpReq := NewHTTPRequest(req)

	if httpReq.QueryParam("page") != "1" {
		t.Error("expected page=1")
	}

	if httpReq.QueryParam("limit") != "10" {
		t.Error("expected limit=10")
	}

	query := httpReq.Query()
	if len(query) != 2 {
		t.Errorf("expected 2 query params, got %d", len(query))
	}
}

func TestNewHTTPResponse(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	w := &testResponseWriter{buf: buf}

	resp := NewHTTPResponse(w)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestHTTPResponseStatus(t *testing.T) {
	t.Parallel()
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	resp.SetStatus(201)
	if resp.Status() != 201 {
		t.Errorf("expected status 201, got %d", resp.Status())
	}

	resp.SetStatus(404)
	if resp.Status() != 404 {
		t.Errorf("expected status 404, got %d", resp.Status())
	}
}

func TestHTTPResponseHeaders(t *testing.T) {
	t.Parallel()
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	resp.SetHeader("Content-Type", "application/json")
	resp.SetHeader("X-Custom", "value")

	if resp.Header("Content-Type") != "application/json" {
		t.Error("expected Content-Type header")
	}

	headers := resp.Headers()
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
}

func TestHTTPResponseBody(t *testing.T) {
	t.Parallel()
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	body := []byte(`{"status":"ok"}`)
	resp.SetBody(body)

	if string(resp.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(resp.Body()))
	}
}

func TestHTTPResponseWrite(t *testing.T) {
	t.Parallel()
	w := &testResponseWriter{buf: &bytes.Buffer{}}
	resp := NewHTTPResponse(w)

	n, err := resp.Write([]byte("hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	if string(resp.Body()) != "hello" {
		t.Errorf("expected body 'hello', got %s", string(resp.Body()))
	}
}

func TestHTTPResponseSend(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	w := &testResponseWriter{buf: buf}
	resp := NewHTTPResponse(w)

	resp.SetStatus(201)
	resp.SetHeader("Location", "/api/users/1")
	resp.SetBody([]byte(`{"id":1}`))

	err := resp.Send()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !resp.IsHeadersSent() {
		t.Error("expected headers to be sent")
	}

	if buf.String() != `{"id":1}` {
		t.Errorf("expected body in buffer, got %s", buf.String())
	}
}

// testResponseWriter is a test helper
type testResponseWriter struct {
	buf     *bytes.Buffer
	headers http.Header
}

func (w *testResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	// No-op for test
}
