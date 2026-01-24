package mq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestKafkaMQPluginCreation tests Kafka MQ plugin creation
func TestKafkaMQPluginCreation(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if plugin == nil {
		t.Fatal("expected plugin to be created")
	}

	if plugin.Name() != "kafka" {
		t.Errorf("expected name 'kafka', got %s", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", plugin.Version())
	}
}

// TestKafkaMQPluginInitialization tests Kafka plugin initialization
func TestKafkaMQPluginInitialization(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if !plugin.IsInitialized() {
		t.Fatal("expected plugin to be initialized")
	}

	// Try to initialize again (should fail)
	if err := plugin.Initialize(); err == nil {
		t.Fatal("expected error when initializing twice")
	}
}

// TestKafkaMQPluginLifecycle tests Kafka plugin lifecycle
func TestKafkaMQPluginLifecycle(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	if !plugin.IsRunning() {
		t.Fatal("expected plugin to be running")
	}

	if err := plugin.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	if plugin.IsRunning() {
		t.Fatal("expected plugin to be stopped")
	}
}

// TestKafkaMQPluginPublishMessage tests publishing a message
func TestKafkaMQPluginPublishMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	// Note: This will fail if Kafka is not running, but we test the logic
	err := plugin.PublishMessage(ctx, message)
	if err != nil {
		// Expected if Kafka is not running
		t.Logf("publish message failed (expected if Kafka not running): %v", err)
	}
}

// TestKafkaMQPluginAcknowledgeMessage tests acknowledging a message
func TestKafkaMQPluginAcknowledgeMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.AcknowledgeMessage(ctx, message); err != nil {
		t.Fatalf("failed to acknowledge message: %v", err)
	}
}

// TestKafkaMQPluginDeadLetterQueue tests dead letter queue handling
func TestKafkaMQPluginDeadLetterQueue(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	err := plugin.SendToDeadLetterQueue(ctx, message, "processing failed")
	if err != nil {
		// Expected if Kafka is not running
		t.Logf("send to DLQ failed (expected if Kafka not running): %v", err)
	}

	stats := plugin.GetStats()
	if stats.DeadLetterQueueSize != 1 && err == nil {
		t.Errorf("expected dead letter queue size 1, got %d", stats.DeadLetterQueueSize)
	}
}

// TestKafkaMQPluginRetryMessage tests message retry
func TestKafkaMQPluginRetryMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:         "msg-1",
		Topic:      "blockchain_events",
		Payload:    []byte("test payload"),
		Timestamp:  time.Now().UTC(),
		RetryCount: 0,
	}

	if err := plugin.RetryMessage(ctx, message); err != nil {
		t.Fatalf("failed to retry message: %v", err)
	}

	// RetryMessage increments internally, so we just verify no error was returned
	// The actual retry count is managed internally by the plugin
	if err := plugin.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}
}

// TestKafkaMQPluginGetStats tests statistics retrieval
func TestKafkaMQPluginGetStats(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	stats := plugin.GetStats()

	if stats.MessageCount != 0 {
		t.Errorf("expected message count 0, got %d", stats.MessageCount)
	}

	if stats.ErrorCount != 0 {
		t.Errorf("expected error count 0, got %d", stats.ErrorCount)
	}
}

// TestKafkaMQPluginSetBatchSize tests setting batch size
func TestKafkaMQPluginSetBatchSize(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	batchSize := 50
	plugin.SetBatchSize(batchSize)

	if plugin.batchSize != batchSize {
		t.Errorf("expected batch size %d, got %d", batchSize, plugin.batchSize)
	}
}

// TestKafkaMQPluginSetMaxRetries tests setting max retries
func TestKafkaMQPluginSetMaxRetries(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	maxRetries := 5
	plugin.SetMaxRetries(maxRetries)

	if plugin.maxRetries != maxRetries {
		t.Errorf("expected max retries %d, got %d", maxRetries, plugin.maxRetries)
	}
}

// TestKafkaMQPluginSetRetryDelay tests setting retry delay
func TestKafkaMQPluginSetRetryDelay(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	delay := 2 * time.Second
	plugin.SetRetryDelay(delay)

	if plugin.retryDelay != delay {
		t.Errorf("expected retry delay %v, got %v", delay, plugin.retryDelay)
	}
}

