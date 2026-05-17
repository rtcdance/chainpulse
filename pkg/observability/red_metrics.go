package observability

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// Metric name constants for RED (Rate/Errors/Duration) observability.
// Every RPC operation should emit these three indicators consistently.
const (
	// MetricRPCCallsTotal is the counter for total RPC calls, tagged by method and chain.
	MetricRPCCallsTotal = "chainpulse_rpc_calls_total"

	// MetricRPCErrorsTotal is the counter for failed RPC calls, tagged by method, chain, and error_code.
	MetricRPCErrorsTotal = "chainpulse_rpc_errors_total"

	// MetricRPCDurationSeconds is a histogram for RPC call latency, tagged by method and chain.
	MetricRPCDurationSeconds = "chainpulse_rpc_duration_seconds"

	// MetricPullerRequestsTotal is the counter for puller poll cycles.
	MetricPullerRequestsTotal = "chainpulse_puller_requests_total"

	// MetricPullerErrorsTotal is the counter for failed puller poll cycles.
	MetricPullerErrorsTotal = "chainpulse_puller_errors_total"

	// MetricPullerEventsTotal is the counter for pulled events.
	MetricPullerEventsTotal = "chainpulse_puller_events_total"

	// MetricIndexerBlocksTotal is the counter for indexed blocks.
	MetricIndexerBlocksTotal = "chainpulse_indexer_blocks_total"

	// MetricIndexerEventsTotal is the counter for indexed events.
	MetricIndexerEventsTotal = "chainpulse_indexer_events_total"

	// MetricApiRequestsTotal is the counter for API requests.
	MetricApiRequestsTotal = "chainpulse_api_requests_total"

	// MetricApiDurationSeconds is a histogram for API request latency.
	MetricApiDurationSeconds = "chainpulse_api_duration_seconds"
)

// Common metric tag keys.
const (
	TagMethod    = "method"
	TagChain     = "chain_id"
	TagStatus    = "status"
	TagErrorCode = "error_code"
	TagEndpoint  = "endpoint"
)

// REDRecorder provides standardized RED (Rate/Errors/Duration) metric recording.
// All RPC operations should use this instead of ad-hoc metric names.
type REDRecorder struct {
	metrics core.MetricsCollector
}

// NewREDRecorder creates a RED recorder wrapping the given metrics collector.
func NewREDRecorder(metrics core.MetricsCollector) *REDRecorder {
	return &REDRecorder{metrics: metrics}
}

// RecordRPCCall records a successful RPC call with its duration.
func (r *REDRecorder) RecordRPCCall(method, chainID string, duration time.Duration) {
	tags := map[string]string{TagMethod: method, TagChain: chainID, TagStatus: "ok"}
	r.metrics.RecordCounter(MetricRPCCallsTotal, 1, tags)
	r.metrics.RecordHistogram(MetricRPCDurationSeconds, duration.Seconds(), tags)
}

// RecordRPCError records a failed RPC call tagged by error code.
func (r *REDRecorder) RecordRPCError(method, chainID, errorCode string, duration time.Duration) {
	tags := map[string]string{TagMethod: method, TagChain: chainID, TagStatus: "error", TagErrorCode: errorCode}
	r.metrics.RecordCounter(MetricRPCErrorsTotal, 1, tags)
	r.metrics.RecordHistogram(MetricRPCDurationSeconds, duration.Seconds(), tags)
}

// RecordPullerCycle records a puller poll cycle.
func (r *REDRecorder) RecordPullerCycle(chainID string, events int, err error) {
	if err != nil {
		tags := map[string]string{TagChain: chainID, TagStatus: "error"}
		r.metrics.RecordCounter(MetricPullerErrorsTotal, 1, tags)
		return
	}
	tags := map[string]string{TagChain: chainID, TagStatus: "ok"}
	r.metrics.RecordCounter(MetricPullerRequestsTotal, 1, tags)
	r.metrics.RecordCounter(MetricPullerEventsTotal, int64(events), tags)
}

// RecordAPIRequest records an API request with its duration and status.
func (r *REDRecorder) RecordAPIRequest(endpoint string, statusCode int, duration time.Duration) {
	status := "ok"
	if statusCode >= 400 {
		status = "error"
	}
	tags := map[string]string{TagEndpoint: endpoint, TagStatus: status}
	r.metrics.RecordCounter(MetricApiRequestsTotal, 1, tags)
	r.metrics.RecordHistogram(MetricApiDurationSeconds, duration.Seconds(), tags)
}
