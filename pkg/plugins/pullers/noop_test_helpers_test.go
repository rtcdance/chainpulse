package pullers

import (
	"testing"
)

func TestNoopLogger_Debug(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	l.Debug("test message", "key", "value")
}

func TestNoopLogger_Info(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	l.Info("test message", "key", "value")
}

func TestNoopLogger_Warn(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	l.Warn("test message", "key", "value")
}

func TestNoopLogger_Error(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	l.Error("test message", "key", "value")
}

func TestNoopLogger_Fatal(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	l.Fatal("test message", "key", "value")
}

func TestNoopLogger_WithCorrelationID(t *testing.T) {
	t.Parallel()
	l := &noopLogger{}
	result := l.WithCorrelationID("corr-123")
	if result != l {
		t.Error("WithCorrelationID should return the same logger instance")
	}
}

func TestNoopMetrics_RecordCounter(t *testing.T) {
	t.Parallel()
	m := &noopMetrics{}
	m.RecordCounter("test_counter", 1, nil)
}

func TestNoopMetrics_RecordGauge(t *testing.T) {
	t.Parallel()
	m := &noopMetrics{}
	m.RecordGauge("test_gauge", 42.0, nil)
}

func TestNoopMetrics_RecordHistogram(t *testing.T) {
	t.Parallel()
	m := &noopMetrics{}
	m.RecordHistogram("test_histogram", 3.14, nil)
}

func TestNoopMetrics_GetMetrics(t *testing.T) {
	t.Parallel()
	m := &noopMetrics{}
	metrics := m.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should return a non-nil map")
	}
	if len(metrics) != 0 {
		t.Errorf("GetMetrics should return an empty map, got %v", metrics)
	}
}
