package observability

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// traceContextKey is the key for storing trace context in context.Context
	traceContextKey contextKey = "trace_context"
)

// SpanKind represents the kind of span
type SpanKind string

const (
	// SpanKindInternal represents an internal span.
	SpanKindInternal SpanKind = "INTERNAL"
	// SpanKindServer represents a server span.
	SpanKindServer SpanKind = "SERVER"
	// SpanKindClient represents a client span.
	SpanKindClient SpanKind = "CLIENT"
	// SpanKindProducer represents a producer span.
	SpanKindProducer SpanKind = "PRODUCER"
	// SpanKindConsumer represents a consumer span.
	SpanKindConsumer SpanKind = "CONSUMER"
)

// SpanStatus represents the status of a span
type SpanStatus string

const (
	// SpanStatusUnset represents an unset span status.
	SpanStatusUnset SpanStatus = "UNSET"
	// SpanStatusOk represents a successful span status.
	SpanStatusOk SpanStatus = "OK"
	// SpanStatusError represents an error span status.
	SpanStatusError SpanStatus = "ERROR"
)

// TraceContext represents the context for distributed tracing
type TraceContext struct {
	TraceID  string
	SpanID   string
	ParentID string
	Flags    uint8
	State    map[string]string
}

// Span represents a single span in a trace
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	Kind       SpanKind
	StartTime  time.Time
	EndTime    time.Time
	Status     SpanStatus
	StatusCode int
	StatusMsg  string
	Attributes map[string]interface{}
	Events     []SpanEvent
	Links      []SpanLink
	Duration   time.Duration
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]interface{}
}

// SpanLink represents a link to another span
type SpanLink struct {
	TraceID    string
	SpanID     string
	Attributes map[string]interface{}
}

// Tracer creates and manages spans
type Tracer interface {
	// StartSpan creates a new span
	StartSpan(ctx context.Context, name string, kind SpanKind) (context.Context, Span)

	// EndSpan ends a span
	EndSpan(span Span)

	// AddEvent adds an event to a span
	AddEvent(span *Span, name string, attributes map[string]interface{})

	// AddLink adds a link to a span
	AddLink(span *Span, traceID, spanID string, attributes map[string]interface{})

	// SetAttribute sets an attribute on a span
	SetAttribute(span *Span, key string, value interface{})

	// SetStatus sets the status of a span
	SetStatus(span *Span, status SpanStatus, code int, msg string)

	// ExtractContext extracts trace context from a carrier
	ExtractContext(carrier map[string]string) *TraceContext

	// InjectContext injects trace context into a carrier
	InjectContext(ctx *TraceContext, carrier map[string]string)

	// GetSpans returns all recorded spans
	GetSpans() []Span
}

// DefaultTracer implements the Tracer interface
type DefaultTracer struct {
	mu               sync.RWMutex
	spans            []Span
	maxSpans         int // cap on recorded spans to prevent unbounded growth
	activeSpans      map[string]*activeSpanState
	traceIDCounter   uint64
	spanIDCounter    uint64
	logger           core.Logger
	metricsCollector core.MetricsCollector
	otelProvider     *sdktrace.TracerProvider
	otelTracer       oteltrace.Tracer
}

// NewDefaultTracer creates a new tracer
func NewDefaultTracer(logger core.Logger, metrics core.MetricsCollector) *DefaultTracer {
	res := sdkresource.NewWithAttributes(
		"",
		otelattribute.String("service.name", "chainpulse"),
		otelattribute.String("service.namespace", "pkg.observability"),
	)

	var provider *sdktrace.TracerProvider

	// Configure OTLP exporter if endpoint is set
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint != "" {
		exporter, err := otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(otlpEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			// Fall back to noop provider if exporter fails
			if logger != nil {
				logger.Error("failed to create OTLP exporter, using noop tracer", "error", err)
			}
			provider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithResource(res),
			)
		} else {
			provider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithResource(res),
				sdktrace.WithBatcher(exporter),
			)
		}
	} else {
		// No endpoint configured, use in-process provider (spans are recorded but not exported)
		provider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
		)
	}

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &DefaultTracer{
		spans:            make([]Span, 0),
		maxSpans:         10000,
		activeSpans:      make(map[string]*activeSpanState),
		traceIDCounter:   1,
		spanIDCounter:    1,
		logger:           logger,
		metricsCollector: metrics,
		otelProvider:     provider,
		otelTracer:       provider.Tracer("chainpulse/pkg.observability"),
	}
}

