package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
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
	SpanKindServer   SpanKind = "SERVER"
	// SpanKindClient represents a client span.
	SpanKindClient   SpanKind = "CLIENT"
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
	SpanStatusOk    SpanStatus = "OK"
	// SpanStatusError represents an error span status.
	SpanStatusError SpanStatus = "ERROR"
)

// TraceContext represents the context for distributed tracing
type TraceContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Flags      uint8
	State      map[string]string
}

// Span represents a single span in a trace
type Span struct {
	TraceID      string
	SpanID       string
	ParentID     string
	Name         string
	Kind         SpanKind
	StartTime    time.Time
	EndTime      time.Time
	Status       SpanStatus
	StatusCode   int
	StatusMsg    string
	Attributes   map[string]interface{}
	Events       []SpanEvent
	Links        []SpanLink
	Duration     time.Duration
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
	ExtractContext(carrier map[string]string) TraceContext

	// InjectContext injects trace context into a carrier
	InjectContext(ctx TraceContext, carrier map[string]string)

	// GetSpans returns all recorded spans
	GetSpans() []Span
}

// DefaultTracer implements the Tracer interface
type DefaultTracer struct {
	mu              sync.RWMutex
	spans           []Span
	activeSpans     map[string]*Span
	traceIDCounter  uint64
	spanIDCounter   uint64
	logger          core.Logger
	metricsCollector core.MetricsCollector
}

// NewDefaultTracer creates a new tracer
func NewDefaultTracer(logger core.Logger, metrics core.MetricsCollector) *DefaultTracer {
	return &DefaultTracer{
		spans:            make([]Span, 0),
		activeSpans:      make(map[string]*Span),
		traceIDCounter:   1,
		spanIDCounter:    1,
		logger:           logger,
		metricsCollector: metrics,
	}
}

// StartSpan creates a new span
func (t *DefaultTracer) StartSpan(ctx context.Context, name string, kind SpanKind) (context.Context, Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Extract parent context if available
	var parentID string
	var traceID string
	if parentCtx, ok := ctx.Value(traceContextKey).(TraceContext); ok {
		parentID = parentCtx.SpanID
		traceID = parentCtx.TraceID
	} else {
		traceID = t.generateTraceID()
	}

	spanID := t.generateSpanID()

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
	t.activeSpans[spanID] = &span

	// Create new context with trace context
	newCtx := context.WithValue(ctx, traceContextKey, TraceContext{
		TraceID:  traceID,
		SpanID:   spanID,
		ParentID: parentID,
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
	t.mu.Lock()
	defer t.mu.Unlock()

	span.EndTime = time.Now().UTC()
	span.Duration = span.EndTime.Sub(span.StartTime)

	// Remove from active spans
	delete(t.activeSpans, span.SpanID)

	// Add to recorded spans
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

	// Log status change
	if t.logger != nil {
		t.logger.WithCorrelationID(span.TraceID).Debug("span status set", "span_id", span.SpanID, "status", status, "code", code, "message", msg)
	}
}

// ExtractContext extracts trace context from a carrier
func (t *DefaultTracer) ExtractContext(carrier map[string]string) TraceContext {
	ctx := TraceContext{
		State: make(map[string]string),
	}

	if traceID, ok := carrier["traceparent"]; ok {
		// Parse W3C Trace Context format: version-traceID-parentID-flags
		parts := parseTraceParent(traceID)
		if len(parts) >= 3 {
			ctx.TraceID = parts[1]
			ctx.SpanID = parts[2]
		}
	}

	if traceState, ok := carrier["tracestate"]; ok {
		ctx.State["tracestate"] = traceState
	}

	return ctx
}

// InjectContext injects trace context into a carrier
func (t *DefaultTracer) InjectContext(ctx TraceContext, carrier map[string]string) {
	if carrier == nil {
		return
	}

	// W3C Trace Context format: version-traceID-parentID-flags
	traceParent := fmt.Sprintf("00-%s-%s-%02x", ctx.TraceID, ctx.SpanID, ctx.Flags)
	carrier["traceparent"] = traceParent

	if traceState, ok := ctx.State["tracestate"]; ok {
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
	TraceID      string
	SpanID       string
	ParentID     string
	CorrelationID string
}

// SpanRecorder records spans for testing and analysis
type SpanRecorder struct {
	mu    sync.RWMutex
	spans []Span
}

// NewSpanRecorder creates a new span recorder
func NewSpanRecorder() *SpanRecorder {
	return &SpanRecorder{
		spans: make([]Span, 0),
	}
}

// Record records a span
func (sr *SpanRecorder) Record(span Span) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
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