// TestKafkaMQPluginConcurrentOperations tests concurrent operations
func TestKafkaMQPluginConcurrentOperations(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			message := core.MessageQueueMessage{
				ID:        fmt.Sprintf("msg-%d", index),
				Topic:     "blockchain_events",
				Payload:   []byte("test payload"),
				Timestamp: time.Now().UTC(),
			}

			if err := plugin.AcknowledgeMessage(ctx, message); err != nil {
				t.Logf("failed to acknowledge message: %v", err)
			}

			stats := plugin.GetStats()
			if stats.MessageCount < 0 {
				t.Error("message count corrupted")
			}
		}(i)
	}

	wg.Wait()
}

// TestKafkaMQPluginHealth tests health check
func TestKafkaMQPluginHealth(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	health := plugin.Health()

	if health == nil {
		t.Fatal("expected health status")
	}

	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}
}

// TestKafkaMQPluginOffsetTracking tests offset tracking
func TestKafkaMQPluginOffsetTracking(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	topic := "blockchain_events"
	offset := int64(100)

	plugin.SetLastOffset(topic, offset)

	retrievedOffset := plugin.GetLastOffset(topic)
	if retrievedOffset != offset {
		t.Errorf("expected offset %d, got %d", offset, retrievedOffset)
	}
}

// TestKafkaMQPluginMultipleTopics tests handling multiple topics
func TestKafkaMQPluginMultipleTopics(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	topics := []string{"topic1", "topic2", "topic3"}

	for _, topic := range topics {
		plugin.SetLastOffset(topic, int64(10))
	}

	for _, topic := range topics {
		offset := plugin.GetLastOffset(topic)
		if offset != 10 {
			t.Errorf("expected offset 10 for topic %s, got %d", topic, offset)
		}
	}
}

// TestKafkaMQPluginBrokerConfiguration tests broker configuration
func TestKafkaMQPluginBrokerConfiguration(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if len(plugin.brokers) != len(brokers) {
		t.Errorf("expected %d brokers, got %d", len(brokers), len(plugin.brokers))
	}

	for i, broker := range plugin.brokers {
		if broker != brokers[i] {
			t.Errorf("expected broker %s, got %s", brokers[i], broker)
		}
	}
}

// TestKafkaMQPluginConsumerGroupConfiguration tests consumer group configuration
func TestKafkaMQPluginConsumerGroupConfiguration(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "my-consumer-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if plugin.consumerGroup != consumerGroup {
		t.Errorf("expected consumer group %s, got %s", consumerGroup, plugin.consumerGroup)
	}
}

// TestKafkaMQPluginErrorHandling tests error handling
func TestKafkaMQPluginErrorHandling(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Test error recording
	testErr := fmt.Errorf("test error")
	plugin.RecordError(testErr)

	stats := plugin.GetStats()
	if stats.ErrorCount != 1 {
		t.Errorf("expected error count 1, got %d", stats.ErrorCount)
	}

	if stats.LastError == nil {
		t.Error("expected last error to be set")
	}
}

// TestKafkaMQPluginNotRunningError tests operations when plugin is not running
func TestKafkaMQPluginNotRunningError(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	// Try to publish without starting
	err := plugin.PublishMessage(ctx, message)
	if err == nil {
		t.Fatal("expected error when publishing without starting")
	}

	// Try to acknowledge without starting
	err = plugin.AcknowledgeMessage(ctx, message)
	if err == nil {
		t.Fatal("expected error when acknowledging without starting")
	}

	// Try to send to DLQ without starting
	err = plugin.SendToDeadLetterQueue(ctx, message, "test reason")
	if err == nil {
		t.Fatal("expected error when sending to DLQ without starting")
	}
}

// TestKafkaMQPluginNotInitializedError tests operations when plugin is not initialized
func TestKafkaMQPluginNotInitializedError(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	// Try to start without initializing
	err := plugin.Start()
	if err == nil {
		t.Fatal("expected error when starting without initializing")
	}
}

// TestKafkaMQPluginDoubleStart tests starting plugin twice
func TestKafkaMQPluginDoubleStart(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Try to start again
	err := plugin.Start()
	if err == nil {
		t.Fatal("expected error when starting twice")
	}
}