// StartSpan creates a new span
func (t *DefaultTracer) StartSpan(ctx context.Context, name string, kind SpanKind) (context.Context, Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	if legacyCtx, ok := ctx.Value(traceContextKey).(TraceContext); ok && legacyCtx.TraceID != "" && legacyCtx.SpanID != "" {
		if legacyParent, ok := legacySpanContext(legacyCtx); ok {
			ctx = oteltrace.ContextWithRemoteSpanContext(ctx, legacyParent)
		}
	}

	startedCtx, otelSpan := t.otelTracer.Start(ctx, name, oteltrace.WithSpanKind(convertSpanKind(kind)))
	spanCtx := otelSpan.SpanContext()
	traceID := spanCtx.TraceID().String()

	if traceID == "" || !spanCtx.IsValid() {
		traceID = t.generateTraceID()
	}

	spanID := spanCtx.SpanID().String()
	if spanID == "" {
		spanID = t.generateSpanID()
	}

	var parentID string

	if parentSpan := oteltrace.SpanFromContext(ctx); parentSpan != nil {
		parentSC := parentSpan.SpanContext()
		if parentSC.IsValid() {
			parentID = parentSC.SpanID().String()
		}
	}

	span := Span{
		TraceID:    traceID,
		SpanID:     spanID,
		ParentID:   parentID,
		Name:       name,
		Kind:       kind,
		StartTime:  time.Now().UTC(),
		Status:     SpanStatusUnset,
		Attributes: make(map[string]interface{}),
		Events:     make([]SpanEvent, 0),
		Links:      make([]SpanLink, 0),
	}

	// Store active span
	t.activeSpans[spanID] = &activeSpanState{otelSpan: otelSpan}

	// Create new context with trace context
	newCtx := context.WithValue(startedCtx, traceContextKey, TraceContext{
		TraceID:  traceID,
		SpanID:   spanID,
		ParentID: parentID,
		Flags:    traceFlagsFromSpanContext(spanCtx),
	})

	// Record metric
	if t.metricsCollector != nil {
		t.metricsCollector.RecordCounter("span_started", 1, map[string]string{
			"span_kind": string(kind),
			"span_name": name,
		})
	}

	// Log span start
	if t.logger != nil {
		t.logger.WithCorrelationID(traceID).Debug("span started", "span_id", spanID, "span_name", name, "span_kind", kind)
	}

	return newCtx, span
}

// EndSpan ends a span
func (t *DefaultTracer) EndSpan(span *Span) {
	if span == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.EndTime = time.Now().UTC()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if activeState, ok := t.activeSpans[span.SpanID]; ok && activeState != nil && activeState.otelSpan != nil {
		activeState.otelSpan.SetAttributes(
			otelattribute.String("chainpulse.span.kind", string(span.Kind)),
			otelattribute.String("chainpulse.span.status", string(span.Status)),
			otelattribute.Int("chainpulse.span.status_code", span.StatusCode),
			otelattribute.String("chainpulse.span.status_message", span.StatusMsg),
		)

		switch span.Status {
		case SpanStatusError:
			activeState.otelSpan.SetStatus(otelcodes.Error, span.StatusMsg)
		case SpanStatusOk:
			activeState.otelSpan.SetStatus(otelcodes.Ok, span.StatusMsg)
		default:
			activeState.otelSpan.SetStatus(otelcodes.Unset, span.StatusMsg)
		}

		activeState.otelSpan.End()
	}

	// Remove from active spans
	delete(t.activeSpans, span.SpanID)

	// Add to recorded spans with bounded capacity
	if t.maxSpans > 0 && len(t.spans) >= t.maxSpans {
		// Evict oldest half to amortize the cost
		half := t.maxSpans / 2
		if half < len(t.spans) {
			copy(t.spans, t.spans[half:])
			t.spans = t.spans[:len(t.spans)-half]
		}
	}
	t.spans = append(t.spans, *span)

	// Record metric
	if t.metricsCollector != nil {
		t.metricsCollector.RecordHistogram("span_duration_ms", float64(span.Duration.Milliseconds()), map[string]string{
			"span_kind": string(span.Kind),
			"span_name": span.Name,
			"status":    string(span.Status),
		})
	}

	// Log span end
	if t.logger != nil {
		t.logger.WithCorrelationID(span.TraceID).Debug("span ended", "span_id", span.SpanID, "span_name", span.Name, "duration_ms", span.Duration.Milliseconds())
	}
}

