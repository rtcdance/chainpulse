package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type activeSpanState struct {
	otelSpan oteltrace.Span
}

type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	wrote      bool
}

func newTracingResponseWriter(w http.ResponseWriter) *tracingResponseWriter {
	return &tracingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *tracingResponseWriter) WriteHeader(statusCode int) {
	if w.wrote {
		return
	}

	w.statusCode = statusCode
	w.wrote = true

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *tracingResponseWriter) Write(data []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

// WrapHTTPHandler adds an inbound server span around the provided handler.
func (t *DefaultTracer) WrapHTTPHandler(next http.Handler, operation string) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "handler unavailable", http.StatusInternalServerError)
		})
	}

	if operation == "" {
		operation = "http.request"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
		ctx, span := t.StartSpan(ctx, operation, SpanKindServer)
		t.SetAttribute(&span, "http.method", r.Method)
		t.SetAttribute(&span, "http.path", r.URL.Path)
		t.SetAttribute(&span, "http.target", r.URL.RequestURI())

		if r.Host != "" {
			t.SetAttribute(&span, "http.host", r.Host)
		}

		if ua := r.UserAgent(); ua != "" {
			t.SetAttribute(&span, "http.user_agent", ua)
		}

		cw := newTracingResponseWriter(w)
		next.ServeHTTP(cw, r.WithContext(ctx))

		switch {
		case cw.statusCode >= 500:
			t.SetStatus(&span, SpanStatusError, cw.statusCode, http.StatusText(cw.statusCode))
		case cw.statusCode >= 400:
			t.SetStatus(&span, SpanStatusError, cw.statusCode, http.StatusText(cw.statusCode))
		default:
			t.SetStatus(&span, SpanStatusOk, cw.statusCode, http.StatusText(cw.statusCode))
		}

		t.SetAttribute(&span, "http.status_code", cw.statusCode)

		if cw.statusCode >= 500 {
			if otelSpan := oteltrace.SpanFromContext(ctx); otelSpan != nil {
				otelSpan.RecordError(fmt.Errorf("http status %d", cw.statusCode))
			}
		}

		t.EndSpan(&span)
	})
}

// InjectTraceHeaders injects the current OTel trace context into outbound HTTP headers.
func InjectTraceHeaders(ctx context.Context, headers http.Header) {
	if headers == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

func convertSpanKind(kind SpanKind) oteltrace.SpanKind {
	switch kind {
	case SpanKindServer:
		return oteltrace.SpanKindServer
	case SpanKindClient:
		return oteltrace.SpanKindClient
	case SpanKindProducer:
		return oteltrace.SpanKindProducer
	case SpanKindConsumer:
		return oteltrace.SpanKindConsumer
	default:
		return oteltrace.SpanKindInternal
	}
}

func convertSpanAttributes(attributes map[string]any) []otelattribute.KeyValue {
	if len(attributes) == 0 {
		return nil
	}

	result := make([]otelattribute.KeyValue, 0, len(attributes))
	for key, value := range attributes {
		result = append(result, convertSpanAttribute(key, value))
	}

	return result
}

func convertSpanAttribute(key string, value any) otelattribute.KeyValue {
	switch v := value.(type) {
	case string:
		return otelattribute.String(key, v)
	case bool:
		return otelattribute.Bool(key, v)
	case int:
		return otelattribute.Int(key, v)
	case int8:
		return otelattribute.Int64(key, int64(v))
	case int16:
		return otelattribute.Int64(key, int64(v))
	case int32:
		return otelattribute.Int64(key, int64(v))
	case int64:
		return otelattribute.Int64(key, v)
	case uint:
		return otelattribute.String(key, fmt.Sprint(v))
	case uint8:
		return otelattribute.String(key, fmt.Sprint(v))
	case uint16:
		return otelattribute.String(key, fmt.Sprint(v))
	case uint32:
		return otelattribute.String(key, fmt.Sprint(v))
	case uint64:
		return otelattribute.String(key, fmt.Sprint(v))
	case float32:
		return otelattribute.Float64(key, float64(v))
	case float64:
		return otelattribute.Float64(key, v)
	default:
		return otelattribute.String(key, fmt.Sprint(value))
	}
}

func legacySpanContext(ctx TraceContext) (oteltrace.SpanContext, bool) {
	traceID, err := oteltrace.TraceIDFromHex(strings.TrimSpace(ctx.TraceID))
	if err != nil {
		return oteltrace.SpanContext{}, false
	}

	spanID, err := oteltrace.SpanIDFromHex(strings.TrimSpace(ctx.SpanID))
	if err != nil {
		return oteltrace.SpanContext{}, false
	}

	flags := oteltrace.TraceFlags(ctx.Flags)
	if flags == 0 && ctx.Flags&0x01 == 0x01 {
		flags = oteltrace.FlagsSampled
	}

	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	})

	return sc, sc.IsValid()
}

func spanContextFromIDs(traceID, spanID string) (oteltrace.SpanContext, bool) {
	tid, err := oteltrace.TraceIDFromHex(strings.TrimSpace(traceID))
	if err != nil {
		return oteltrace.SpanContext{}, false
	}

	sid, err := oteltrace.SpanIDFromHex(strings.TrimSpace(spanID))
	if err != nil {
		return oteltrace.SpanContext{}, false
	}

	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: oteltrace.FlagsSampled,
	})

	return sc, sc.IsValid()
}

func traceFlagsFromSpanContext(sc oteltrace.SpanContext) uint8 {
	if sc.TraceFlags().IsSampled() {
		return 0x01
	}

	return 0
}
