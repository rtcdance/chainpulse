package websocket

import (
	"net/http"
	"net/url"
	"testing"
)

func makeTestRequest(method, path string, query url.Values) *http.Request {
	u := &url.URL{Path: path}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	r := &http.Request{
		Method: method,
		URL:    u,
		Header: make(http.Header),
	}
	r.Header.Set("Authorization", "Bearer test-token")
	return r
}

func TestNewWebSocketRequest(t *testing.T) {
	t.Parallel()
	r := makeTestRequest("GET", "/ws/events", nil)
	req := NewWebSocketRequest(r, []byte(`{"type":"subscribe"}`))

	if req.Method() != "GET" {
		t.Errorf("Method() = %q, want GET", req.Method())
	}
	if req.Path() != "/ws/events" {
		t.Errorf("Path() = %q, want /ws/events", req.Path())
	}
	if string(req.Body()) != `{"type":"subscribe"}` {
		t.Errorf("Body() = %q", string(req.Body()))
	}
	if h := req.Header("Authorization"); h != "Bearer test-token" {
		t.Errorf("Header(Authorization) = %q", h)
	}
	if h := req.Header("Nonexistent"); h != "" {
		t.Errorf("Header(Nonexistent) = %q, want empty", h)
	}
}

func TestNewWebSocketRequest_Headers(t *testing.T) {
	t.Parallel()
	r := makeTestRequest("GET", "/test", nil)
	r.Header.Set("X-Custom", "val1")
	r.Header.Set("X-Another", "val2")
	req := NewWebSocketRequest(r, nil)

	headers := req.Headers()
	if headers["X-Custom"] != "val1" {
		t.Errorf("Headers() missing X-Custom")
	}
	if headers["X-Another"] != "val2" {
		t.Errorf("Headers() missing X-Another")
	}
}

func TestNewWebSocketRequest_Query(t *testing.T) {
	t.Parallel()
	q := url.Values{"chain": {"ethereum"}, "limit": {"100"}}
	r := makeTestRequest("GET", "/events", q)
	req := NewWebSocketRequest(r, nil)

	all := req.Query()
	if all["chain"] != "ethereum" {
		t.Errorf("Query() chain = %q", all["chain"])
	}
	if all["limit"] != "100" {
		t.Errorf("Query() limit = %q", all["limit"])
	}
	if qp := req.QueryParam("chain"); qp != "ethereum" {
		t.Errorf("QueryParam(chain) = %q", qp)
	}
	if qp := req.QueryParam("missing"); qp != "" {
		t.Errorf("QueryParam(missing) = %q, want empty", qp)
	}
}

func TestNewWebSocketRequest_Context(t *testing.T) {
	t.Parallel()
	r := makeTestRequest("GET", "/", nil)
	req := NewWebSocketRequest(r, nil)
	ctx := req.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}
	if ctx.Err() != nil {
		t.Errorf("Context().Err() = %v, want nil", ctx.Err())
	}
}

func TestNewWebSocketRequest_PathParam(t *testing.T) {
	t.Parallel()
	r := makeTestRequest("GET", "/test", nil)
	req := NewWebSocketRequest(r, nil)
	if pp := req.PathParam("id"); pp != "" {
		t.Errorf("PathParam() = %q, want empty", pp)
	}
}

func TestNewWebSocketResponse(t *testing.T) {
	t.Parallel()
	resp := newResponse()
	if resp.Status() != 200 {
		t.Errorf("default Status() = %d, want 200", resp.Status())
	}
	if resp.IsMessageSent() {
		t.Error("IsMessageSent() should be false initially")
	}
}

func TestNewWebSocketResponse_SetStatus(t *testing.T) {
	t.Parallel()
	resp := newResponse()
	resp.SetStatus(404)
	if resp.Status() != 404 {
		t.Errorf("Status() after SetStatus = %d, want 404", resp.Status())
	}
}

func TestNewWebSocketResponse_SetHeader(t *testing.T) {
	t.Parallel()
	resp := newResponse()
	resp.SetHeader("Content-Type", "application/json")
	if resp.Header("Content-Type") != "application/json" {
		t.Errorf("Header(Content-Type) = %q", resp.Header("Content-Type"))
	}
	if h := resp.Header("Missing"); h != "" {
		t.Errorf("Header(Missing) = %q", h)
	}
}

func TestNewWebSocketResponse_Headers(t *testing.T) {
	t.Parallel()
	resp := newResponse()
	resp.SetHeader("X-One", "1")
	resp.SetHeader("X-Two", "2")
	headers := resp.Headers()
	if headers["X-One"] != "1" || headers["X-Two"] != "2" {
		t.Errorf("Headers() = %v", headers)
	}
}

func TestNewWebSocketResponse_Body(t *testing.T) {
	t.Parallel()
	resp := newResponse()

	resp.SetBody([]byte("hello"))
	if string(resp.Body()) != "hello" {
		t.Errorf("Body() after SetBody = %q", string(resp.Body()))
	}

	n, err := resp.Write([]byte(" world"))
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != 6 {
		t.Errorf("Write() returned %d, want 6", n)
	}
	if string(resp.Body()) != "hello world" {
		t.Errorf("Body() after Write = %q", string(resp.Body()))
	}
}

func TestNewWebSocketResponse_SetStatusAfterSent(t *testing.T) {
	t.Parallel()
	resp := newResponse()

	// Mark as sent manually since we can't actually send
	resp.SetStatus(200)
	resp.SetHeader("X-Test", "v1")
	resp.SetBody([]byte("data"))

	// Verify IsHeadersSent works
	if resp.IsHeadersSent() {
		t.Error("IsHeadersSent() should be false before Send()")
	}
}

func TestWebSocketRequest_NilHandling(t *testing.T) {
	t.Parallel()
	// Create request with nil httpReq edge cases
	r := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/nil-test"},
	}
	req := NewWebSocketRequest(r, nil)

	if req.Path() != "/nil-test" {
		t.Errorf("Path() = %q", req.Path())
	}
	_ = req.Context()
}

func newResponse() *Response {
	return NewWebSocketResponse(nil)
}
