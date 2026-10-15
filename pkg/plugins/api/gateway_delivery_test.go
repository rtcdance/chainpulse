package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/require"
)

type deliveryProbe struct{}

func (*deliveryProbe) Initialize(*core.Config) error { return nil }
func (*deliveryProbe) Stop() error                   { return nil }
func (*deliveryProbe) Handle(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

func TestGatewayRegisteredHandlerIsActuallyDispatched(t *testing.T) {
	t.Parallel()
	g := NewAPIGatewayPlugin(&MockLogger{}, NewMockMetricsCollector())
	g.initialized = true
	require.NoError(t, g.RegisterHandler("/readyz", &deliveryProbe{}))
	w := httptest.NewRecorder()
	g.withRegisteredHandlers(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReadinessFailsWhenIndexingCannotProgress(t *testing.T) {
	t.Parallel()
	h := NewHealthCheckHandler(&mockHealthDatabaseManager{}, &MockLogger{}, NewMockMetricsCollector())
	require.NoError(t, h.Initialize(context.Background()))
	h.SetReadinessCheck(func(context.Context) bool { return false })
	w := httptest.NewRecorder()
	h.HandleReady(w, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
