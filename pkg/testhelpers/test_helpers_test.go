package testhelpers

import (
	"context"
	"testing"
	"time"
)

func TestNewTestLogger(t *testing.T) {
	t.Parallel()
	logger := NewTestLogger()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestGoroutineTracker(t *testing.T) {
	t.Parallel()
	gt := NewGoroutineTracker()
	if gt == nil {
		t.Fatal("expected non-nil tracker")
	}
	gt.Check()
	leak := gt.GetLeak()
	_ = leak
}

func TestNewTestMetricsCollector(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollector()
	if mc == nil {
		t.Fatal("expected non-nil metrics collector")
	}
}

func TestNewTestLoggerWithCapture(t *testing.T) {
	t.Parallel()
	l := NewTestLoggerWithCapture()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestTestLogger_AllLevels(t *testing.T) {
	t.Parallel()
	l := NewTestLoggerWithCapture()
	l.Debug("debug msg")
	l.Info("info msg")
	l.Warn("warn msg")
	l.Error("error msg")
	l.Fatal("fatal msg")
	msgs := l.GetMessages()
	if len(msgs) != 5 {
		t.Errorf("expected 5 messages, got %d", len(msgs))
	}
}

func TestTestLogger_WithCorrelationID(t *testing.T) {
	t.Parallel()
	l := NewTestLoggerWithCapture()
	result := l.WithCorrelationID("corr-1")
	if result == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestTestLogger_WithField(t *testing.T) {
	t.Parallel()
	l := NewTestLoggerWithCapture()
	result := l.WithField("key", "value")
	if result == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestTestLogger_LogMethods(t *testing.T) {
	t.Parallel()
	l := NewTestLoggerWithCapture()
	l.LogDebug("debug")
	l.LogInfo("info")
	l.LogWarn("warn")
	l.LogError("error")
	l.LogFatal("fatal")
	msgs := l.GetMessages()
	if len(msgs) != 5 {
		t.Errorf("expected 5 messages, got %d", len(msgs))
	}
}

func TestTestMetricsCollectorWithCapture(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	if mc == nil {
		t.Fatal("expected non-nil collector")
	}
}

func TestTestMetricsCollector_RecordCounter(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordCounter("requests", 1, nil)
	mc.RecordCounter("requests", 2, nil)
	if val := mc.GetCounter("requests"); val != 3 {
		t.Errorf("expected 3, got %d", val)
	}
}

func TestTestMetricsCollector_RecordGauge(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordGauge("memory", 1024.0, nil)
	if val := mc.GetGauge("memory"); val != 1024.0 {
		t.Errorf("expected 1024.0, got %f", val)
	}
}

func TestTestMetricsCollector_RecordHistogram(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordHistogram("latency", 100.0, nil)
	mc.RecordHistogram("latency", 200.0, nil)
	vals := mc.GetHistogram("latency")
	if len(vals) != 2 {
		t.Errorf("expected 2 values, got %d", len(vals))
	}
}

func TestTestMetricsCollector_GetMetrics(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordCounter("req", 5, nil)
	metrics := mc.GetMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
}

func TestTestMetricsCollector_Reset(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordCounter("req", 5, nil)
	mc.Reset()
	if val := mc.GetCounter("req"); val != 0 {
		t.Errorf("expected 0 after reset, got %d", val)
	}
}

func TestTestMetricsCollector_GetHistogramStats(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	stats := mc.GetHistogramStats("nonexistent")
	if stats.Count != 0 {
		t.Errorf("expected 0 count, got %d", stats.Count)
	}

	mc.RecordHistogram("latency", 100.0, nil)
	mc.RecordHistogram("latency", 200.0, nil)
	mc.RecordHistogram("latency", 300.0, nil)
	stats = mc.GetHistogramStats("latency")
	if stats.Count != 3 {
		t.Errorf("expected 3, got %d", stats.Count)
	}
	if stats.Min != 100.0 {
		t.Errorf("expected min 100.0, got %f", stats.Min)
	}
	if stats.Max != 300.0 {
		t.Errorf("expected max 300.0, got %f", stats.Max)
	}
}

func TestTestMetricsCollector_GetAllMetrics(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	all := mc.GetAllMetrics()
	if all == nil {
		t.Fatal("expected non-nil metrics")
	}
}

func TestTestMetricsCollector_RecordLatency(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordLatency("api_call", 150*time.Millisecond, nil)
	vals := mc.GetHistogram("api_call")
	if len(vals) != 1 {
		t.Errorf("expected 1 value, got %d", len(vals))
	}
}

func TestTestMetricsCollector_RecordError(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordError("errors", nil)
	if val := mc.GetCounter("errors"); val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}

func TestTestMetricsCollector_RecordSuccess(t *testing.T) {
	t.Parallel()
	mc := NewTestMetricsCollectorWithCapture()
	mc.RecordSuccess("success", nil)
	if val := mc.GetCounter("success"); val != 1 {
		t.Errorf("expected 1, got %d", val)
	}
}

func TestMockCache_GetSetDelete(t *testing.T) {
	t.Parallel()
	cache := NewMockCache()
	ctx := context.Background()

	err := cache.Set(ctx, "key1", []byte("value1"), time.Minute)
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}

	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("expected value1, got %s", string(val))
	}

	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	val, _ = cache.Get(ctx, "key1")
	if val != nil {
		t.Error("expected nil after delete")
	}
}

func TestMockCache_GetMiss(t *testing.T) {
	t.Parallel()
	cache := NewMockCache()
	ctx := context.Background()
	val, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if val != nil {
		t.Error("expected nil for missing key")
	}
}

func TestMockCache_Clear(t *testing.T) {
	t.Parallel()
	cache := NewMockCache()
	ctx := context.Background()
	_ = cache.Set(ctx, "key1", []byte("val"), time.Minute)
	_ = cache.Clear(ctx)
	val, _ := cache.Get(ctx, "key1")
	if val != nil {
		t.Error("expected nil after clear")
	}
}

func TestMockCache_ExpiredTTL(t *testing.T) {
	t.Parallel()
	cache := NewMockCache()
	ctx := context.Background()
	_ = cache.Set(ctx, "key1", []byte("val"), time.Nanosecond)
	time.Sleep(time.Millisecond)
	val, _ := cache.Get(ctx, "key1")
	if val != nil {
		t.Error("expected nil for expired key")
	}
}

func TestMockCache_GetCallCount(t *testing.T) {
	t.Parallel()
	cache := NewMockCache()
	ctx := context.Background()
	_ = cache.Set(ctx, "k", []byte("v"), time.Minute)
	_, _ = cache.Get(ctx, "k")
	_, _ = cache.Get(ctx, "k")
	_ = cache.Delete(ctx, "k")
	_ = cache.Clear(ctx)

	if c := cache.GetCallCount("Get"); c != 2 {
		t.Errorf("Get count = %d", c)
	}
	if c := cache.GetCallCount("Set"); c != 1 {
		t.Errorf("Set count = %d", c)
	}
	if c := cache.GetCallCount("Delete"); c != 1 {
		t.Errorf("Delete count = %d", c)
	}
	if c := cache.GetCallCount("Clear"); c != 1 {
		t.Errorf("Clear count = %d", c)
	}
}

func TestMockEventBus_SubscribePublish(t *testing.T) {
	t.Parallel()
	bus := NewMockEventBus()
	received := make(chan string, 1)
	bus.Subscribe("test.event", func(event any) {
		received <- event.(string)
	})
	bus.Publish("test.event", "hello")

	select {
	case msg := <-received:
		if msg != "hello" {
			t.Errorf("expected 'hello', got %s", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMockEventBus_GetPublishedEvents(t *testing.T) {
	t.Parallel()
	bus := NewMockEventBus()
	bus.Publish("e1", "data1")
	bus.Publish("e2", "data2")
	events := bus.GetPublishedEvents()
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestMockEventBus_GetPublishedEventCount(t *testing.T) {
	t.Parallel()
	bus := NewMockEventBus()
	bus.Publish("e1", "data")
	if c := bus.GetPublishedEventCount(); c != 1 {
		t.Errorf("expected 1, got %d", c)
	}
}

func TestMockRegistry_RegisterGet(t *testing.T) {
	t.Parallel()
	r := NewMockRegistry()
	_ = r.Register("plugin1", "value1")
	val, err := r.Get("plugin1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestMockRegistry_GetMissing(t *testing.T) {
	t.Parallel()
	r := NewMockRegistry()
	val, err := r.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if val != nil {
		t.Error("expected nil for missing plugin")
	}
}

func TestMockRegistry_RegisterFactoryGetFactory(t *testing.T) {
	t.Parallel()
	r := NewMockRegistry()
	factory := func() any { return "produced" }
	_ = r.RegisterFactory("factory1", factory)
	f, err := r.GetFactory("factory1")
	if err != nil {
		t.Fatalf("GetFactory error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
	if f() != "produced" {
		t.Error("factory produced wrong value")
	}
}

func TestMockRegistry_GetFactoryMissing(t *testing.T) {
	t.Parallel()
	r := NewMockRegistry()
	f, err := r.GetFactory("nonexistent")
	if err != nil {
		t.Fatalf("GetFactory error: %v", err)
	}
	if f != nil {
		t.Error("expected nil factory")
	}
}

func TestTestContextBuilder_Build(t *testing.T) {
	t.Parallel()
	builder := NewTestContextBuilder()
	ctx, cancel := builder.Build()
	defer cancel()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestTestContextBuilder_WithTimeout(t *testing.T) {
	t.Parallel()
	builder := NewTestContextBuilder().WithTimeout(time.Second)
	ctx, cancel := builder.Build()
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if time.Until(deadline) > time.Second {
		t.Error("expected deadline within 1 second")
	}
}

func TestTestContextBuilder_WithValue(t *testing.T) {
	t.Parallel()
	type ctxKey string
	builder := NewTestContextBuilder().WithValue(ctxKey("test"), "val")
	ctx, cancel := builder.Build()
	defer cancel()
	if v := ctx.Value(ctxKey("test")); v != "val" {
		t.Errorf("expected 'val', got %v", v)
	}
}

func TestTestDataBuilder_BuildEvent(t *testing.T) {
	t.Parallel()
	builder := NewTestDataBuilder()
	event := builder.BuildEvent("evt-1")
	if event.ID != "evt-1" {
		t.Errorf("expected evt-1, got %s", event.ID)
	}
	if event.ChainID != "ethereum" {
		t.Errorf("expected ethereum, got %s", event.ChainID)
	}
	if event.BlockNumber != 1 {
		t.Errorf("expected 1, got %d", event.BlockNumber)
	}
	if event.EventName != "TestEvent" {
		t.Errorf("expected TestEvent, got %s", event.EventName)
	}
}

func TestTestDataBuilder_BuildEvents(t *testing.T) {
	t.Parallel()
	builder := NewTestDataBuilder()
	events := builder.BuildEvents()
	if len(events) != 10 {
		t.Errorf("expected 10 events, got %d", len(events))
	}
}

func TestTestDataBuilder_WithChainID(t *testing.T) {
	t.Parallel()
	builder := NewTestDataBuilder().WithChainID("polygon")
	event := builder.BuildEvent("evt-1")
	if event.ChainID != "polygon" {
		t.Errorf("expected polygon, got %s", event.ChainID)
	}
}

func TestTestDataBuilder_WithBlockNumber(t *testing.T) {
	t.Parallel()
	builder := NewTestDataBuilder().WithBlockNumber(42)
	event := builder.BuildEvent("evt-1")
	if event.BlockNumber != 42 {
		t.Errorf("expected 42, got %d", event.BlockNumber)
	}
}

func TestTestDataBuilder_WithEventCount(t *testing.T) {
	t.Parallel()
	builder := NewTestDataBuilder().WithEventCount(5)
	events := builder.BuildEvents()
	if len(events) != 5 {
		t.Errorf("expected 5, got %d", len(events))
	}
}

func TestNilPointerDetector_Check(t *testing.T) {
	t.Parallel()
	npd := NewNilPointerDetector()
	if npd.Check("var1", nil) != true {
		t.Error("expected true for nil value")
	}
	if npd.Check("var2", "not nil") != false {
		t.Error("expected false for non-nil value")
	}
}

func TestNilPointerDetector_GetDetected(t *testing.T) {
	t.Parallel()
	npd := NewNilPointerDetector()
	npd.Check("var1", nil)
	npd.Check("var2", nil)
	detected := npd.GetDetected()
	if len(detected) != 2 {
		t.Errorf("expected 2, got %d", len(detected))
	}
	if detected[0] != "var1" || detected[1] != "var2" {
		t.Errorf("unexpected detected: %v", detected)
	}
}

func TestNilPointerDetector_HasDetected(t *testing.T) {
	t.Parallel()
	npd := NewNilPointerDetector()
	if npd.HasDetected() {
		t.Error("expected false initially")
	}
	npd.Check("var1", nil)
	if !npd.HasDetected() {
		t.Error("expected true after detecting nil")
	}
}

func TestNilPointerDetector_Reset(t *testing.T) {
	t.Parallel()
	npd := NewNilPointerDetector()
	npd.Check("var1", nil)
	npd.Reset()
	if npd.HasDetected() {
		t.Error("expected false after reset")
	}
}