package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"chainpulse/pkg/core"
)

// MockLogger for testing
type MockLogger struct {
	messages []string
}

func (ml *MockLogger) Debug(msg string, args ...any) { ml.messages = append(ml.messages, msg) }

func (ml *MockLogger) Info(msg string, args ...any) { ml.messages = append(ml.messages, msg) }

func (ml *MockLogger) Warn(msg string, args ...any) { ml.messages = append(ml.messages, msg) }

func (ml *MockLogger) Error(msg string, args ...any) { ml.messages = append(ml.messages, msg) }

func (ml *MockLogger) Fatal(msg string, args ...any) { ml.messages = append(ml.messages, msg) }

func (ml *MockLogger) WithCorrelationID(id string) core.Logger {
	return ml
}

// MockMetricsCollector for testing
type MockMetricsCollector struct {
	counters   map[string]int64
	histograms map[string][]float64
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
	}
}

func (mmc *MockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	mmc.counters[name] += value
}

func (mmc *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	mmc.histograms[name] = append(mmc.histograms[name], value)
}

func (mmc *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {}

func (mmc *MockMetricsCollector) GetMetrics() map[string]any {
	return map[string]any{
		"counters":   mmc.counters,
		"histograms": mmc.histograms,
	}
}

func (mmc *MockMetricsCollector) Close() error { return nil }

func TestNewDefaultTracer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	tracer := NewDefaultTracer(logger, metrics)

	assert.NotNil(t, tracer)
	assert.Equal(t, 0, len(tracer.GetSpans()))
}

func TestStartSpan(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	ctx, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	assert.NotNil(t, ctx)
	assert.NotEmpty(t, span.TraceID)
	assert.NotEmpty(t, span.SpanID)
	assert.Equal(t, "test_span", span.Name)
	assert.Equal(t, SpanKindInternal, span.Kind)
	assert.Equal(t, SpanStatusUnset, span.Status)
}

func TestEndSpan(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	// Add a small delay to ensure measurable duration
	time.Sleep(1 * time.Millisecond)

	tracer.EndSpan(&span)

	spans := tracer.GetSpans()
	assert.Equal(t, 1, len(spans))
	assert.NotZero(t, spans[0].Duration)
}

func TestAddEvent(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	tracer.AddEvent(&span, "test_event", map[string]any{"key": "value"})

	assert.Equal(t, 1, len(span.Events))
	assert.Equal(t, "test_event", span.Events[0].Name)
}

func TestAddLink(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	tracer.AddLink(&span, "trace123", "span456", map[string]any{"key": "value"})

	assert.Equal(t, 1, len(span.Links))
	assert.Equal(t, "trace123", span.Links[0].TraceID)
	assert.Equal(t, "span456", span.Links[0].SpanID)
}

func TestSetAttribute(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	tracer.SetAttribute(&span, "key1", "value1")
	tracer.SetAttribute(&span, "key2", 42)

	assert.Equal(t, "value1", span.Attributes["key1"])
	assert.Equal(t, 42, span.Attributes["key2"])
}

func TestSetStatus(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	tracer.SetStatus(&span, SpanStatusOk, 200, "success")

	assert.Equal(t, SpanStatusOk, span.Status)
	assert.Equal(t, 200, span.StatusCode)
	assert.Equal(t, "success", span.StatusMsg)
}

func TestSetStatusError(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	tracer.SetStatus(&span, SpanStatusError, 500, "internal error")

	assert.Equal(t, SpanStatusError, span.Status)
	assert.Equal(t, 500, span.StatusCode)
}

func TestExtractContext(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	carrier := map[string]string{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
	}

	ctx := tracer.ExtractContext(carrier)

	assert.NotEmpty(t, ctx.TraceID)
	assert.NotEmpty(t, ctx.SpanID)
}

func TestInjectContext(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	ctx := &TraceContext{
		TraceID: "0af7651916cd43dd8448eb211c80319c",
		SpanID:  "b7ad6b7169203331",
		Flags:   0x01,
	}

	carrier := make(map[string]string)
	tracer.InjectContext(ctx, carrier)

	assert.NotEmpty(t, carrier["traceparent"])
	assert.Contains(t, carrier["traceparent"], "0af7651916cd43dd8448eb211c80319c")
}

func TestGetSpans(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span1 := tracer.StartSpan(context.Background(), "span1", SpanKindInternal)
	tracer.EndSpan(&span1)

	_, span2 := tracer.StartSpan(context.Background(), "span2", SpanKindServer)
	tracer.EndSpan(&span2)

	spans := tracer.GetSpans()

	assert.Equal(t, 2, len(spans))
}

func TestNestedSpans(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	ctx1, span1 := tracer.StartSpan(context.Background(), "parent", SpanKindInternal)
	_, span2 := tracer.StartSpan(ctx1, "child", SpanKindInternal)

	assert.Equal(t, span1.TraceID, span2.TraceID)
	assert.Equal(t, span1.SpanID, span2.ParentID)
	assert.NotEqual(t, span1.SpanID, span2.SpanID)

	tracer.EndSpan(&span2)
	tracer.EndSpan(&span1)

	spans := tracer.GetSpans()
	assert.Equal(t, 2, len(spans))
}

