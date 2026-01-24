package grpc

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewGRPCRequest(t *testing.T) {
	ctx := context.Background()
	headers := map[string]string{"Authorization": "Bearer token"}
	body := []byte(`{"method":"Query","params":{}}`)

	req := NewGRPCRequest("POST", "/api.Service/Method", headers, body, ctx)

	if req == nil {
		t.Fatal("expected request, got nil")
	}

	if req.Method() != "POST" {
		t.Errorf("expected method POST, got %s", req.Method())
	}
}

func TestGRPCRequestMethod(t *testing.T) {
	methods := []string{"POST", "GET", "PUT"}

	for _, method := range methods {
		req := NewGRPCRequest(method, "/api.Service/Method", nil, []byte("{}"), context.Background())

		if req.Method() != method {
			t.Errorf("expected method %s, got %s", method, req.Method())
		}
	}
}

func TestGRPCRequestPath(t *testing.T) {
	paths := []string{"/api.Service/Method", "/api.Query/Execute", "/api.Admin/GetStatus"}

	for _, path := range paths {
		req := NewGRPCRequest("POST", path, nil, []byte("{}"), context.Background())

		if req.Path() != path {
			t.Errorf("expected path %s, got %s", path, req.Path())
		}
	}
}

func TestGRPCRequestHeaders(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"X-Request-ID":  "req-123",
	}

	req := NewGRPCRequest("POST", "/api.Service/Method", headers, []byte("{}"), context.Background())

	if req.Header("Authorization") != "Bearer token" {
		t.Error("expected Authorization header")
	}

	if req.Header("X-Request-ID") != "req-123" {
		t.Error("expected X-Request-ID header")
	}

	allHeaders := req.Headers()
	if len(allHeaders) != 2 {
		t.Errorf("expected 2 headers, got %d", len(allHeaders))
	}
}

func TestGRPCRequestBody(t *testing.T) {
	body := []byte(`{"method":"Query","params":{"id":123}}`)
	req := NewGRPCRequest("POST", "/api.Service/Method", nil, body, context.Background())

	if string(req.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(req.Body()))
	}
}

func TestGRPCRequestContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey, "user-123")
	req := NewGRPCRequest("POST", "/api.Service/Method", nil, []byte("{}"), ctx)

	if req.Context().Value(userIDKey) != "user-123" {
		t.Error("expected context value")
	}
}

func TestGRPCRequestQuery(t *testing.T) {
	req := NewGRPCRequest("POST", "/api.Service/Method", nil, []byte("{}"), context.Background())

	// gRPC doesn't use query parameters
	query := req.Query()
	if len(query) != 0 {
		t.Errorf("expected 0 query params, got %d", len(query))
	}

	if req.QueryParam("test") != "" {
		t.Error("expected empty query param")
	}
}

func TestNewGRPCResponse(t *testing.T) {
	resp := NewGRPCResponse()

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestGRPCResponseStatus(t *testing.T) {
	resp := NewGRPCResponse()

	resp.SetStatus(201)
	if resp.Status() != 201 {
		t.Errorf("expected status 201, got %d", resp.Status())
	}

	resp.SetStatus(400)
	if resp.Status() != 400 {
		t.Errorf("expected status 400, got %d", resp.Status())
	}
}

func TestGRPCResponseHeaders(t *testing.T) {
	resp := NewGRPCResponse()

	resp.SetHeader("X-Message-ID", "msg-123")
	resp.SetHeader("X-Timestamp", "2024-01-10")

	if resp.Header("X-Message-ID") != "msg-123" {
		t.Error("expected X-Message-ID header")
	}

	headers := resp.Headers()
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
}

func TestGRPCResponseBody(t *testing.T) {
	resp := NewGRPCResponse()

	body := []byte(`{"status":"ok","data":{"id":1}}`)
	resp.SetBody(body)

	if string(resp.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(resp.Body()))
	}
}

func TestGRPCResponseWrite(t *testing.T) {
	resp := NewGRPCResponse()

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

func TestGRPCResponseSend(t *testing.T) {
	resp := NewGRPCResponse()

	resp.SetStatus(201)
	resp.SetHeader("X-Message-ID", "msg-123")
	resp.SetBody([]byte(`{"id":1}`))

	err := resp.Send()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !resp.IsMessageSent() {
		t.Error("expected message to be sent")
	}
}

func TestGRPCResponseToJSON(t *testing.T) {
	resp := NewGRPCResponse()

	resp.SetStatus(200)
	resp.SetHeader("X-Message-ID", "msg-123")
	resp.SetBody([]byte(`{"data":"test"}`))

	data, err := resp.ToJSON()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var envelope map[string]interface{}
	err = json.Unmarshal(data, &envelope)
	if err != nil {
		t.Errorf("unexpected error unmarshaling: %v", err)
	}

	if envelope["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", envelope["status"])
	}
}

func TestGRPCResponseFromJSON(t *testing.T) {
	data := []byte(`{"status":201,"headers":{"X-Message-ID":"msg-123"},"body":"{\"id\":1}"}`)

	resp := NewGRPCResponse()
	err := resp.FromJSON(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Status() != 201 {
		t.Errorf("expected status 201, got %d", resp.Status())
	}

	if resp.Header("X-Message-ID") != "msg-123" {
		t.Error("expected X-Message-ID header")
	}

	if string(resp.Body()) != `{"id":1}` {
		t.Errorf("expected body, got %s", string(resp.Body()))
	}
}

func TestGRPCResponseHeaderImmutabilityAfterSend(t *testing.T) {
	resp := NewGRPCResponse()

	resp.SetStatus(200)
	resp.SetHeader("X-Original", "value")
	_ = resp.Send()

	// Try to modify after send
	resp.SetStatus(500)
	resp.SetHeader("X-New", "value")

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.Header("X-New") != "" {
		t.Error("expected header not to be set after send")
	}
}
