package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the prometheus metrics for the application
type Metrics struct {
	// Blockchain metrics
	BlocksProcessedTotal    prometheus.Counter
	EventsProcessedTotal    prometheus.Counter
	EventsIndexedTotal      prometheus.Counter
	EventsCacheHitsTotal    prometheus.Counter
	EventsCacheMissesTotal  prometheus.Counter
	
	// API metrics
	APIRequestsTotal        *prometheus.CounterVec
	APIRequestDuration      *prometheus.HistogramVec
	ActiveConnections       prometheus.Gauge
	
	// Database metrics
	DatabaseQueryDuration   *prometheus.HistogramVec
	DatabaseConnections     prometheus.Gauge
	
	// Error metrics
	ErrorsTotal             *prometheus.CounterVec
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// Blockchain metrics
		BlocksProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chainpulse_blocks_processed_total",
			Help: "Total number of blocks processed",
		}),
		EventsProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chainpulse_events_processed_total",
			Help: "Total number of events processed",
		}),
		EventsIndexedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chainpulse_events_indexed_total",
			Help: "Total number of events indexed in database",
		}),
		EventsCacheHitsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chainpulse_events_cache_hits_total",
			Help: "Total number of cache hits for events",
		}),
		EventsCacheMissesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chainpulse_events_cache_misses_total",
			Help: "Total number of cache misses for events",
		}),
		
		// API metrics
		APIRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chainpulse_api_requests_total",
			Help: "Total number of API requests",
		}, []string{"method", "endpoint", "status"}),
		APIRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "chainpulse_api_request_duration_seconds",
			Help: "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "endpoint"}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chainpulse_active_connections",
			Help: "Number of active API connections",
		}),
		
		// Database metrics
		DatabaseQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "chainpulse_database_query_duration_seconds",
			Help: "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"query_type", "table"}),
		DatabaseConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chainpulse_database_connections",
			Help: "Number of active database connections",
		}),
		
		// Error metrics
		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "chainpulse_errors_total",
			Help: "Total number of errors",
		}, []string{"component", "error_type"}),
	}
	
	return m
}

// IncrementBlocksProcessed increments the blocks processed counter
func (m *Metrics) IncrementBlocksProcessed() {
	m.BlocksProcessedTotal.Inc()
}

// IncrementEventsProcessed increments the events processed counter
func (m *Metrics) IncrementEventsProcessed() {
	m.EventsProcessedTotal.Inc()
}

// IncrementEventsIndexed increments the events indexed counter
func (m *Metrics) IncrementEventsIndexed() {
	m.EventsIndexedTotal.Inc()
}

// IncrementCacheHit increments the cache hit counter
func (m *Metrics) IncrementCacheHit() {
	m.EventsCacheHitsTotal.Inc()
}

// IncrementCacheMiss increments the cache miss counter
func (m *Metrics) IncrementCacheMiss() {
	m.EventsCacheMissesTotal.Inc()
}

// RecordAPIRequest records an API request
func (m *Metrics) RecordAPIRequest(method, endpoint, status string) {
	m.APIRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordAPIRequestDuration records API request duration
func (m *Metrics) RecordAPIRequestDuration(method, endpoint string, duration float64) {
	m.APIRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// SetActiveConnections sets the active connections gauge
func (m *Metrics) SetActiveConnections(count float64) {
	m.ActiveConnections.Set(count)
}

// RecordDatabaseQueryDuration records database query duration
func (m *Metrics) RecordDatabaseQueryDuration(queryType, table string, duration float64) {
	m.DatabaseQueryDuration.WithLabelValues(queryType, table).Observe(duration)
}

// SetDatabaseConnections sets the database connections gauge
func (m *Metrics) SetDatabaseConnections(count float64) {
	m.DatabaseConnections.Set(count)
}

// IncrementError increments the error counter
func (m *Metrics) IncrementError(component, errorType string) {
	m.ErrorsTotal.WithLabelValues(component, errorType).Inc()
}