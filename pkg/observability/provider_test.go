package observability

import (
	"context"
	"testing"

	"chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObservabilityProviderNoEndpoint(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	provider, err := NewObservabilityProvider(ObservabilityConfig{
		ServiceName: "test-service",
	}, logger)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Should return a valid tracer
	tracer := provider.Tracer("test")
	assert.NotNil(t, tracer)

	// Shutdown should succeed
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestObservabilityProviderShutdownIdempotent(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	provider, err := NewObservabilityProvider(ObservabilityConfig{
		ServiceName: "test-service",
	}, logger)
	require.NoError(t, err)

	// Shutdown twice — second call should be a no-op
	err1 := provider.Shutdown(context.Background())
	err2 := provider.Shutdown(context.Background())
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestNewDefaultTracerWithProvider(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	provider, err := NewObservabilityProvider(ObservabilityConfig{
		ServiceName: "test-service",
	}, logger)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	tracer := NewDefaultTracerWithProvider(provider, logger, metrics)
	assert.NotNil(t, tracer)
}

func TestNewDefaultTracerWithNilProvider(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	// Nil provider should fall back to legacy behavior
	tracer := NewDefaultTracerWithProvider(nil, logger, metrics)
	assert.NotNil(t, tracer)
}

func TestObservabilityProviderDefaultServiceName(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	provider, err := NewObservabilityProvider(ObservabilityConfig{}, logger)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	assert.Equal(t, "chainpulse", provider.serviceName)
}
