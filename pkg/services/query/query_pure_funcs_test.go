package query

import "testing"

func TestClassifyQueryRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   string
		expected string
	}{
		{"healthy", "query-runtime-ready"},
		{"degraded", "query-runtime-degraded"},
		{"unhealthy", "query-runtime-unhealthy"},
		{"unknown", "query-runtime-unobserved"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			if got := classifyQueryRuntimePosture(tt.status); got != tt.expected {
				t.Errorf("classifyQueryRuntimePosture(%q) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestBuildQueryRuntimeReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       string
		cachePosture string
		expected     string
	}{
		{
			name:         "healthy with cache-ready",
			status:       "healthy",
			cachePosture: "cache-ready",
			expected:     "query runtime is healthy and cache-first reads are available",
		},
		{
			name:         "healthy without cache-ready",
			status:       "healthy",
			cachePosture: "cache-unhealthy",
			expected:     "query runtime is healthy, but cache posture should be verified before assuming cache-first reads",
		},
		{
			name:         "degraded with cache-unhealthy",
			status:       "degraded",
			cachePosture: "cache-unhealthy",
			expected:     "query runtime is degraded and cache is unhealthy; expect store-backed reads while cache is restored",
		},
		{
			name:         "degraded without cache-unhealthy",
			status:       "degraded",
			cachePosture: "cache-ready",
			expected:     "query runtime is degraded; investigate backing services before treating query reads as fully reliable",
		},
		{
			name:         "unhealthy",
			status:       "unhealthy",
			cachePosture: "cache-unhealthy",
			expected:     "query runtime is unhealthy; restore backing services before relying on query reads",
		},
		{
			name:         "unknown default",
			status:       "unknown",
			cachePosture: "cache-ready",
			expected:     "verify query runtime posture before relying on query service reads",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildQueryRuntimeReliabilityHint(tt.status, tt.cachePosture); got != tt.expected {
				t.Errorf("buildQueryRuntimeReliabilityHint(%q, %q) = %q, want %q", tt.status, tt.cachePosture, got, tt.expected)
			}
		})
	}
}