// AddEvent adds an event to a span
func (t *DefaultTracer) AddEvent(span *Span, name string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span == nil {
		return
	}

	event := SpanEvent{
		Name:       name,
		Timestamp:  time.Now().UTC(),
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)

	if activeState, ok := t.activeSpans[span.SpanID]; ok && activeState != nil && activeState.otelSpan != nil {
		activeState.otelSpan.AddEvent(name, oteltrace.WithAttributes(convertSpanAttributes(attributes)...))
	}

	// Log event
	if t.logger != nil {
		t.logger.WithCorrelationID(span.TraceID).Debug("span event added", "span_id", span.SpanID, "event_name", name)
	}
}

// AddLink adds a link to a span
func (t *DefaultTracer) AddLink(span *Span, traceID, spanID string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span == nil {
		return
	}

	link := SpanLink{
		TraceID:    traceID,
		SpanID:     spanID,
		Attributes: attributes,
	}

	span.Links = append(span.Links, link)

	if activeState, ok := t.activeSpans[span.SpanID]; ok && activeState != nil && activeState.otelSpan != nil {
		if linkCtx, ok := spanContextFromIDs(traceID, spanID); ok {
			activeState.otelSpan.AddLink(oteltrace.Link{
				SpanContext: linkCtx,
				Attributes:  convertSpanAttributes(attributes),
			})
		}
	}

	// Log link
	if t.logger != nil {
		t.logger.WithCorrelationID(span.TraceID).Debug("span link added", "span_id", span.SpanID, "linked_trace_id", traceID, "linked_span_id", spanID)
	}
}

// SetAttribute sets an attribute on a span
func (t *DefaultTracer) SetAttribute(span *Span, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span == nil {
		return
	}

	span.Attributes[key] = value

	if activeState, ok := t.activeSpans[span.SpanID]; ok && activeState != nil && activeState.otelSpan != nil {
		activeState.otelSpan.SetAttributes(convertSpanAttribute(key, value))
	}
}

// SetStatus sets the status of a span
func (t *DefaultTracer) SetStatus(span *Span, status SpanStatus, code int, msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if span == nil {
		return
	}

	span.Status = status
	span.StatusCode = code
	span.StatusMsg = msg

	if activeState, ok := t.activeSpans[span.SpanID]; ok && activeState != nil && activeState.otelSpan != nil {
		switch status {
		case SpanStatusError:
			activeState.otelSpan.SetStatus(otelcodes.Error, msg)
		case SpanStatusOk:
			activeState.otelSpan.SetStatus(otelcodes.Ok, msg)
		default:
			activeState.otelSpan.SetStatus(otelcodes.Unset, msg)
		}
	}

	// Log status change
	if t.logger != nil {
		t.logger.WithCorrelationID(span.TraceID).Debug("span status set", "span_id", span.SpanID, "status", status, "code", code, "message", msg)
	}
}