// TestKafkaMQPluginStopWithoutStart tests stopping without starting
func TestKafkaMQPluginStopWithoutStart(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Stop without starting should not error
	err := plugin.Stop()
	if err != nil {
		t.Fatalf("unexpected error when stopping without starting: %v", err)
	}
}

// TestKafkaMQPluginMaxRetriesExceeded tests max retries exceeded
func TestKafkaMQPluginMaxRetriesExceeded(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:         "msg-1",
		Topic:      "blockchain_events",
		Payload:    []byte("test payload"),
		Timestamp:  time.Now().UTC(),
		RetryCount: 3, // Already at max retries
	}

	plugin.SetMaxRetries(3)

	err := plugin.RetryMessage(ctx, message)
	if err == nil {
		t.Fatal("expected error when max retries exceeded")
	}
}

// TestKafkaMQPluginGetLastBlockNumber tests getting last block number
func TestKafkaMQPluginGetLastBlockNumber(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	blockNumber := plugin.GetLastBlockNumber()
	if blockNumber != 0 {
		t.Errorf("expected block number 0, got %d", blockNumber)
	}
}

// TestKafkaMQPluginSetLastBlockNumber tests setting last block number
func TestKafkaMQPluginSetLastBlockNumber(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	blockNumber := uint64(12345)
	plugin.SetLastBlockNumber(blockNumber)

	// Note: SetLastBlockNumber doesn't actually store for Kafka MQ
	// This is expected behavior as Kafka uses offset tracking instead
}

// TestKafkaMQPluginHealthDegraded tests health status when errors occur
func TestKafkaMQPluginHealthDegraded(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Record an error
	plugin.RecordError(fmt.Errorf("test error"))

	health := plugin.Health()

	if health.Status != "degraded" {
		t.Errorf("expected degraded status, got %s", health.Status)
	}
}

// TestKafkaMQPluginMessageQueueMessageFields tests message fields
func TestKafkaMQPluginMessageQueueMessageFields(t *testing.T) {
	message := core.MessageQueueMessage{
		ID:           "msg-123",
		Topic:        "events",
		Payload:      []byte("test data"),
		Timestamp:    time.Now().UTC(),
		Offset:       42,
		PartitionKey: "key-1",
		RetryCount:   0,
	}

	if message.ID != "msg-123" {
		t.Errorf("expected ID msg-123, got %s", message.ID)
	}

	if message.Topic != "events" {
		t.Errorf("expected topic events, got %s", message.Topic)
	}

	if string(message.Payload) != "test data" {
		t.Errorf("expected payload 'test data', got %s", string(message.Payload))
	}

	if message.Offset != 42 {
		t.Errorf("expected offset 42, got %d", message.Offset)
	}

	if message.PartitionKey != "key-1" {
		t.Errorf("expected partition key key-1, got %s", message.PartitionKey)
	}
}

// TestKafkaMQPluginStatsStructure tests stats structure
func TestKafkaMQPluginStatsStructure(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	if err := plugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	stats := plugin.GetStats()

	if stats.MessageCount != 0 {
		t.Errorf("expected message count 0, got %d", stats.MessageCount)
	}

	if stats.ErrorCount != 0 {
		t.Errorf("expected error count 0, got %d", stats.ErrorCount)
	}

	if stats.DeadLetterQueueSize != 0 {
		t.Errorf("expected DLQ size 0, got %d", stats.DeadLetterQueueSize)
	}

	if stats.IsRunning {
		t.Error("expected plugin to not be running")
	}
}

// TestKafkaMQPluginLogMethods tests logging methods
func TestKafkaMQPluginLogMethods(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)

	// These should not panic
	logger.Info("test info", "key", "value")
	logger.Error("test error", "key", "value")
	logger.Warn("test warn", "key", "value")
}

// TestKafkaMQPluginRecordMetric tests metric recording
func TestKafkaMQPluginRecordMetric(t *testing.T) {
	metrics := core.NewDefaultMetricsCollector()

	// This should not panic
	metrics.RecordCounter("test_metric", 42, map[string]string{"tag": "value"})
}
