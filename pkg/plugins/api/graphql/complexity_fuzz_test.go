package graphql

import (
	"testing"
)

func FuzzCalculateQueryComplexity(f *testing.F) {
	seeds := []string{
		`{ event(id: "1") { id name } }`,
		`query { events(first: 10) { edges { node { id blockNumber } cursor } } }`,
		`mutation { warmCache(limit: 100) }`,
		`{ __schema { types { name } } }`,
		``,
		`not valid graphql at all {`,
		`{ deeply { nested { field { with { many { levels { here } } } } } } }`,
		`fragment EventFields on Event { id name blockNumber } query { event(id: "1") { ...EventFields } }`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, query string) {
		// Must not panic for any input
		complexity := calculateQueryComplexity(query)

		// Complexity must be non-negative
		if complexity < 0 {
			t.Errorf("negative complexity %d for query %q", complexity, query)
		}

		// Empty query should return minimum complexity (1)
		if query == "" && complexity != 1 {
			t.Errorf("empty query should have complexity 1, got %d", complexity)
		}
	})
}

func FuzzComplexityMiddlewareAnalyzeQuery(f *testing.F) {
	seeds := []string{
		`{ event(id: "1") { id } }`,
		`{ events(first: 100) { edges { node { id name contractAddress } } } }`,
		``,
		`{ invalid {{{{`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, query string) {
		logger := NewMockLogger()
		metrics := NewMockMetrics()
		cm := NewComplexityMiddleware(1000, logger, metrics)

		// Must not panic for any input
		_, _ = cm.AnalyzeQuery(query)
	})
}
