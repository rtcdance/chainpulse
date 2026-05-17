package main

import (
	"net/http"

	"chainpulse/pkg/core"
	"chainpulse/pkg/observability"
)

func buildMonolithicMetricsProvider(metrics core.MetricsCollector, indexerMetrics *observability.IndexerMetrics) func(*http.Request) any {
	return func(r *http.Request) any {
		if metrics == nil {
			return nil
		}

		// Sync business metrics from IndexerMetrics to MetricsCollector
		// so they appear on the /metrics endpoint alongside system metrics
		if indexerMetrics != nil {
			indexerMetrics.SyncToMetricsCollector(metrics)
		}

		if prefersJSONMetrics(r) {
			return metrics.GetMetrics()
		}

		return core.ExportMetricsPrometheus(metrics)
	}
}

func prefersJSONMetrics(r *http.Request) bool {
	if r == nil {
		return false
	}

	format := r.URL.Query().Get("format")
	if format == "json" {
		return true
	}

	return r.Header.Get("Accept") == "application/json"
}
