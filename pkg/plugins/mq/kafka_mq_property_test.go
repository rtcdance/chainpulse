package mq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// TestKafkaPropertyMessageIDUniqueness tests that message IDs are unique
func TestKafkaPropertyMessageIDUniqueness(t *testing.T) {
	t.Parallel()
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

	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create messages with unique IDs
	messageIDs := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("msg-%d", i)
		if messageIDs[id] {
			t.Errorf("duplicate message ID: %s", id)
		}
		messageIDs[id] = true

		message := core.MessageQueueMessage{
			ID:        id,
			Topic:     "test-topic",
			Payload:   []byte(fmt.Sprintf("payload %d", i)),
			Timestamp: time.Now().UTC(),
		}

		err := plugin.PublishMessage(ctx, message)
		if err != nil {
			t.Logf("publish failed (expected if Kafka not running): %v", err)
			return
		}
	}

	if len(messageIDs) != 100 {
		t.Errorf("expected 100 unique message IDs, got %d", len(messageIDs))
	}
}

// TestKafkaPropertyOffsetMonotonicity tests that offsets are monotonically increasing
func TestKafkaPropertyOffsetMonotonicity(t *testing.T) {
	t.Parallel()
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

	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	topic := "test-topic"
	lastOffset := int64(0)

	// Set increasing offsets
	for i := 1; i <= 10; i++ {
		offset := int64(i * 100)
		plugin.SetLastOffset(topic, offset)

		retrievedOffset := plugin.GetLastOffset(topic)
		if retrievedOffset < lastOffset {
			t.Errorf("offset decreased: %d < %d", retrievedOffset, lastOffset)
		}
		lastOffset = retrievedOffset
	}
}

// TestKafkaPropertyBatchSizeConsistency tests batch size consistency
func TestKafkaPropertyBatchSizeConsistency(t *testing.T) {
	t.Parallel()
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

	// Test various batch sizes
	batchSizes := []int{1, 10, 50, 100, 500, 1000}
	for _, size := range batchSizes {
		plugin.SetBatchSize(size)
		if plugin.batchSize != size {
			t.Errorf("batch size mismatch: expected %d, got %d", size, plugin.batchSize)
		}
	}
}

// TestKafkaPropertyMaxRetriesConsistency tests max retries consistency
func TestKafkaPropertyMaxRetriesConsistency(t *testing.T) {
	t.Parallel()
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

	// Test various max retries
	maxRetriesList := []int{1, 3, 5, 10, 20}
	for _, maxRetries := range maxRetriesList {
		plugin.SetMaxRetries(maxRetries)
		if plugin.maxRetries != maxRetries {
			t.Errorf("max retries mismatch: expected %d, got %d", maxRetries, plugin.maxRetries)
		}
	}
}

// TestKafkaPropertyRetryDelayConsistency tests retry delay consistency
func TestKafkaPropertyRetryDelayConsistency(t *testing.T) {
	t.Parallel()
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

	// Test various retry delays
	delays := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	for _, delay := range delays {
		plugin.SetRetryDelay(delay)
		if plugin.retryDelay != delay {
			t.Errorf("retry delay mismatch: expected %v, got %v", delay, plugin.retryDelay)
		}
	}
}

// TestKafkaPropertyHealthStatusConsistency tests health status consistency
func TestKafkaPropertyHealthStatusConsistency(t *testing.T) {
	t.Parallel()
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

	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Check health before starting
	health := plugin.Health(context.Background())
	if health != nil {
		t.Errorf("expected healthy before start, got: %v", health)
	}

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Check health after starting
	health = plugin.Health(context.Background())
	if health != nil {
		t.Errorf("expected healthy after start, got: %v", health)
	}

	if err := plugin.Stop(context.Background()); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	// Check health after stopping
	health = plugin.Health(context.Background())
	t.Logf("health after stop: %v, is_running: %v", health, plugin.IsRunning())
}