func TestSpanDuration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)

	time.Sleep(10 * time.Millisecond)

	tracer.EndSpan(&span)

	assert.Greater(t, span.Duration, 0*time.Millisecond)
	assert.GreaterOrEqual(t, span.Duration, 5*time.Millisecond)
}

func TestSpanKinds(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	kinds := []SpanKind{
		SpanKindInternal,
		SpanKindServer,
		SpanKindClient,
		SpanKindProducer,
		SpanKindConsumer,
	}

	for _, kind := range kinds {
		_, span := tracer.StartSpan(context.Background(), "test", kind)
		assert.Equal(t, kind, span.Kind)
		tracer.EndSpan(&span)
	}

	spans := tracer.GetSpans()
	assert.Equal(t, 5, len(spans))
}

func TestNewSpanRecorder(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	assert.NotNil(t, recorder)
	assert.Equal(t, 0, recorder.GetSpanCount())
}

func TestSpanRecorderRecord(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	span := Span{
		TraceID: "trace1",
		SpanID:  "span1",
		Name:    "test_span",
	}

	recorder.Record(span)

	assert.Equal(t, 1, recorder.GetSpanCount())
}

func TestSpanRecorderGetRecordedSpans(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	span1 := Span{TraceID: "trace1", SpanID: "span1", Name: "span1"}
	span2 := Span{TraceID: "trace1", SpanID: "span2", Name: "span2"}

	recorder.Record(span1)
	recorder.Record(span2)

	spans := recorder.GetRecordedSpans()

	assert.Equal(t, 2, len(spans))
}

func TestSpanRecorderClear(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	span := Span{TraceID: "trace1", SpanID: "span1"}
	recorder.Record(span)

	assert.Equal(t, 1, recorder.GetSpanCount())

	recorder.Clear()

	assert.Equal(t, 0, recorder.GetSpanCount())
}

func TestSpanRecorderGetSpansByTraceID(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	span1 := Span{TraceID: "trace1", SpanID: "span1"}
	span2 := Span{TraceID: "trace1", SpanID: "span2"}
	span3 := Span{TraceID: "trace2", SpanID: "span3"}

	recorder.Record(span1)
	recorder.Record(span2)
	recorder.Record(span3)

	trace1Spans := recorder.GetSpansByTraceID("trace1")

	assert.Equal(t, 2, len(trace1Spans))
}

func TestSpanRecorderGetSpansByName(t *testing.T) {
	t.Parallel()
	recorder := NewSpanRecorder()

	span1 := Span{TraceID: "trace1", SpanID: "span1", Name: "operation1"}
	span2 := Span{TraceID: "trace1", SpanID: "span2", Name: "operation2"}
	span3 := Span{TraceID: "trace2", SpanID: "span3", Name: "operation1"}

	recorder.Record(span1)
	recorder.Record(span2)
	recorder.Record(span3)

	operation1Spans := recorder.GetSpansByName("operation1")

	assert.Equal(t, 2, len(operation1Spans))
}

func TestSpanEventStructure(t *testing.T) {
	t.Parallel()
	event := SpanEvent{
		Name:       "test_event",
		Timestamp:  time.Now(),
		Attributes: map[string]any{"key": "value"},
	}

	assert.Equal(t, "test_event", event.Name)
	assert.NotNil(t, event.Timestamp)
	assert.Equal(t, "value", event.Attributes["key"])
}

func TestSpanLinkStructure(t *testing.T) {
	t.Parallel()
	link := SpanLink{
		TraceID:    "trace123",
		SpanID:     "span456",
		Attributes: map[string]any{"key": "value"},
	}

	assert.Equal(t, "trace123", link.TraceID)
	assert.Equal(t, "span456", link.SpanID)
}

func TestTraceContextStructure(t *testing.T) {
	t.Parallel()
	ctx := TraceContext{
		TraceID:  "trace123",
		SpanID:   "span456",
		ParentID: "parent789",
		Flags:    0x01,
		State:    make(map[string]string),
	}

	assert.Equal(t, "trace123", ctx.TraceID)
	assert.Equal(t, "span456", ctx.SpanID)
	assert.Equal(t, "parent789", ctx.ParentID)
	assert.Equal(t, uint8(0x01), ctx.Flags)
}

func TestMetricsRecording(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	_, span := tracer.StartSpan(context.Background(), "test_span", SpanKindInternal)
	tracer.EndSpan(&span)

	assert.Greater(t, metrics.counters["span_started"], int64(0))
	assert.Greater(t, len(metrics.histograms["span_duration_ms"]), 0)
}

func TestConcurrentSpans(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tracer := NewDefaultTracer(logger, metrics)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			_, span := tracer.StartSpan(context.Background(), "span", SpanKindInternal)
			time.Sleep(1 * time.Millisecond)
			tracer.EndSpan(&span)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	spans := tracer.GetSpans()
	assert.Equal(t, 10, len(spans))
}
