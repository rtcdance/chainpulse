package ports

// MetricsCollector collects metrics
type MetricsCollector interface {
	RecordCounter(name string, value int64, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
	GetMetrics() map[string]any
}

// PrometheusMetricsExporter is an optional capability for collectors that can
// render Prometheus exposition text directly.
type PrometheusMetricsExporter interface {
	ExportPrometheus() string
}
