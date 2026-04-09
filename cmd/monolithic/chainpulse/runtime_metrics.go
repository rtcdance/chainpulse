package main

import (
	"net/http"

	"chainpulse/pkg/core"
)

func buildMonolithicMetricsProvider(metrics core.MetricsCollector) func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		if metrics == nil {
			return nil
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
