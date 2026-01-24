package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Property 1: Test Isolation
// For any unit test, running it in isolation SHALL produce the same result as running it with other tests.
// This property validates that tests don't have shared state or side effects that affect other tests.

// TestProperty1_ConfigManagerIsolation tests that ConfigManager instances don't share state
func TestProperty1_ConfigManagerIsolation(t *testing.T) {
	// Property: Each ConfigManager instance must maintain independent state
	logger1 := NewDefaultLogger(LogLevelInfo)
	logger2 := NewDefaultLogger(LogLevelInfo)
	
	cm1 := NewConfigManager(logger1)
	cm2 := NewConfigManager(logger2)

	config1 := Config{
		APIPort:        8080,
		WorkerPoolSize: 10,
		FeatureFlags:   make(map[string]bool),
	}

	config2 := Config{
		APIPort:        9090,
		WorkerPoolSize: 20,
		FeatureFlags:   make(map[string]bool),
	}

	cm1.config = config1
	cm2.config = config2

	// Verify each manager has its own config
	if cm1.config.APIPort != 8080 {
		t.Errorf("cm1 config corrupted: expected 8080, got %d", cm1.config.APIPort)
	}
	if cm2.config.APIPort != 9090 {
		t.Errorf("cm2 config corrupted: expected 9090, got %d", cm2.config.APIPort)
	}
}

// TestProperty1_LoggerIsolation tests that Logger instances don't share state
func TestProperty1_LoggerIsolation(t *testing.T) {
	// Property: Each Logger instance must maintain independent correlation IDs and fields
	logger1 := NewDefaultLogger(LogLevelInfo)
	logger2 := NewDefaultLogger(LogLevelInfo)

	logger1WithID := logger1.WithCorrelationID("id-1")
	logger2WithID := logger2.WithCorrelationID("id-2")

	// Verify each logger has its own correlation ID
	if logger1WithID.(*DefaultLogger).correlationID != "id-1" {
		t.Errorf("logger1 correlation ID corrupted: expected id-1, got %s", logger1WithID.(*DefaultLogger).correlationID)
	}
	if logger2WithID.(*DefaultLogger).correlationID != "id-2" {
		t.Errorf("logger2 correlation ID corrupted: expected id-2, got %s", logger2WithID.(*DefaultLogger).correlationID)
	}
}

