package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBatchGetBatchSizeInitial(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, 0, plugin.GetBatchSize())
}

func TestBatchAddToBatchAndGetBatchSize(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	msg := MessageQueueMessage{ID: "msg-1", Topic: "test-topic"}
	err := plugin.AddToBatch(msg)
	assert.NoError(t, err)
	assert.Equal(t, 1, plugin.GetBatchSize())

	msg2 := MessageQueueMessage{ID: "msg-2", Topic: "test-topic"}
	err = plugin.AddToBatch(msg2)
	assert.NoError(t, err)
	assert.Equal(t, 2, plugin.GetBatchSize())
}

func TestBatchClearBatch(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	msg := MessageQueueMessage{ID: "msg-1", Topic: "test-topic"}
	_ = plugin.AddToBatch(msg)
	_ = plugin.AddToBatch(msg)
	assert.Equal(t, 2, plugin.GetBatchSize())

	plugin.ClearBatch()
	assert.Equal(t, 0, plugin.GetBatchSize())
}

func TestBatchSetAndGetBatchTimeout(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, 5*time.Second, plugin.GetBatchTimeout())

	plugin.SetBatchTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, plugin.GetBatchTimeout())
}

func TestBatchGetBatchProcessedCount(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	assert.Equal(t, int64(0), plugin.GetBatchProcessedCount())
}

func TestBatchProcessBatchEmpty(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	called := false
	err := plugin.ProcessBatch(context.Background(), func(msgs []MessageQueueMessage) error {
		called = true
		return nil
	})
	assert.NoError(t, err)
	assert.False(t, called)
}

func TestBatchProcessBatchSuccess(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	msg := MessageQueueMessage{ID: "msg-1", Topic: "test-topic"}
	_ = plugin.AddToBatch(msg)

	var received []MessageQueueMessage
	err := plugin.ProcessBatch(context.Background(), func(msgs []MessageQueueMessage) error {
		received = msgs
		return nil
	})
	assert.NoError(t, err)
	assert.Len(t, received, 1)
	assert.Equal(t, "msg-1", received[0].ID)
	assert.Equal(t, int64(1), plugin.GetBatchProcessedCount())
	assert.Equal(t, 0, plugin.GetBatchSize())
}

func TestBatchProcessBatchError(t *testing.T) {
	t.Parallel()
	config := Config{BlockchainNodeURL: "localhost:50051", StartBlock: 0}
	logger := NewDefaultLogger(LogLevelInfo)
	metrics := NewDefaultMetricsCollector()
	eventBus := NewEventBus(nil)
	plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

	msg := MessageQueueMessage{ID: "msg-1", Topic: "test-topic"}
	_ = plugin.AddToBatch(msg)

	err := plugin.ProcessBatch(context.Background(), func(msgs []MessageQueueMessage) error {
		return assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, int64(0), plugin.GetBatchProcessedCount())
}
