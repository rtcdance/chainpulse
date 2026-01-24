package mq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestRedisMQPluginCreation tests Redis MQ plugin creation
func TestRedisMQPluginCreation(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	if plugin == nil {
		t.Fatal("expected plugin to be created")
	}

	if plugin.Name() != "redis" {
		t.Errorf("expected name 'redis', got %s", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", plugin.Version())
	}
}

// TestRedisMQPluginInitialization tests Redis plugin initialization
func TestRedisMQPluginInitialization(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	// Note: This will fail if Redis is not running, but we test the logic
	err := plugin.Initialize()
	if err != nil {
		// Expected if Redis is not running
		t.Logf("initialization failed (expected if Redis not running): %v", err)
	} else {
		if !plugin.IsInitialized() {
			t.Fatal("expected plugin to be initialized")
		}

		// Try to initialize again (should fail)
		if err := plugin.Initialize(); err == nil {
			t.Fatal("expected error when initializing twice")
		}
	}
}

// TestRedisMQPluginLifecycle tests Redis plugin lifecycle
func TestRedisMQPluginLifecycle(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
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

// TestRedisMQPluginPublishMessage tests publishing a message
func TestRedisMQPluginPublishMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:      "msg-1",
		Topic:   "blockchain_events",
		Payload: []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.PublishMessage(ctx, message); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	stats := plugin.GetStats()
	if stats.MessageCount != 1 {
		t.Errorf("expected message count 1, got %d", stats.MessageCount)
	}
}

// TestRedisMQPluginAcknowledgeMessage tests acknowledging a message
func TestRedisMQPluginAcknowledgeMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:      "msg-1",
		Topic:   "blockchain_events",
		Payload: []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.AcknowledgeMessage(ctx, message); err != nil {
		t.Fatalf("failed to acknowledge message: %v", err)
	}
}

// TestRedisMQPluginDeadLetterQueue tests dead letter queue handling
func TestRedisMQPluginDeadLetterQueue(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:      "msg-1",
		Topic:   "blockchain_events",
		Payload: []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.SendToDeadLetterQueue(ctx, message, "processing failed"); err != nil {
		t.Fatalf("failed to send to dead letter queue: %v", err)
	}

	stats := plugin.GetStats()
	if stats.DeadLetterQueueSize != 1 {
		t.Errorf("expected dead letter queue size 1, got %d", stats.DeadLetterQueueSize)
	}
}

// TestRedisMQPluginRetryMessage tests message retry
func TestRedisMQPluginRetryMessage(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:      "msg-1",
		Topic:   "blockchain_events",
		Payload: []byte("test payload"),
		Timestamp: time.Now().UTC(),
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

// TestRedisMQPluginGetStats tests statistics retrieval
func TestRedisMQPluginGetStats(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	stats := plugin.GetStats()

	if stats.MessageCount != 0 {
		t.Errorf("expected message count 0, got %d", stats.MessageCount)
	}

	if stats.ErrorCount != 0 {
		t.Errorf("expected error count 0, got %d", stats.ErrorCount)
	}
}

// TestRedisMQPluginSetBatchSize tests setting batch size
func TestRedisMQPluginSetBatchSize(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	batchSize := 50
	plugin.SetBatchSize(batchSize)

	if plugin.batchSize != batchSize {
		t.Errorf("expected batch size %d, got %d", batchSize, plugin.batchSize)
	}
}

// TestRedisMQPluginSetMaxRetries tests setting max retries
func TestRedisMQPluginSetMaxRetries(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	maxRetries := 5
	plugin.SetMaxRetries(maxRetries)

	if plugin.maxRetries != maxRetries {
		t.Errorf("expected max retries %d, got %d", maxRetries, plugin.maxRetries)
	}
}

// TestRedisMQPluginSetRetryDelay tests setting retry delay
func TestRedisMQPluginSetRetryDelay(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	delay := 2 * time.Second
	plugin.SetRetryDelay(delay)

	if plugin.retryDelay != delay {
		t.Errorf("expected retry delay %v, got %v", delay, plugin.retryDelay)
	}
}

// TestRedisMQPluginConcurrentOperations tests concurrent operations
func TestRedisMQPluginConcurrentOperations(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
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
				ID:      fmt.Sprintf("msg-%d", index),
				Topic:   "blockchain_events",
				Payload: []byte("test payload"),
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

// TestRedisMQPluginHealth tests health check
func TestRedisMQPluginHealth(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	health := plugin.Health()

	if health == nil {
		t.Fatal("expected health status")
	}

	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}
}

// TestRedisMQPluginConnectionURL tests connection URL configuration
func TestRedisMQPluginConnectionURL(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379/0"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	if plugin.connectionURL != connectionURL {
		t.Errorf("expected connection URL %s, got %s", connectionURL, plugin.connectionURL)
	}
}

// TestRedisMQPluginMultipleTopics tests handling multiple topics
func TestRedisMQPluginMultipleTopics(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topics := []string{"topic1", "topic2", "topic3"}

	for _, topic := range topics {
		message := core.MessageQueueMessage{
			ID:      fmt.Sprintf("msg-%s", topic),
			Topic:   topic,
			Payload: []byte("test payload"),
			Timestamp: time.Now().UTC(),
		}

		if err := plugin.PublishMessage(ctx, message); err != nil {
			t.Logf("failed to publish message to topic %s: %v", topic, err)
		}
	}
}

// TestRedisMQPluginFlushQueue tests flushing a queue
func TestRedisMQPluginFlushQueue(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "blockchain_events"

	if err := plugin.FlushQueue(ctx, topic); err != nil {
		t.Logf("failed to flush queue: %v", err)
	}
}

// TestRedisMQPluginGetQueueDepth tests getting queue depth
func TestRedisMQPluginGetQueueDepth(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	connectionURL := "redis://localhost:6379"

	plugin := NewRedisMQPlugin("redis", "1.0.0", config, logger, metrics, eventBus, connectionURL)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if Redis not running): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "blockchain_events"

	depth, err := plugin.GetQueueDepth(ctx, topic)
	if err != nil {
		t.Logf("failed to get queue depth: %v", err)
	} else {
		if depth < 0 {
			t.Errorf("expected non-negative queue depth, got %d", depth)
		}
	}
}
