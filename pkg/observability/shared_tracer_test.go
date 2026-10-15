package observability

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestSharedTracerPreservesIncomingTraceAcrossComponents(t *testing.T) {
	t.Parallel()
	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	require.NoError(t, err)
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, Remote: true}))
	first := NewSharedTracer(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector())
	second := NewSharedTracer(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector())
	ctx, parent := first.StartSpan(ctx, "pull", SpanKindProducer)
	_, child := second.StartSpan(ctx, "consume", SpanKindConsumer)
	require.Equal(t, traceID.String(), parent.TraceID)
	require.Equal(t, parent.TraceID, child.TraceID)
	require.Equal(t, parent.SpanID, child.ParentID)
	first.EndSpan(&parent)
	second.EndSpan(&child)
}
