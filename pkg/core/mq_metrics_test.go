package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBoundedShiftMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero", 0, 1},
		{"negative", -5, 1},
		{"one", 1, 1},
		{"two", 2, 2},
		{"three", 3, 4},
		{"four", 4, 8},
		{"five", 5, 16},
		{"ten", 10, 512},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, boundedShiftMultiplier(tt.input))
		})
	}
}

func TestMetricsRecordMetric(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordMetric("test_metric", int64(42), map[string]string{"key": "value"})
}

func TestMetricsLogInfo(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.LogInfo("test info message", "key1", "val1", "key2", 42)
}

func TestMetricsLogError(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.LogError("test error message", "error", "something went wrong")
}

func TestMetricsLogWarn(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.LogWarn("test warn message", "threshold", 0.8)
}

func TestMetricsRecordError(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordError(assert.AnError)
}

func TestMetricsGetAndSetLastBlockNumber(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, uint64(0), plugin.GetLastBlockNumber())

	plugin.SetLastBlockNumber(uint64(12345))
	assert.Equal(t, uint64(12345), plugin.GetLastBlockNumber())
}

func TestMetricsRecordPublishMetrics(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordPublishMetrics("test-topic", int64(1024), int64(50), true)
	plugin.RecordPublishMetrics("test-topic", int64(2048), int64(100), false)
}

func TestMetricsRecordConsumeMetrics(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordConsumeMetrics("test-topic", int64(30), true)
	plugin.RecordConsumeMetrics("test-topic", int64(60), false)
}

func TestMetricsRecordDLQMetrics(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordDLQMetrics("test-topic", "max_retries", int64(500), true)
	plugin.RecordDLQMetrics("test-topic", "timeout", int64(1000), false)
}

func TestMetricsRecordRetryMetrics(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordRetryMetrics("test-topic", 3, int64(1000))
}

func TestMetricsRecordAcknowledgmentMetrics(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	plugin.RecordAcknowledgmentMetrics("test-topic", int64(10))
}

func TestMetricsTrackAndCompleteInFlightOperation(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, int64(0), plugin.GetInFlightOperationCount())

	plugin.TrackInFlightOperation()
	assert.Equal(t, int64(1), plugin.GetInFlightOperationCount())

	plugin.CompleteInFlightOperation()
	assert.Equal(t, int64(0), plugin.GetInFlightOperationCount())
}

func TestMetricsCalculateExponentialBackoffDelay(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, 1*time.Second, plugin.CalculateExponentialBackoffDelay(0))
	assert.Equal(t, 1*time.Second, plugin.CalculateExponentialBackoffDelay(1))
	assert.Equal(t, 2*time.Second, plugin.CalculateExponentialBackoffDelay(2))
	assert.Equal(t, 4*time.Second, plugin.CalculateExponentialBackoffDelay(3))
	assert.Equal(t, 8*time.Second, plugin.CalculateExponentialBackoffDelay(4))
}
