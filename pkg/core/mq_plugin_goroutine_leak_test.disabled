package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestMQPluginConsumeMessagesNoGoroutineLeaks tests that ConsumeMessages doesn't leak goroutines
func TestMQPluginConsumeMessagesNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		// Create a context that we'll cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Run ConsumeMessages in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- plugin.ConsumeMessages(ctx, "test-topic", func(msg MessageQueueMessage) error {
				return nil
			})
		}()

		// Give it a moment to start
		time.Sleep(50 * time.Millisecond)

		// Cancel the context to stop consuming
		cancel()

		// Wait for the goroutine to finish
		select {
		case <-done:
			// Expected
		case <-time.After(2 * time.Second):
			t.Fatal("ConsumeMessages did not exit after context cancellation")
		}

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginPublishMessageNoGoroutineLeaks tests that PublishMessage doesn't leak goroutines
func TestMQPluginPublishMessageNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx := context.Background()

		// Publish multiple messages
		for i := 0; i < 10; i++ {
			msg := MessageQueueMessage{
				ID:      fmt.Sprintf("msg-%d", i),
				Topic:   "test-topic",
				Payload: []byte(fmt.Sprintf("payload-%d", i)),
			}

			if err := plugin.PublishMessage(ctx, msg); err != nil {
				t.Fatalf("failed to publish message: %v", err)
			}
		}

		// Wait for in-flight operations to complete
		plugin.inFlightWaitGroup.Wait()

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginRetryMessageNoGoroutineLeaks tests that RetryMessage doesn't leak goroutines
func TestMQPluginRetryMessageNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)
		plugin.SetRetryDelay(10 * time.Millisecond) // Short delay for testing

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx := context.Background()

		msg := MessageQueueMessage{
			ID:      "test-msg",
			Topic:   "test-topic",
			Payload: []byte("test-payload"),
		}

		// Retry the message
		if err := plugin.RetryMessage(ctx, msg); err != nil {
			t.Fatalf("failed to retry message: %v", err)
		}

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginRetryMessageContextCancellationNoGoroutineLeaks tests that RetryMessage properly handles context cancellation
func TestMQPluginRetryMessageContextCancellationNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)
		plugin.SetRetryDelay(1 * time.Second) // Long delay to test cancellation

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		msg := MessageQueueMessage{
			ID:      "test-msg",
			Topic:   "test-topic",
			Payload: []byte("test-payload"),
		}

		// Start retry in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- plugin.RetryMessage(ctx, msg)
		}()

		// Give it a moment to start the delay
		time.Sleep(50 * time.Millisecond)

		// Cancel the context
		cancel()

		// Wait for the retry to complete
		select {
		case <-done:
			// Expected
		case <-time.After(2 * time.Second):
			t.Fatal("RetryMessage did not exit after context cancellation")
		}

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginProcessBatchNoGoroutineLeaks tests that ProcessBatch doesn't leak goroutines
func TestMQPluginProcessBatchNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx := context.Background()

		// Add messages to batch
		for i := 0; i < 5; i++ {
			msg := MessageQueueMessage{
				ID:      fmt.Sprintf("msg-%d", i),
				Topic:   "test-topic",
				Payload: []byte(fmt.Sprintf("payload-%d", i)),
			}
			if err := plugin.AddToBatch(msg); err != nil {
				t.Fatalf("failed to add to batch: %v", err)
			}
		}

		// Process batch
		if err := plugin.ProcessBatch(ctx, func(messages []MessageQueueMessage) error {
			return nil
		}); err != nil {
			t.Fatalf("failed to process batch: %v", err)
		}

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginConcurrentPublishNoGoroutineLeaks tests concurrent publish operations don't leak goroutines
func TestMQPluginConcurrentPublishNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx := context.Background()

		// Publish messages concurrently
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				msg := MessageQueueMessage{
					ID:      fmt.Sprintf("msg-%d", index),
					Topic:   "test-topic",
					Payload: []byte(fmt.Sprintf("payload-%d", index)),
				}
				if err := plugin.PublishMessage(ctx, msg); err != nil {
					t.Errorf("failed to publish message: %v", err)
				}
			}(i)
		}

		wg.Wait()

		// Wait for in-flight operations to complete
		plugin.inFlightWaitGroup.Wait()

		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginLifecycleNoGoroutineLeaks tests full lifecycle doesn't leak goroutines
func TestMQPluginLifecycleNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		// Initialize
		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		// Start
		if err := plugin.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		ctx := context.Background()

		// Publish messages
		for i := 0; i < 5; i++ {
			msg := MessageQueueMessage{
				ID:      fmt.Sprintf("msg-%d", i),
				Topic:   "test-topic",
				Payload: []byte(fmt.Sprintf("payload-%d", i)),
			}
			if err := plugin.PublishMessage(ctx, msg); err != nil {
				t.Fatalf("failed to publish message: %v", err)
			}
		}

		// Acknowledge messages
		for i := 0; i < 5; i++ {
			msg := MessageQueueMessage{
				ID:     fmt.Sprintf("msg-%d", i),
				Topic:  "test-topic",
				Offset: int64(i),
			}
			if err := plugin.AcknowledgeMessage(ctx, msg); err != nil {
				t.Fatalf("failed to acknowledge message: %v", err)
			}
		}

		// Wait for in-flight operations
		plugin.inFlightWaitGroup.Wait()

		// Stop
		if err := plugin.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	})
}

// TestMQPluginMultipleStartStopNoGoroutineLeaks tests multiple start/stop cycles don't leak goroutines
func TestMQPluginMultipleStartStopNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	WithGoroutineLeakDetection(t, func() {
		config := Config{
			BlockchainNodeURL: "localhost:50051",
			StartBlock:        0,
		}
		logger := NewDefaultLogger(LogLevelInfo)
		metrics := NewDefaultMetricsCollector()
		eventBus := NewEventBus(nil)

		plugin := NewBaseMQPlugin("kafka", "1.0.0", config, logger, metrics, eventBus)

		if err := plugin.Initialize(); err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}

		// Multiple start/stop cycles
		for i := 0; i < 3; i++ {
			if err := plugin.Start(); err != nil {
				t.Fatalf("failed to start (cycle %d): %v", i, err)
			}

			// Do some work
			ctx := context.Background()
			msg := MessageQueueMessage{
				ID:      fmt.Sprintf("msg-%d", i),
				Topic:   "test-topic",
				Payload: []byte(fmt.Sprintf("payload-%d", i)),
			}
			if err := plugin.PublishMessage(ctx, msg); err != nil {
				t.Fatalf("failed to publish message (cycle %d): %v", i, err)
			}

			// Wait for in-flight operations
			plugin.inFlightWaitGroup.Wait()

			if err := plugin.Stop(); err != nil {
				t.Fatalf("failed to stop (cycle %d): %v", i, err)
			}
		}
	})
}

// TestMQPluginGoroutineLeakDetectorAccuracy tests the accuracy of the goroutine leak detector in MQ plugin context
func TestMQPluginGoroutineLeakDetectorAccuracy(t *testing.T) {
	detector := NewGoroutineLeakDetector()
	initialCount := detector.GetInitialCount()

	// Create some goroutines
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()

	leaked := detector.Finish()

	if leaked != 0 {
		t.Errorf("expected no leaks, got %d", leaked)
	}

	// Verify goroutines were cleaned up
	if detector.GetFinalCount() > initialCount {
		t.Errorf("goroutines not cleaned up: initial=%d, final=%d", initialCount, detector.GetFinalCount())
	}
}

// TestGoroutineLeakDetectorDetectsLeaks tests that the detector can detect actual leaks
func TestGoroutineLeakDetectorDetectsLeaks(t *testing.T) {
	detector := NewGoroutineLeakDetector()

	// Create a goroutine that doesn't exit
	ch := make(chan bool)
	go func() {
		<-ch // Will block forever
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	leaked := detector.Finish()

	if leaked <= 0 {
		t.Errorf("expected to detect leak, got %d", leaked)
	}

	// Clean up
	close(ch)
}

// TestGoroutineLeakDetectorReport tests the leak detector report generation
func TestGoroutineLeakDetectorReport(t *testing.T) {
	detector := NewGoroutineLeakDetector()

	// Create some goroutines
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()

	detector.Finish()
	report := detector.GenerateReport()

	// LeakedCount should be <= 0 (negative or zero) since goroutines completed
	if report.LeakedCount > 0 {
		t.Errorf("expected no leaks in report, got %d", report.LeakedCount)
	}

	if report.InitialCount <= 0 {
		t.Errorf("expected positive initial count, got %d", report.InitialCount)
	}
}
