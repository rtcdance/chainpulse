package core

import (
	"bytes"
	"io"
	"testing"
)

func TestNewBaseResponse(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	resp := NewBaseResponse(buf)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.IsHeadersSent() {
		t.Error("expected headers not sent")
	}
}

func TestNewBaseResponseWithNilWriter(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Should not panic
	resp.SetBody([]byte("test"))
}

func TestBaseResponseStatus(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	resp.SetStatus(201)
	if resp.Status() != 201 {
		t.Errorf("expected status 201, got %d", resp.Status())
	}

	resp.SetStatus(404)
	if resp.Status() != 404 {
		t.Errorf("expected status 404, got %d", resp.Status())
	}
}

func TestBaseResponseHeaders(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	resp.SetHeader("Content-Type", "application/json")
	resp.SetHeader("X-Custom-Header", "custom-value")

	if resp.Header("Content-Type") != "application/json" {
		t.Error("expected Content-Type header")
	}

	if resp.Header("X-Custom-Header") != "custom-value" {
		t.Error("expected X-Custom-Header")
	}

	headers := resp.Headers()
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
}

func TestBaseResponseBody(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	body := []byte(`{"status":"ok"}`)
	resp.SetBody(body)

	if string(resp.Body()) != string(body) {
		t.Errorf("expected body %s, got %s", string(body), string(resp.Body()))
	}
}

func TestBaseResponseWrite(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	n, err := resp.Write([]byte("hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	n, err = resp.Write([]byte(" world"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 6 {
		t.Errorf("expected 6 bytes written, got %d", n)
	}

	if string(resp.Body()) != "hello world" {
		t.Errorf("expected body 'hello world', got %s", string(resp.Body()))
	}
}

func TestBaseResponseSend(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	resp := NewBaseResponse(buf)

	resp.SetBody([]byte("test response"))
	err := resp.Send()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !resp.IsHeadersSent() {
		t.Error("expected headers sent flag to be true")
	}

	if buf.String() != "test response" {
		t.Errorf("expected 'test response', got %s", buf.String())
	}
}

func TestBaseResponseHeadersAfterSend(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	resp.SetStatus(200)
	resp.SetHeader("Content-Type", "application/json")
	_ = resp.Send()

	// Try to set headers after send - should not change
	resp.SetStatus(500)
	resp.SetHeader("X-New-Header", "value")

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.Header("X-New-Header") != "" {
		t.Error("expected header not to be set after send")
	}
}

func TestBaseResponseRuntimeMetricsStaged(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	metrics := resp.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "response-empty" {
		t.Fatalf("expected response-empty, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "response-staged" {
		t.Fatalf("expected response-staged, got %v", metrics["runtime_posture"])
	}
}

func TestBaseResponseRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)
	resp.SetHeader("Content-Type", "application/json")
	resp.SetBody([]byte(`{"ok":true}`))

	metrics := resp.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "response-complete" {
		t.Fatalf("expected response-complete, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "response-ready" {
		t.Fatalf("expected response-ready, got %v", metrics["runtime_posture"])
	}
}

func TestBaseResponseRuntimeMetricsSent(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)
	resp.SetBody([]byte("done"))
	if err := resp.Send(); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	metrics := resp.GetRuntimeMetrics()
	if metrics["runtime_posture"] != "response-sent" {
		t.Fatalf("expected response-sent, got %v", metrics["runtime_posture"])
	}
}

func TestBaseResponseImplementsInterface(t *testing.T) {
	t.Parallel()
	resp := NewBaseResponse(nil)

	// Verify it implements Response interface
	var _ Response = resp
}

func TestBaseResponseWithCustomWriter(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	resp := NewBaseResponse(buf)

	resp.SetStatus(201)
	resp.SetHeader("Location", "/api/users/1")
	resp.SetBody([]byte(`{"id":1}`))

	err := resp.Send()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if buf.String() != `{"id":1}` {
		t.Errorf("expected body in buffer, got %s", buf.String())
	}
}

func TestBaseResponseWriteError(t *testing.T) {
	t.Parallel()
	// Create a writer that returns an error
	failingWriter := &failingWriter{}
	resp := NewBaseResponse(failingWriter)

	resp.SetBody([]byte("test"))
	err := resp.Send()

	if err == nil {
		t.Error("expected error from failing writer")
	}
}

// failingWriter is a test helper that always fails
type failingWriter struct{}

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
