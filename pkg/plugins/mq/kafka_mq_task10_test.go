package mq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// MockMetricsCollector for testing
type MockMetricsCollector struct {
	metrics map[string]int64
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string]int64),
	}
}

func (m *MockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.metrics[name] += value
}

func (m *MockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.metrics[name] = int64(value)
}

func (m *MockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.metrics[name] = int64(value)
}

func (m *MockMetricsCollector) GetMetrics() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

// MockLogger for testing
type MockLogger struct {
	logs []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]string, 0),
	}
}

func (m *MockLogger) Debug(message string, fields ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("DEBUG: %s", message))
}

func (m *MockLogger) Info(message string, fields ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("INFO: %s", message))
}

func (m *MockLogger) Error(message string, fields ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("ERROR: %s", message))
}

func (m *MockLogger) Warn(message string, fields ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("WARN: %s", message))
}

func (m *MockLogger) Fatal(message string, fields ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf("FATAL: %s", message))
}

func (m *MockLogger) WithCorrelationID(id string) core.Logger {
	return m
}

// TestKafkaOffsetPersistence tests offset persistence functionality
func TestKafkaOffsetPersistence(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Test persisting offsets
	_ = plugin.PersistOffset("test-topic", 0, 100)

	// Test retrieving persisted offset
	offset, err := plugin.GetPersistedOffset("test-topic", 0)
	if err != nil {
		t.Fatalf("failed to get persisted offset: %v", err)
	}

	if offset != 100 {
		t.Errorf("expected offset 100, got %d", offset)
	}

	// Test retrieving non-existent offset
	_, err = plugin.GetPersistedOffset("non-existent", 0)
	if err == nil {
		t.Errorf("expected error for non-existent offset, got nil")
	}

	// Test persisting multiple offsets for same topic
	_ = plugin.PersistOffset("test-topic", 1, 200)

	offset, err = plugin.GetPersistedOffset("test-topic", 1)
	if err != nil {
		t.Fatalf("failed to get second persisted offset: %v", err)
	}

	if offset != 200 {
		t.Errorf("expected offset 200, got %d", offset)
	}
}

// TestKafkaConsumerGroupMetrics tests consumer group metrics
func TestKafkaConsumerGroupMetrics(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Test updating metrics
	plugin.UpdateConsumerGroupMetric("active_members", 5)
	plugin.UpdateConsumerGroupMetric("lag", 1000)

	// Test retrieving metrics
	metrics := plugin.GetConsumerGroupMetrics()
	if metrics["active_members"] != 5 {
		t.Errorf("expected active_members 5, got %d", metrics["active_members"])
	}

	if metrics["lag"] != 1000 {
		t.Errorf("expected lag 1000, got %d", metrics["lag"])
	}

	// Test updating existing metric
	plugin.UpdateConsumerGroupMetric("active_members", 3)
	metrics = plugin.GetConsumerGroupMetrics()
	if metrics["active_members"] != 3 {
		t.Errorf("expected active_members 3 after update, got %d", metrics["active_members"])
	}
}

// TestKafkaBrokerFailureRecovery tests broker failure and recovery tracking
func TestKafkaBrokerFailureRecovery(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Test initial state
	if plugin.GetBrokerFailureCount() != 0 {
		t.Errorf("expected initial failure count 0, got %d", plugin.GetBrokerFailureCount())
	}

	if plugin.GetBrokerRecoveryCount() != 0 {
		t.Errorf("expected initial recovery count 0, got %d", plugin.GetBrokerRecoveryCount())
	}

	// Test recording failures
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerFailure()

	if plugin.GetBrokerFailureCount() != 2 {
		t.Errorf("expected failure count 2, got %d", plugin.GetBrokerFailureCount())
	}

	// Test recording recovery
	plugin.RecordBrokerRecovery()

	if plugin.GetBrokerRecoveryCount() != 1 {
		t.Errorf("expected recovery count 1, got %d", plugin.GetBrokerRecoveryCount())
	}
}