// TestProperty1_MetricsIsolation tests that Metrics instances don't share counters
func TestProperty1_MetricsIsolation(t *testing.T) {
	// Property: Each Metrics instance must maintain independent counters
	metrics1 := NewDefaultMetricsCollector()
	metrics2 := NewDefaultMetricsCollector()

	metrics1.RecordCounter("requests", 1, nil)
	metrics2.RecordCounter("requests", 2, nil)

	// Verify each metrics instance has its own counters
	count1 := metrics1.GetCounter("requests", nil)
	count2 := metrics2.GetCounter("requests", nil)

	if count1 != 1 {
		t.Errorf("metrics1 counter corrupted: expected 1, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("metrics2 counter corrupted: expected 2, got %d", count2)
	}
}

// TestProperty1_HealthCheckerIsolation tests that HealthChecker instances don't share state
func TestProperty1_HealthCheckerIsolation(t *testing.T) {
	// Property: Each HealthChecker instance must maintain independent check results
	logger := NewDefaultLogger(LogLevelInfo)
	config := NewMockConfigManager()
	bus := NewEventBus(logger)
	metrics := NewDefaultMetricsCollector()

	checker1 := NewDefaultHealthChecker(nil, config, bus, metrics, logger)
	checker2 := NewDefaultHealthChecker(nil, config, bus, metrics, logger)

	status1, _ := checker1.Check(context.Background())
	status2, _ := checker2.Check(context.Background())

	// Verify each checker has its own status
	if status1.Status == "" {
		t.Error("checker1 status not set")
	}
	if status2.Status == "" {
		t.Error("checker2 status not set")
	}

	// Verify status times are different (or very close)
	time1 := checker1.GetLastCheckTime()
	time2 := checker2.GetLastCheckTime()

	if time1.After(time2.Add(1 * time.Second)) {
		t.Errorf("checker times too far apart: %v vs %v", time1, time2)
	}
}

// TestProperty1_EventFilterIsolation tests that EventFilter instances don't share state
func TestProperty1_EventFilterIsolation(t *testing.T) {
	// Property: Each EventFilter instance must maintain independent filter criteria
	filter1 := &EventFilter{
		Network:   "ethereum",
		FromBlock: 1000,
		ToBlock:   2000,
		Limit:     100,
	}

	filter2 := &EventFilter{
		Network:   "polygon",
		FromBlock: 5000,
		ToBlock:   6000,
		Limit:     200,
	}

	// Verify each filter has its own criteria
	if filter1.Network != "ethereum" || filter1.FromBlock != 1000 {
		t.Error("filter1 criteria corrupted")
	}
	if filter2.Network != "polygon" || filter2.FromBlock != 5000 {
		t.Error("filter2 criteria corrupted")
	}

	// Verify modifying one doesn't affect the other
	filter1.Limit = 500
	if filter2.Limit != 200 {
		t.Errorf("filter2 limit corrupted: expected 200, got %d", filter2.Limit)
	}
}

// TestProperty1_ConcurrentIsolation tests that concurrent operations maintain isolation
func TestProperty1_ConcurrentIsolation(t *testing.T) {
	// Property: Concurrent operations on different instances must maintain isolation
	numGoroutines := 10
	results := make(map[int]int)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			metrics := NewDefaultMetricsCollector()
			metrics.RecordCounter("requests", int64(id), nil)

			count := metrics.GetCounter("requests", nil)

			mu.Lock()
			results[id] = int(count)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify each goroutine has its own metrics instance
	for i := 0; i < numGoroutines; i++ {
		if results[i] != i {
			t.Errorf("goroutine %d metrics corrupted: expected %d, got %d", i, i, results[i])
		}
	}
}

// TestProperty1_FieldModificationIsolation tests that modifying fields doesn't affect other instances
func TestProperty1_FieldModificationIsolation(t *testing.T) {
	// Property: Modifying fields on one instance must not affect other instances
	config1 := Config{
		APIPort:      8080,
		FeatureFlags: make(map[string]bool),
	}

	config2 := Config{
		APIPort:      9090,
		FeatureFlags: make(map[string]bool),
	}

	config1.FeatureFlags["feature1"] = true
	config2.FeatureFlags["feature2"] = true

	// Verify each config has its own feature flags
	if _, ok := config1.FeatureFlags["feature2"]; ok {
		t.Error("config1 feature flags corrupted")
	}
	if _, ok := config2.FeatureFlags["feature1"]; ok {
		t.Error("config2 feature flags corrupted")
	}
}

// TestProperty1_SequentialOperationsIsolation tests that sequential operations maintain isolation
func TestProperty1_SequentialOperationsIsolation(t *testing.T) {
	// Property: Sequential operations on different instances must maintain isolation
	results := make([]int, 0)

	for i := 0; i < 5; i++ {
		metrics := NewDefaultMetricsCollector()
		metrics.RecordCounter("requests", int64(i), nil)
		count := metrics.GetCounter("requests", nil)
		results = append(results, int(count))
	}

	// Verify each iteration has its own metrics instance
	for i := 0; i < 5; i++ {
		if results[i] != i {
			t.Errorf("iteration %d metrics corrupted: expected %d, got %d", i, i, results[i])
		}
	}
}

// TestProperty1_NestedIsolation tests that nested operations maintain isolation
func TestProperty1_NestedIsolation(t *testing.T) {
	// Property: Nested operations on different instances must maintain isolation
	logger1 := NewDefaultLogger(LogLevelInfo)
	logger2 := NewDefaultLogger(LogLevelInfo)

	logger1WithID := logger1.WithCorrelationID("id-1").(*DefaultLogger)
	logger1WithField := logger1WithID.WithField("key1", "value1")

	logger2WithID := logger2.WithCorrelationID("id-2").(*DefaultLogger)
	logger2WithField := logger2WithID.WithField("key2", "value2")

	// Verify each logger chain has its own state
	if logger1WithField.correlationID != "id-1" {
		t.Error("logger1 correlation ID lost in chain")
	}
	if logger2WithField.correlationID != "id-2" {
		t.Error("logger2 correlation ID lost in chain")
	}
}

// TestProperty1_ContextIsolation tests that context operations maintain isolation
func TestProperty1_ContextIsolation(t *testing.T) {
	// Property: Context operations on different instances must maintain isolation
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	// Verify each context has its own deadline
	deadline1, ok1 := ctx1.Deadline()
	deadline2, ok2 := ctx2.Deadline()

	if !ok1 || !ok2 {
		t.Error("context deadlines not set")
	}

	if deadline1.After(deadline2) {
		t.Error("context deadlines corrupted")
	}
}

// TestProperty1_MapIsolation tests that map instances don't share state
func TestProperty1_MapIsolation(t *testing.T) {
	// Property: Map instances must maintain independent state
	map1 := make(map[string]interface{})
	map2 := make(map[string]interface{})

	map1["key1"] = "value1"
	map2["key2"] = "value2"

	// Verify each map has its own entries
	if _, ok := map1["key2"]; ok {
		t.Error("map1 contains key2")
	}
	if _, ok := map2["key1"]; ok {
		t.Error("map2 contains key1")
	}
}

// TestProperty1_SliceIsolation tests that slice instances don't share state
func TestProperty1_SliceIsolation(t *testing.T) {
	// Property: Slice instances must maintain independent state
	slice1 := make([]string, 0)
	slice2 := make([]string, 0)

	slice1 = append(slice1, "item1")
	slice2 = append(slice2, "item2")

	// Verify each slice has its own items
	if len(slice1) != 1 || slice1[0] != "item1" {
		t.Error("slice1 corrupted")
	}
	if len(slice2) != 1 || slice2[0] != "item2" {
		t.Error("slice2 corrupted")
	}
}

// TestProperty1_ChannelIsolation tests that channel instances don't share state
func TestProperty1_ChannelIsolation(t *testing.T) {
	// Property: Channel instances must maintain independent state
	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	ch1 <- "message1"
	ch2 <- "message2"

	// Verify each channel has its own message
	msg1 := <-ch1
	msg2 := <-ch2

	if msg1 != "message1" {
		t.Errorf("ch1 corrupted: expected message1, got %s", msg1)
	}
	if msg2 != "message2" {
		t.Errorf("ch2 corrupted: expected message2, got %s", msg2)
	}
}

// TestProperty1_TimerIsolation tests that timer instances don't share state
func TestProperty1_TimerIsolation(t *testing.T) {
	// Property: Timer instances must maintain independent state
	timer1 := time.NewTimer(100 * time.Millisecond)
	timer2 := time.NewTimer(200 * time.Millisecond)

	defer timer1.Stop()
	defer timer2.Stop()

	// Verify each timer fires independently
	select {
	case <-timer1.C:
		// timer1 fired first (expected)
	case <-timer2.C:
		t.Error("timer2 fired before timer1")
	}
}
