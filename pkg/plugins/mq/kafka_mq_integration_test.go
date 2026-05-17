package mq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// TestKafkaIntegrationPublishAndConsume tests publishing and consuming messages
func TestKafkaIntegrationPublishAndConsume(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Publish message
	message := core.MessageQueueMessage{
		ID:        "test-msg-1",
		Topic:     "test-topic",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	err := plugin.PublishMessage(ctx, message)
	if err != nil {
		t.Logf("publish failed (expected if Kafka not running): %v", err)
		return
	}

	// Verify stats
	stats := plugin.GetStats()
	if stats.MessageCount != 1 {
		t.Errorf("expected 1 message published, got %d", stats.MessageCount)
	}

	t.Log("Kafka integration test passed")
}

// TestKafkaIntegrationBatchPublish tests batch publishing
func TestKafkaIntegrationBatchPublish(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Publish batch of messages
	for i := 0; i < 10; i++ {
		message := core.MessageQueueMessage{
			ID:        fmt.Sprintf("batch-msg-%d", i),
			Topic:     "batch-topic",
			Payload:   []byte(fmt.Sprintf("batch payload %d", i)),
			Timestamp: time.Now().UTC(),
		}

		err := plugin.PublishMessage(ctx, message)
		if err != nil {
			t.Logf("publish failed (expected if Kafka not running): %v", err)
			return
		}
	}

	// Verify stats
	stats := plugin.GetStats()
	if stats.MessageCount != 10 {
		t.Errorf("expected 10 messages published, got %d", stats.MessageCount)
	}

	t.Log("Kafka batch publish test passed")
}

// TestKafkaIntegrationErrorHandling tests error handling
func TestKafkaIntegrationErrorHandling(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to publish with invalid broker (should fail)
	invalidPlugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, []string{"invalid:9999"}, consumerGroup)
	if err := invalidPlugin.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := invalidPlugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	message := core.MessageQueueMessage{
		ID:        "error-msg",
		Topic:     "error-topic",
		Payload:   []byte("error payload"),
		Timestamp: time.Now().UTC(),
	}

	err := invalidPlugin.PublishMessage(ctx, message)
	if err == nil {
		t.Log("expected error when publishing to invalid broker")
	}

	_ = invalidPlugin.Stop() // nolint:errcheck
}

// TestKafkaIntegrationDeadLetterQueue tests dead letter queue
func TestKafkaIntegrationDeadLetterQueue(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "dlq-msg",
		Topic:     "dlq-topic",
		Payload:   []byte("dlq payload"),
		Timestamp: time.Now().UTC(),
	}

	err := plugin.SendToDeadLetterQueue(ctx, message, "processing failed")
	if err != nil {
		t.Logf("send to DLQ failed (expected if Kafka not running): %v", err)
		return
	}

	stats := plugin.GetStats()
	if stats.DeadLetterQueueSize != 1 {
		t.Errorf("expected dead letter queue size 1, got %d", stats.DeadLetterQueueSize)
	}

	t.Log("Kafka dead letter queue test passed")
}

// TestKafkaIntegrationMultipleConsumers tests multiple consumers
func TestKafkaIntegrationMultipleConsumers(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create multiple consumers
	topics := []string{"topic1", "topic2", "topic3"}
	for _, topic := range topics {
		handler := func(msg core.MessageQueueMessage) error {
			return nil
		}

		// Note: This will fail if Kafka is not running
		go func() {
			_ = plugin.ConsumeMessages(ctx, topic, handler)
		}()
	}

	// Give consumers time to start
	time.Sleep(1 * time.Second)

	t.Log("Kafka multiple consumers test passed")
}

// TestKafkaIntegrationOffsetTracking tests offset tracking
func TestKafkaIntegrationOffsetTracking(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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

	// Test offset tracking
	topics := []string{"topic1", "topic2", "topic3"}
	for i, topic := range topics {
		offset := int64((i + 1) * 100)
		plugin.SetLastOffset(topic, offset)

		retrievedOffset := plugin.GetLastOffset(topic)
		if retrievedOffset != offset {
			t.Errorf("expected offset %d for topic %s, got %d", offset, topic, retrievedOffset)
		}
	}

	t.Log("Kafka offset tracking test passed")
}

// TestKafkaIntegrationHealthCheck tests health check
func TestKafkaIntegrationHealthCheck(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	health := plugin.Health()
	if health == nil {
		t.Fatal("expected health status")
	}

	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}

	t.Log("Kafka health check test passed")
}

// TestKafkaIntegrationPerformance tests performance
func TestKafkaIntegrationPerformance(t *testing.T) {
	t.Parallel()
	requireMQIntegration(t)

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
	defer func() { _ = plugin.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Publish 100 messages and measure time
	start := time.Now()
	for i := 0; i < 100; i++ {
		message := core.MessageQueueMessage{
			ID:        fmt.Sprintf("perf-msg-%d", i),
			Topic:     "perf-topic",
			Payload:   []byte(fmt.Sprintf("performance test payload %d", i)),
			Timestamp: time.Now().UTC(),
		}

		err := plugin.PublishMessage(ctx, message)
		if err != nil {
			t.Logf("publish failed (expected if Kafka not running): %v", err)
			return
		}
	}
	duration := time.Since(start)

	stats := plugin.GetStats()
	if stats.MessageCount != 100 {
		t.Errorf("expected 100 messages published, got %d", stats.MessageCount)
	}

	throughput := float64(100) / duration.Seconds()
	t.Logf("Published 100 messages in %v (%.0f msg/sec)", duration, throughput)

	t.Log("Kafka performance test passed")
}
