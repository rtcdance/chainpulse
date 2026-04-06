package main

import (
	"net/http"
	"strings"

	"chainpulse/pkg/core"
)

func buildAPIGatewayMetricsProvider(metrics core.MetricsCollector) func(*http.Request) interface{} {
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

	return strings.Contains(r.Header.Get("Accept"), "application/json")
}
