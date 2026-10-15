package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHealthProbesDoNotConsumeOrDependOnAPIQuota(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/health", "/health/ready", "/health/live", "/readyz", "/livez"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			limiter := NewRateLimiter(&MockLogger{}, NewMockMetricsCollector(), &RateLimitConfig{DefaultRequestsPerSecond: 0.000001, DefaultBurstSize: 1, CleanupInterval: time.Hour})
			middleware := NewRateLimitMiddleware(limiter, &MockLogger{})
			handler := middleware.Middleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == path {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			request := func(method, target string) int {
				r := httptest.NewRequest(method, target, nil)
				r.RemoteAddr = "192.0.2.1:1234"
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				return w.Code
			}
			for range 100 {
				require.Equal(t, 503, request(http.MethodGet, path), "probe must preserve actual unhealthy status without spending API tokens")
			}
			require.Equal(t, 200, request(http.MethodGet, "/events"), "probes must not spend API quota")
			require.Equal(t, 429, request(http.MethodGet, "/events"))
			require.Equal(t, 503, request(http.MethodGet, path), "exhausted business quota must not break probes")
			for _, limited := range []struct{ method, path string }{{http.MethodPost, path}, {http.MethodGet, path + "/admin"}, {http.MethodGet, "/health/components"}, {http.MethodGet, "/runtime/control"}} {
				require.Equal(t, 429, request(limited.method, limited.path), "only exact read-only probe routes are exempt")
			}
		})
	}
}
