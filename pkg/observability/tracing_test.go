package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultTracer_generateTraceID(t *testing.T) {
	t.Parallel()
	tr := &DefaultTracer{}

	id1 := tr.generateTraceID()
	id2 := tr.generateTraceID()

	if id1 == id2 {
		t.Error("expected unique trace IDs")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-char trace ID, got %d", len(id1))
	}
}

func TestDefaultTracer_generateSpanID(t *testing.T) {
	t.Parallel()
	tr := &DefaultTracer{}

	id1 := tr.generateSpanID()
	id2 := tr.generateSpanID()

	if id1 == id2 {
		t.Error("expected unique span IDs")
	}
	if len(id1) != 16 {
		t.Errorf("expected 16-char span ID, got %d", len(id1))
	}
}

func TestTracingResponseWriter_Write(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	w := newTracingResponseWriter(rec)

	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if !w.wrote {
		t.Error("expected wrote=true after Write")
	}
}

func TestTracingResponseWriter_WriteHeader(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	w := newTracingResponseWriter(rec)

	w.WriteHeader(http.StatusNotFound)

	if w.statusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.statusCode)
	}
	if !w.wrote {
		t.Error("expected wrote=true after WriteHeader")
	}
}

func TestInjectTraceHeaders_nilHeaders(t *testing.T) {
	t.Parallel()
	InjectTraceHeaders(context.Background(), nil)
}

func TestInjectTraceHeaders_withHeaders(t *testing.T) {
	t.Parallel()
	headers := make(http.Header)
	InjectTraceHeaders(context.Background(), headers)
}