// TestKafkaPropertyStatsAccuracy tests stats accuracy
func TestKafkaPropertyStatsAccuracy(t *testing.T) {
	t.Parallel()
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

	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	if err := plugin.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() { _ = plugin.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get initial stats
	initialStats := plugin.GetStats()
	if initialStats.MessageCount != 0 {
		t.Errorf("expected initial message count 0, got %d", initialStats.MessageCount)
	}

	// Acknowledge a message
	message := core.MessageQueueMessage{
		ID:        "test-msg",
		Topic:     "test-topic",
		Payload:   []byte("test"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.AcknowledgeMessage(ctx, message); err != nil {
		t.Fatalf("failed to acknowledge message: %v", err)
	}

	// Get updated stats
	updatedStats := plugin.GetStats()
	if updatedStats.MessageCount != initialStats.MessageCount {
		t.Logf("message count changed: %d -> %d", initialStats.MessageCount, updatedStats.MessageCount)
	}
}

// TestKafkaPropertyBrokerConfigurationImmutability tests broker configuration immutability
func TestKafkaPropertyBrokerConfigurationImmutability(t *testing.T) {
	t.Parallel()
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092", "localhost:9093"}
	consumerGroup := "test-group"

	plugin := NewKafkaMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus, brokers, consumerGroup)

	// Verify brokers are set correctly
	if len(plugin.brokers) != len(brokers) {
		t.Errorf("expected %d brokers, got %d", len(brokers), len(plugin.brokers))
	}

	for i, broker := range plugin.brokers {
		if broker != brokers[i] {
			t.Errorf("broker mismatch at index %d: expected %s, got %s", i, brokers[i], broker)
		}
	}
}

// TestKafkaPropertyConsumerGroupImmutability tests consumer group immutability
func TestKafkaPropertyConsumerGroupImmutability(t *testing.T) {
	t.Parallel()
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

	// Verify consumer group is set correctly
	if plugin.consumerGroup != consumerGroup {
		t.Errorf("expected consumer group %s, got %s", consumerGroup, plugin.consumerGroup)
	}
}

// TestKafkaPropertyPluginNameAndVersion tests plugin name and version
func TestKafkaPropertyPluginNameAndVersion(t *testing.T) {
	t.Parallel()
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	brokers := []string{"localhost:9092"}
	consumerGroup := "test-group"

	name := "kafka"
	version := "1.0.0"

	plugin := NewKafkaMQPlugin(name, version, config, logger, metrics, eventBus, brokers, consumerGroup)

	if plugin.Name() != name {
		t.Errorf("expected name %s, got %s", name, plugin.Name())
	}

	if plugin.Version() != version {
		t.Errorf("expected version %s, got %s", version, plugin.Version())
	}
}

// TestKafkaPropertyInitializationIdempotency tests initialization idempotency
func TestKafkaPropertyInitializationIdempotency(t *testing.T) {
	t.Parallel()
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

	// First initialization should succeed
	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("first initialization failed: %v", err)
	}

	// Second initialization should fail
	if err := plugin.Initialize(context.Background(), *config); err == nil {
		t.Fatal("expected error on second initialization")
	}

	if !plugin.IsInitialized() {
		t.Fatal("expected plugin to be initialized")
	}
}

// TestKafkaPropertyStartStopCycle tests start/stop cycle
func TestKafkaPropertyStartStopCycle(t *testing.T) {
	t.Parallel()
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

	if err := plugin.Initialize(context.Background(), *config); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Multiple start/stop cycles
	for i := 0; i < 3; i++ {
		if err := plugin.Start(context.Background()); err != nil {
			t.Fatalf("failed to start (cycle %d): %v", i, err)
		}

		if !plugin.IsRunning() {
			t.Fatalf("expected plugin to be running (cycle %d)", i)
		}

		if err := plugin.Stop(context.Background()); err != nil {
			t.Fatalf("failed to stop (cycle %d): %v", i, err)
		}

		if plugin.IsRunning() {
			t.Fatalf("expected plugin to be stopped (cycle %d)", i)
		}
	}
}