// TestExponentialBackoffDelay tests exponential backoff calculation
func TestExponentialBackoffDelay(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Set base retry delay to 100ms for testing
	plugin.SetRetryDelay(100 * time.Millisecond)

	tests := []struct {
		retryCount  int
		expectedMs  int64
		description string
	}{
		{0, 100, "retry 0 should use base delay"},
		{1, 100, "retry 1: 100ms * 2^0 = 100ms"},
		{2, 200, "retry 2: 100ms * 2^1 = 200ms"},
		{3, 400, "retry 3: 100ms * 2^2 = 400ms"},
		{4, 800, "retry 4: 100ms * 2^3 = 800ms"},
		{5, 1600, "retry 5: 100ms * 2^4 = 1600ms"},
	}

	for _, test := range tests {
		delay := plugin.CalculateExponentialBackoffDelay(test.retryCount)
		expectedDelay := time.Duration(test.expectedMs) * time.Millisecond

		if delay != expectedDelay {
			t.Errorf("%s: expected %v, got %v", test.description, expectedDelay, delay)
		}
	}
}

// TestKafkaSpecificMetrics tests Kafka-specific metrics collection
func TestKafkaSpecificMetrics(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092", "localhost:9093"},
		"test-group",
	)

	// Record some activity
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerRecovery()
	plugin.UpdateConsumerGroupMetric("active_members", 3)
	_ = plugin.PersistOffset("topic1", 0, 100)
	_ = plugin.PersistOffset("topic1", 1, 200)

	// Get metrics
	metrics := plugin.GetKafkaSpecificMetrics()

	// Verify metrics
	if metrics["brokers"] == nil {
		t.Errorf("expected brokers in metrics")
	}

	if metrics["consumer_group"] != "test-group" {
		t.Errorf("expected consumer_group test-group, got %v", metrics["consumer_group"])
	}

	if metrics["broker_failures"] != int64(1) {
		t.Errorf("expected broker_failures 1, got %v", metrics["broker_failures"])
	}

	if metrics["broker_recoveries"] != int64(1) {
		t.Errorf("expected broker_recoveries 1, got %v", metrics["broker_recoveries"])
	}

	if metrics["active_consumers"] != 0 {
		t.Errorf("expected active_consumers 0, got %v", metrics["active_consumers"])
	}
}

// TestConsumerGroupStatus tests consumer group status reporting
func TestConsumerGroupStatus(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Initialize and start plugin
	_ = plugin.Initialize()
	_ = plugin.Start()

	// Get status
	status := plugin.GetConsumerGroupStatus()

	if status["consumer_group"] != "test-group" {
		t.Errorf("expected consumer_group test-group, got %v", status["consumer_group"])
	}

	if status["is_running"] != true {
		t.Errorf("expected is_running true, got %v", status["is_running"])
	}

	if status["message_count"] != int64(0) {
		t.Errorf("expected message_count 0, got %v", status["message_count"])
	}
}

// TestOffsetPersistenceStats tests offset persistence statistics
func TestOffsetPersistenceStats(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Persist some offsets
	_ = plugin.PersistOffset("topic1", 0, 100)
	_ = plugin.PersistOffset("topic1", 1, 200)
	_ = plugin.PersistOffset("topic2", 0, 300)

	// Get stats
	stats := plugin.GetOffsetPersistenceStats()

	// Verify the stats are present and have expected values
	if topicsCount, ok := stats["topics_with_persisted_offsets"]; ok {
		if topicsCount != int64(2) {
			t.Errorf("expected 2 topics with persisted offsets, got %v", topicsCount)
		}
	}

	if totalOffsets, ok := stats["total_persisted_offsets"]; ok {
		if totalOffsets != int64(3) {
			t.Errorf("expected 3 total persisted offsets, got %v", totalOffsets)
		}
	}
}

