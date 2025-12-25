package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
}

func TestMetricsIncrementBlocksProcessed(t *testing.T) {
	m := NewMetrics()
	
	initialValue := testutil.ToFloat64(m.BlocksProcessedTotal)
	m.IncrementBlocksProcessed()
	
	finalValue := testutil.ToFloat64(m.BlocksProcessedTotal)
	if finalValue <= initialValue {
		t.Errorf("IncrementBlocksProcessed() did not increment the counter, initial: %f, final: %f", initialValue, finalValue)
	}
}

func TestMetricsIncrementEventsProcessed(t *testing.T) {
	m := NewMetrics()
	
	initialValue := testutil.ToFloat64(m.EventsProcessedTotal)
	m.IncrementEventsProcessed()
	
	finalValue := testutil.ToFloat64(m.EventsProcessedTotal)
	if finalValue <= initialValue {
		t.Errorf("IncrementEventsProcessed() did not increment the counter, initial: %f, final: %f", initialValue, finalValue)
	}
}

func TestMetricsIncrementEventsIndexed(t *testing.T) {
	m := NewMetrics()
	
	initialValue := testutil.ToFloat64(m.EventsIndexedTotal)
	m.IncrementEventsIndexed()
	
	finalValue := testutil.ToFloat64(m.EventsIndexedTotal)
	if finalValue <= initialValue {
		t.Errorf("IncrementEventsIndexed() did not increment the counter, initial: %f, final: %f", initialValue, finalValue)
	}
}

func TestMetricsIncrementCacheHit(t *testing.T) {
	m := NewMetrics()
	
	initialValue := testutil.ToFloat64(m.EventsCacheHitsTotal)
	m.IncrementCacheHit()
	
	finalValue := testutil.ToFloat64(m.EventsCacheHitsTotal)
	if finalValue <= initialValue {
		t.Errorf("IncrementCacheHit() did not increment the counter, initial: %f, final: %f", initialValue, finalValue)
	}
}

func TestMetricsIncrementCacheMiss(t *testing.T) {
	m := NewMetrics()
	
	initialValue := testutil.ToFloat64(m.EventsCacheMissesTotal)
	m.IncrementCacheMiss()
	
	finalValue := testutil.ToFloat64(m.EventsCacheMissesTotal)
	if finalValue <= initialValue {
		t.Errorf("IncrementCacheMiss() did not increment the counter, initial: %f, final: %f", initialValue, finalValue)
	}
}

func TestMetricsAPIFunctions(t *testing.T) {
	m := NewMetrics()
	
	// Test that these functions can be called without errors
	m.RecordAPIRequest("GET", "/test", "200")
	m.RecordAPIRequestDuration("GET", "/test", 0.1)
	m.SetActiveConnections(5)
	
	// Basic validation that the functions executed without panicking
	// The actual metrics collection would be validated in integration tests
}

func TestMetricsDatabaseFunctions(t *testing.T) {
	m := NewMetrics()
	
	// Test that these functions can be called without errors
	m.RecordDatabaseQueryDuration("SELECT", "events", 0.05)
	m.SetDatabaseConnections(10)
	
	// Basic validation that the functions executed without panicking
}

func TestMetricsErrorFunction(t *testing.T) {
	m := NewMetrics()
	
	// Test that this function can be called without errors
	m.IncrementError("test_component", "test_error")
	
	// Basic validation that the function executed without panicking
}