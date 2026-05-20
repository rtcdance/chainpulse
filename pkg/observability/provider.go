package observability

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ObservabilityProvider owns a shared TracerProvider and manages its lifecycle.
// It should be created once at application startup and passed to all components
// that need tracing, replacing the per-component TracerProvider creation pattern.
type ObservabilityProvider struct {
	tracerProvider *sdktrace.TracerProvider
	serviceName    string
	shutdownOnce   sync.Once
	shutdownErr    error
}

// ObservabilityConfig holds configuration for the shared observability provider.
type ObservabilityConfig struct {
	ServiceName  string
	OTLPEndpoint string // OTEL_EXPORTER_OTLP_ENDPOINT override
	SamplingRate float64
}

// NewObservabilityProvider creates a shared observability provider with a single
// TracerProvider. If no OTLP endpoint is configured, it creates a no-export provider
// (spans are recorded but not exported).
func NewObservabilityProvider(cfg ObservabilityConfig, logger core.Logger) (*ObservabilityProvider, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
		if cfg.ServiceName == "" {
			cfg.ServiceName = "chainpulse"
		}
	}
	if cfg.SamplingRate <= 0 {
		// Support OTEL_TRACES_SAMPLER_ARG env var (0.0-1.0)
		if rateStr := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); rateStr != "" {
			if rate, err := strconv.ParseFloat(rateStr, 64); err == nil && rate > 0 && rate <= 1.0 {
				cfg.SamplingRate = rate
			} else {
				cfg.SamplingRate = 1.0
			}
		} else {
			cfg.SamplingRate = 1.0
		}
	}

	// Use config endpoint, then fall back to env var
	otlpEndpoint := cfg.OTLPEndpoint
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	res := sdkresource.NewWithAttributes(
		"",
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.namespace", "chainpulse"),
	)

	var provider *sdktrace.TracerProvider

	if otlpEndpoint != "" {
		exporter, err := otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(otlpEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			if logger != nil {
				logger.Error("failed to create OTLP exporter, falling back to noop tracer", "error", err)
			}
			provider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
				sdktrace.WithResource(res),
			)
		} else {
			provider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
				sdktrace.WithResource(res),
				sdktrace.WithBatcher(exporter),
			)
		}
	} else {
		// No endpoint configured — spans are recorded but not exported
		provider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
			sdktrace.WithResource(res),
		)
	}

	// Set the global TracerProvider and propagator once
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &ObservabilityProvider{
		tracerProvider: provider,
		serviceName:    cfg.ServiceName,
	}, nil
}

// Tracer returns a named tracer from the shared TracerProvider.
func (op *ObservabilityProvider) Tracer(name string, options ...oteltrace.TracerOption) oteltrace.Tracer {
	return op.tracerProvider.Tracer(name, options...)
}

// Shutdown flushes pending spans and shuts down the TracerProvider.
// It is safe to call multiple times; subsequent calls are no-ops.
func (op *ObservabilityProvider) Shutdown(ctx context.Context) error {
	op.shutdownOnce.Do(func() {
		op.shutdownErr = op.tracerProvider.Shutdown(ctx)
	})
	return op.shutdownErr
}

// TracerProvider returns the underlying TracerProvider for advanced usage.
func (op *ObservabilityProvider) TracerProvider() *sdktrace.TracerProvider {
	return op.tracerProvider
}

// NewDefaultTracerWithProvider creates a DefaultTracer using the shared
// ObservabilityProvider instead of creating a new TracerProvider.
func NewDefaultTracerWithProvider(provider *ObservabilityProvider, logger core.Logger, metrics core.MetricsCollector) *DefaultTracer {
	if provider == nil {
		// Fallback to legacy behavior if no provider is given
		return NewDefaultTracer(logger, metrics)
	}

	return &DefaultTracer{
		spans:            make([]Span, 0),
		maxSpans:         10000,
		activeSpans:      make(map[string]*activeSpanState),
		traceIDCounter:   1,
		spanIDCounter:    1,
		logger:           logger,
		metricsCollector: metrics,
		otelProvider:     provider.TracerProvider(),
		otelTracer:       provider.Tracer(fmt.Sprintf("github.com/rtcdance/chainpulse/%s", provider.serviceName)),
	}
}
