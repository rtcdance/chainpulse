package mq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestZeroMQMQPluginCreation tests ZeroMQ MQ plugin creation
func TestZeroMQMQPluginCreation(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	if plugin == nil {
		t.Fatal("expected plugin to be created")
	}

	if plugin.Name() != "zeromq" {
		t.Errorf("expected name 'zeromq', got %s", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", plugin.Version())
	}
}

// TestZeroMQMQPluginInitialization tests ZeroMQ plugin initialization
func TestZeroMQMQPluginInitialization(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	// Note: This will fail if ZeroMQ is not properly configured, but we test the logic
	err := plugin.Initialize()
	if err != nil {
		// Expected if ZeroMQ is not available
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
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

// TestZeroMQMQPluginLifecycle tests ZeroMQ plugin lifecycle
func TestZeroMQMQPluginLifecycle(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
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

// TestZeroMQMQPluginPublishMessage tests publishing a message
func TestZeroMQMQPluginPublishMessage(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.PublishMessage(ctx, message); err != nil {
		t.Logf("publish message failed (expected if ZeroMQ not available): %v", err)
	}
}

// TestZeroMQMQPluginAcknowledgeMessage tests acknowledging a message
func TestZeroMQMQPluginAcknowledgeMessage(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

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

// TestZeroMQMQPluginDeadLetterQueue tests dead letter queue handling
func TestZeroMQMQPluginDeadLetterQueue(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := core.MessageQueueMessage{
		ID:        "msg-1",
		Topic:     "blockchain_events",
		Payload:   []byte("test payload"),
		Timestamp: time.Now().UTC(),
	}

	if err := plugin.SendToDeadLetterQueue(ctx, message, "processing failed"); err != nil {
		t.Logf("send to DLQ failed (expected if ZeroMQ not available): %v", err)
	}

	stats := plugin.GetStats()
	if stats.DeadLetterQueueSize != 1 && err == nil {
		t.Errorf("expected dead letter queue size 1, got %d", stats.DeadLetterQueueSize)
	}
}

// TestZeroMQMQPluginRetryMessage tests message retry
func TestZeroMQMQPluginRetryMessage(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

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

	// Note: RetryMessage receives a copy of the message, so the original
	// message.RetryCount won't be modified. This is expected behavior.
	// The method logs the retry and updates metrics internally.
	// We just verify that the method succeeds without error.
}

// TestZeroMQMQPluginGetStats tests statistics retrieval
func TestZeroMQMQPluginGetStats(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
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

// TestZeroMQMQPluginSetBatchSize tests setting batch size
func TestZeroMQMQPluginSetBatchSize(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	batchSize := 50
	plugin.SetBatchSize(batchSize)

	if plugin.batchSize != batchSize {
		t.Errorf("expected batch size %d, got %d", batchSize, plugin.batchSize)
	}
}

// TestZeroMQMQPluginSetMaxRetries tests setting max retries
func TestZeroMQMQPluginSetMaxRetries(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	maxRetries := 5
	plugin.SetMaxRetries(maxRetries)

	if plugin.maxRetries != maxRetries {
		t.Errorf("expected max retries %d, got %d", maxRetries, plugin.maxRetries)
	}
}

// TestZeroMQMQPluginSetRetryDelay tests setting retry delay
func TestZeroMQMQPluginSetRetryDelay(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	delay := 2 * time.Second
	plugin.SetRetryDelay(delay)

	if plugin.retryDelay != delay {
		t.Errorf("expected retry delay %v, got %v", delay, plugin.retryDelay)
	}
}

// TestZeroMQMQPluginConcurrentOperations tests concurrent operations
func TestZeroMQMQPluginConcurrentOperations(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

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

// TestZeroMQMQPluginHealth tests health check
func TestZeroMQMQPluginHealth(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
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

// TestZeroMQMQPluginEndpointConfiguration tests endpoint configuration
func TestZeroMQMQPluginEndpointConfiguration(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	if plugin.GetEndpoint() != endpoint {
		t.Errorf("expected endpoint %s, got %s", endpoint, plugin.GetEndpoint())
	}

	newEndpoint := "tcp://127.0.0.1:5556"
	plugin.SetEndpoint(newEndpoint)

	if plugin.GetEndpoint() != newEndpoint {
		t.Errorf("expected endpoint %s, got %s", newEndpoint, plugin.GetEndpoint())
	}
}

// TestZeroMQMQPluginMultipleEndpoints tests handling multiple endpoints
func TestZeroMQMQPluginMultipleEndpoints(t *testing.T) {
	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)

	endpoints := []string{
		"tcp://127.0.0.1:5555",
		"tcp://127.0.0.1:5556",
		"tcp://127.0.0.1:5557",
	}

	for _, endpoint := range endpoints {
		plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

		if plugin.GetEndpoint() != endpoint {
			t.Errorf("expected endpoint %s, got %s", endpoint, plugin.GetEndpoint())
		}
	}
}

// TestZeroMQMQPluginGetDeadLetterQueueMessages tests retrieving DLQ messages
func TestZeroMQMQPluginGetDeadLetterQueueMessages(t *testing.T) {
	requireMQIntegration(t)

	config := &core.Config{
		BlockchainNodeURL: "localhost:50051",
		StartBlock:        0,
	}
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	eventBus := core.NewEventBus(logger)
	endpoint := "tcp://127.0.0.1:5555"

	plugin := NewZeroMQMQPlugin("zeromq", "1.0.0", config, logger, metrics, eventBus, endpoint)

	err := plugin.Initialize()
	if err != nil {
		t.Logf("initialization failed (expected if ZeroMQ not available): %v", err)
		return
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := plugin.GetDeadLetterQueueMessages(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get dead letter queue messages: %v", err)
	}

	if messages == nil {
		t.Fatal("expected messages slice")
	}
}
