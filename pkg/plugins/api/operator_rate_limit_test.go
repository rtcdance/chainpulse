package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOperatorRateLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		want    map[string]int
		wantErr bool
	}{
		{name: "empty spec means no overrides", spec: "", want: map[string]int{}},
		{name: "single operator", spec: "alice=60", want: map[string]int{"alice": 60}},
		{
			name: "multiple operators with whitespace",
			spec: " alice = 60 , bob=120 ",
			want: map[string]int{"alice": 60, "bob": 120},
		},
		{name: "trailing comma tolerated", spec: "alice=60,", want: map[string]int{"alice": 60}},
		{name: "missing value", spec: "alice", wantErr: true},
		{name: "zero budget rejected", spec: "alice=0", wantErr: true},
		{name: "negative budget rejected", spec: "alice=-5", wantErr: true},
		{name: "non-integer budget", spec: "alice=fast", wantErr: true},
		{name: "empty operator name", spec: "=60", wantErr: true},
		{name: "duplicate operator", spec: "alice=60,alice=70", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseOperatorRateLimits(tt.spec)
			if tt.wantErr {
				require.Error(t, err, "spec %q must be rejected", tt.spec)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewRateLimitConfigFromPerMinute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		defaultPerMin int
		operators     map[string]int
		wantErr       bool
		wantOperators int
	}{
		{name: "defaults only", defaultPerMin: 60, wantOperators: 0},
		{name: "operator quotas merged", defaultPerMin: 60, operators: map[string]int{"alice": 120, "bob": 30}, wantOperators: 2},
		{name: "zero default rejected", defaultPerMin: 0, wantErr: true},
		{name: "negative default rejected", defaultPerMin: -1, wantErr: true},
		{name: "zero operator budget rejected", defaultPerMin: 60, operators: map[string]int{"alice": 0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := NewRateLimitConfigFromPerMinute(tt.defaultPerMin, tt.operators)
			if tt.wantErr {
				require.Error(t, err, "unlimited budgets must be rejected, not silently accepted")
				return
			}
			require.NoError(t, err)

			assert.Greater(t, cfg.DefaultRequestsPerSecond, 0.0, "default refill rate must be finite and positive")
			assert.GreaterOrEqual(t, cfg.DefaultBurstSize, 1, "default burst must be at least 1")
			require.Len(t, cfg.ClientLimits, tt.wantOperators)
			for operator, limit := range cfg.ClientLimits {
				assert.Greater(t, limit.RequestsPerSecond, 0.0, "operator %q refill rate must be finite and positive", operator)
				assert.GreaterOrEqual(t, limit.BurstSize, 1, "operator %q burst must be at least 1", operator)
			}
		})
	}
}

func TestExtractClientID_UsesAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyClientID, "0xOperatorSubject"))
	assert.Equal(t, "0xOperatorSubject", extractClientID(req),
		"per-operator quotas key off the identity AuthMiddleware stores")

	legacy := httptest.NewRequest(http.MethodGet, "/events", nil)
	//lint:ignore SA1029 This case verifies compatibility with the historical plain-string context key.
	legacy = legacy.WithContext(context.WithValue(legacy.Context(), "clientID", "legacy-client"))
	assert.Equal(t, "legacy-client", extractClientID(legacy),
		"the legacy plain-string key must keep working")

	anonymous := httptest.NewRequest(http.MethodGet, "/events", nil)
	assert.Equal(t, "", extractClientID(anonymous))
}

// TestRateLimitMiddleware_PerOperatorQuota is the T08 per-operator quota
// test: one operator exhausts its own bucket and receives 429 RATE_LIMITED,
// while another operator and unattributed traffic keep flowing from their own
// buckets. It exercises the exported middleware constructor directly, so it
// does not depend on how any entry point stacks the middleware.
func TestRateLimitMiddleware_PerOperatorQuota(t *testing.T) {
	t.Parallel()

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		ClientLimits: map[string]ClientLimit{
			"alice": {ClientID: "alice", RequestsPerSecond: 0, BurstSize: 1},
		},
		CleanupInterval: time.Minute,
	})
	middleware := NewRateLimitMiddleware(limiter, logger)

	handler := middleware.Middleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	serve := func(operator string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		if operator != "" {
			req = req.WithContext(context.WithValue(req.Context(), ContextKeyClientID, operator))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusOK, serve("alice").Code, "first request fits in alice's burst")

	second := serve("alice")
	require.Equal(t, http.StatusTooManyRequests, second.Code, "alice's second request must be throttled")
	assert.Contains(t, second.Body.String(), "RATE_LIMITED")
	assert.Equal(t, "1", second.Header().Get("X-RateLimit-Limit"))

	require.Equal(t, http.StatusOK, serve("bob").Code, "bob has a separate bucket")
	require.Equal(t, http.StatusOK, serve("").Code, "unattributed traffic uses the default bucket")
}