// ExtractContext extracts trace context from a carrier
func (t *DefaultTracer) ExtractContext(carrier map[string]string) *TraceContext {
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(carrier))
	sc := oteltrace.SpanContextFromContext(ctx)

	if !sc.IsValid() {
		return &TraceContext{State: make(map[string]string)}
	}

	result := &TraceContext{
		TraceID: sc.TraceID().String(),
		SpanID:  sc.SpanID().String(),
		Flags:   traceFlagsFromSpanContext(sc),
		State:   make(map[string]string),
	}
	if ts := sc.TraceState().String(); ts != "" {
		result.State["tracestate"] = ts
	}

	return result
}

// InjectContext injects trace context into a carrier
func (t *DefaultTracer) InjectContext(ctx *TraceContext, carrier map[string]string) {
	if carrier == nil || ctx == nil {
		return
	}

	sc, ok := legacySpanContext(*ctx)
	if !ok {
		return
	}

	otelCtx := oteltrace.ContextWithSpanContext(context.Background(), sc)
	otel.GetTextMapPropagator().Inject(otelCtx, propagation.MapCarrier(carrier))

	if traceState, ok := ctx.State["tracestate"]; ok && traceState != "" {
		carrier["tracestate"] = traceState
	}
}

// GetSpans returns all recorded spans
func (t *DefaultTracer) GetSpans() []Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to prevent external modification
	spans := make([]Span, len(t.spans))
	copy(spans, t.spans)
	return spans
}

// generateTraceID generates a unique trace ID
func (t *DefaultTracer) generateTraceID() string {
	t.traceIDCounter++
	return fmt.Sprintf("%032x", t.traceIDCounter)
}

// generateSpanID generates a unique span ID
func (t *DefaultTracer) generateSpanID() string {
	t.spanIDCounter++
	return fmt.Sprintf("%016x", t.spanIDCounter)
}

// parseTraceParent parses W3C Trace Context traceparent header
//
//nolint:unused
func parseTraceParent(traceparent string) []string {
	// Format: version-traceID-parentID-flags
	// Example: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
	parts := make([]string, 0)
	var current string
	for _, ch := range traceparent {
		if ch == '-' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// TracingContext represents the context for tracing operations
type TracingContext struct {
	TraceID       string
	SpanID        string
	ParentID      string
	CorrelationID string
}

// SpanRecorder records spans for testing and analysis
type SpanRecorder struct {
	mu       sync.RWMutex
	spans    []Span
	maxSpans int // cap to prevent unbounded growth
}

// NewSpanRecorder creates a new span recorder
func NewSpanRecorder() *SpanRecorder {
	return &SpanRecorder{
		spans:    make([]Span, 0),
		maxSpans: 10000,
	}
}

// Record records a span
func (sr *SpanRecorder) Record(span Span) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if sr.maxSpans > 0 && len(sr.spans) >= sr.maxSpans {
		half := sr.maxSpans / 2
		if half < len(sr.spans) {
			copy(sr.spans, sr.spans[half:])
			sr.spans = sr.spans[:len(sr.spans)-half]
		}
	}
	sr.spans = append(sr.spans, span)
}

// GetRecordedSpans returns all recorded spans
func (sr *SpanRecorder) GetRecordedSpans() []Span {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	spans := make([]Span, len(sr.spans))
	copy(spans, sr.spans)
	return spans
}

// Clear clears all recorded spans
func (sr *SpanRecorder) Clear() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.spans = make([]Span, 0)
}

// GetSpanCount returns the number of recorded spans
func (sr *SpanRecorder) GetSpanCount() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return len(sr.spans)
}

// GetSpansByTraceID returns all spans for a given trace ID
func (sr *SpanRecorder) GetSpansByTraceID(traceID string) []Span {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make([]Span, 0)
	for _, span := range sr.spans {
		if span.TraceID == traceID {
			result = append(result, span)
		}
	}
	return result
}

// GetSpansByName returns all spans with a given name
func (sr *SpanRecorder) GetSpansByName(name string) []Span {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make([]Span, 0)
	for _, span := range sr.spans {
		if span.Name == name {
			result = append(result, span)
		}
	}
	return result
}
