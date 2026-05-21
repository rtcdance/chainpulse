package observability

import (
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestREDRecorderRecordRPCCall(t *testing.T) {
	t.Parallel()

	metrics := core.NewDefaultMetricsCollector()
	recorder := NewREDRecorder(metrics)

	recorder.RecordRPCCall("eth_getLogs", "ethereum", 100*time.Millisecond)

	count := metrics.GetCounter(MetricRPCCallsTotal, map[string]string{"method": "eth_getLogs", "chain_id": "ethereum", "status": "ok"})
	if count != 1 {
		t.Errorf("expected 1 RPC call, got %d", count)
	}
}

func TestREDRecorderRecordRPCError(t *testing.T) {
	t.Parallel()

	metrics := core.NewDefaultMetricsCollector()
	recorder := NewREDRecorder(metrics)

	recorder.RecordRPCError("eth_getLogs", "sepolia", "RPC_RATE_LIMITED", 200*time.Millisecond)

	errCount := metrics.GetCounter(MetricRPCErrorsTotal, map[string]string{
		"method": "eth_getLogs", "chain_id": "sepolia",
		"status": "error", "error_code": "RPC_RATE_LIMITED",
	})
	if errCount != 1 {
		t.Errorf("expected 1 RPC error, got %d", errCount)
	}
}

func TestREDRecorderRecordPullerCycle(t *testing.T) {
	t.Parallel()

	metrics := core.NewDefaultMetricsCollector()
	recorder := NewREDRecorder(metrics)

	recorder.RecordPullerCycle("ethereum", 5, nil)

	reqCount := metrics.GetCounter(MetricPullerRequestsTotal, map[string]string{"chain_id": "ethereum", "status": "ok"})
	if reqCount != 1 {
		t.Errorf("expected 1 puller request, got %d", reqCount)
	}

	eventCount := metrics.GetCounter(MetricPullerEventsTotal, map[string]string{"chain_id": "ethereum", "status": "ok"})
	if eventCount != 5 {
		t.Errorf("expected 5 events, got %d", eventCount)
	}

	// Record error cycle
	recorder.RecordPullerCycle("ethereum", 0, core.ErrRPCError)

	errCount := metrics.GetCounter(MetricPullerErrorsTotal, map[string]string{"chain_id": "ethereum", "status": "error"})
	if errCount != 1 {
		t.Errorf("expected 1 puller error, got %d", errCount)
	}
}

func TestREDRecorderRecordAPIRequest(t *testing.T) {
	t.Parallel()

	metrics := core.NewDefaultMetricsCollector()
	recorder := NewREDRecorder(metrics)

	recorder.RecordAPIRequest("/health", 200, 5*time.Millisecond)

	count := metrics.GetCounter(MetricApiRequestsTotal, map[string]string{"endpoint": "/health", "status": "ok"})
	if count != 1 {
		t.Errorf("expected 1 API request, got %d", count)
	}
}

func TestClassifyErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, "OK"},
		{"timeout", core.ErrTimeout, core.ErrorCodeTimeout},
		{"not found", core.ErrNotFound, core.ErrorCodeNotFound},
		{"block not found", core.ErrBlockNotFound, core.ErrorCodeBlockNotFound},
		{"RPC rate limited", core.ErrRPCRateLimited, core.ErrorCodeRPCRateLimited},
		{"unknown error", &testError{"something"}, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := core.ClassifyErrorCode(tt.err); got != tt.want {
				t.Errorf("ClassifyErrorCode(%v) = %s, want %s", tt.err, got, tt.want)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