// TestRebalanceConsumerGroup tests consumer group rebalancing
func TestRebalanceConsumerGroup(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Initialize and start plugin
	_ = plugin.Initialize()
	_ = plugin.Start()

	// Rebalance should succeed
	err := plugin.RebalanceConsumerGroup(context.Background())
	if err != nil {
		t.Fatalf("failed to rebalance consumer group: %v", err)
	}

	// Rebalance when not running should fail
	_ = plugin.Stop() // nolint:errcheck
	err = plugin.RebalanceConsumerGroup(context.Background())
	if err == nil {
		t.Errorf("expected error when rebalancing stopped plugin, got nil")
	}
}

// TestConcurrentOffsetPersistence tests concurrent offset persistence
func TestConcurrentOffsetPersistence(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Persist offsets concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				topic := fmt.Sprintf("topic-%d", id)
				partition := int32(j)
				offset := int64(id*10 + j)
				_ = plugin.PersistOffset(topic, partition, offset)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all offsets were persisted
	stats := plugin.GetOffsetPersistenceStats()
	if topicsCount, ok := stats["topics_with_persisted_offsets"]; ok {
		if topicsCount != int64(10) {
			t.Errorf("expected 10 topics, got %v", topicsCount)
		}
	}

	if totalOffsets, ok := stats["total_persisted_offsets"]; ok {
		if totalOffsets != int64(100) {
			t.Errorf("expected 100 total offsets, got %v", totalOffsets)
		}
	}
}

// TestConcurrentMetricsUpdates tests concurrent consumer group metrics updates
func TestConcurrentMetricsUpdates(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Update metrics concurrently
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				plugin.UpdateConsumerGroupMetric("concurrent_updates", int64(id*5+j+1))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify metrics were updated (last value should be set)
	metrics := plugin.GetConsumerGroupMetrics()
	if concurrentUpdates, ok := metrics["concurrent_updates"]; ok {
		// Just verify that a value was set (not 0)
		if concurrentUpdates == 0 {
			t.Errorf("expected concurrent_updates to be set, got 0")
		}
	}
}

// TestBrokerFailureRecoverySequence tests a sequence of failures and recoveries
func TestBrokerFailureRecoverySequence(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092"},
		"test-group",
	)

	// Simulate failure sequence
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerRecovery()
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerRecovery()
	plugin.RecordBrokerRecovery()

	if plugin.GetBrokerFailureCount() != 3 {
		t.Errorf("expected 3 failures, got %d", plugin.GetBrokerFailureCount())
	}

	if plugin.GetBrokerRecoveryCount() != 3 {
		t.Errorf("expected 3 recoveries, got %d", plugin.GetBrokerRecoveryCount())
	}
}

// TestKafkaSpecificFeaturesIntegration tests integration of all Kafka-specific features
func TestKafkaSpecificFeaturesIntegration(t *testing.T) {
	plugin := NewKafkaMQPlugin(
		"kafka-test",
		"1.0.0",
		nil,
		NewMockLogger(),
		NewMockMetricsCollector(),
		nil,
		[]string{"localhost:9092", "localhost:9093"},
		"test-group",
	)

	// Initialize and start
	_ = plugin.Initialize()
	_ = plugin.Start()

	// Simulate broker activity
	plugin.RecordBrokerFailure()
	plugin.RecordBrokerRecovery()

	// Persist offsets
	_ = plugin.PersistOffset("events", 0, 1000)
	_ = plugin.PersistOffset("events", 1, 2000)

	// Update consumer group metrics
	plugin.UpdateConsumerGroupMetric("active_members", 3)
	plugin.UpdateConsumerGroupMetric("lag", 500)

	// Get comprehensive metrics
	metrics := plugin.GetKafkaSpecificMetrics()
	status := plugin.GetConsumerGroupStatus()
	stats := plugin.GetOffsetPersistenceStats()

	// Verify all components
	if metrics["broker_failures"] != int64(1) {
		t.Errorf("expected 1 broker failure in metrics")
	}

	if status["consumer_group"] != "test-group" {
		t.Errorf("expected consumer_group in status")
	}

	if stats["total_persisted_offsets"] != int64(2) {
		t.Errorf("expected 2 persisted offsets in stats")
	}

	// Cleanup
	_ = plugin.Stop() // nolint:errcheck
}
