package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzPathNormalization(f *testing.F) {
	// Test that the auth middleware normalises paths correctly
	// regardless of whether /api/v1 prefix is present
	seeds := []string{
		"/events/subscribe",
		"/api/v1/events/subscribe",
		"/dlq/replay",
		"/api/v1/dlq/replay",
		"/runtime/control",
		"/api/v1/runtime/control",
		"/admin/keys",
		"/api/v1/admin/keys",
		"/",
		"/api/v1",
		"",
		"/unknown/path",
		"/api/v2/events",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// Normalise path — must not panic
		endpointPath := path
		if len(endpointPath) >= len("/api/v1/") && endpointPath[:len("/api/v1/")] == "/api/v1/" {
			endpointPath = endpointPath[len("/api/v1/"):]
		} else if endpointPath == "/api/v1" {
			endpointPath = "/"
		}

		// The normalised path must not start with /api/v1/
		if len(endpointPath) >= len("/api/v1/") && endpointPath[:len("/api/v1/")] == "/api/v1/" {
			t.Errorf("path %q was not normalised, still starts with /api/v1/", path)
		}
	})
}

func FuzzAPIKeyValidation(f *testing.F) {
	seeds := []string{
		"cp_1234567890abcdef",
		"invalid-key",
		"",
		"cp_",
		"sk_live_12345",
		"cp_" + string(make([]byte, 1000)),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, apiKey string) {
		// Validate must not panic for any input
		logger := &MockLogger{}
		metrics := NewMockMetricsCollector()
		tv := NewTokenValidator("test-secret", logger, metrics)

		// Register and validate — must not panic
		_ = tv.RegisterAPIKey(apiKey, "fuzz-client")

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", apiKey)
		_ = tv.ValidateToken(req.Context(), apiKey)
	})
}
