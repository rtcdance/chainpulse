package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestWrapHTTPHandlerCreatesSpanAndPropagatesContext(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	var sawSpan bool
	handler := tracer.WrapHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawSpan = oteltrace.SpanFromContext(r.Context()).SpanContext().IsValid()
		w.WriteHeader(http.StatusAccepted)
	}), "gateway.request")

	req := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.True(t, sawSpan)
	assert.Equal(t, http.StatusAccepted, rr.Code)

	spans := tracer.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "gateway.request", spans[0].Name)
	assert.Equal(t, SpanStatusOk, spans[0].Status)
	assert.Equal(t, http.StatusAccepted, spans[0].StatusCode)
	assert.Equal(t, "/events", spans[0].Attributes["http.path"])
	assert.Equal(t, "GET", spans[0].Attributes["http.method"])
}
