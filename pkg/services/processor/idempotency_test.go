package processor

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

type testMetricsCollector struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func newTestMetricsCollector() *testMetricsCollector {
	return &testMetricsCollector{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *testMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *testMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *testMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
}

func (m *testMetricsCollector) GetMetrics() map[string]any {
	return nil
}

func (m *testMetricsCollector) getCounter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func makeTestEvent(id int, chainID int64) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ID:              fmt.Sprintf("evt_%d", id),
		ChainID:         fmt.Sprintf("%d", chainID),
		BlockNumber:     uint64(1000 + id),
		TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
		LogIndex:        uint64(id % 5),
		Network:         "mainnet",
		EventName:       "Transfer",
	}
}

func TestIdempotency_Initialize(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	err := svc.Initialize(&core.Config{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = svc.Initialize(&core.Config{})
	if err == nil {
		t.Fatal("second Initialize should fail")
	}
}

func TestIdempotency_InitializeNilConfig(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	err := svc.Initialize(nil)
	if err == nil {
		t.Fatal("Initialize with nil config should fail")
	}
}

func TestIdempotency_StartStop(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})

	err := svc.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = svc.Start()
	if err == nil {
		t.Fatal("second Start should fail")
	}

	err = svc.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	err = svc.Stop()
	if err == nil {
		t.Fatal("second Stop should fail")
	}
}

func TestIdempotency_StartWithoutInit(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	err := svc.Start()
	if err == nil {
		t.Fatal("Start without Initialize should fail")
	}
}

func TestIdempotency_HealthStates(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	status := svc.Health()
	if status.Status != "unhealthy" {
		t.Fatalf("uninitialized service should be unhealthy, got %s", status.Status)
	}

	_ = svc.Initialize(&core.Config{})

	status = svc.Health()
	if status.Status != "unhealthy" {
		t.Fatalf("initialized but not started should be unhealthy, got %s", status.Status)
	}

	_ = svc.Start()

	status = svc.Health()
	if status.Status != "healthy" {
		t.Fatalf("started service should be healthy, got %s", status.Status)
	}

	_ = svc.Stop()

	status = svc.Health()
	if status.Status != "unhealthy" {
		t.Fatalf("stopped service should be unhealthy, got %s", status.Status)
	}
}

func TestIdempotency_GenerateHash(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})

	event := makeTestEvent(1, 1)
	hash, err := svc.GenerateHash(event)
	if err != nil {
		t.Fatalf("GenerateHash failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	_, err = svc.GenerateHash(nil)
	if err == nil {
		t.Fatal("GenerateHash with nil event should fail")
	}
}

func TestIdempotency_GenerateHashDeterministic(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	event := makeTestEvent(1, 1)
	h1, _ := svc.GenerateHash(event)
	h2, _ := svc.GenerateHash(event)

	if h1 != h2 {
		t.Fatalf("same event produced different hashes: %s vs %s", h1, h2)
	}
}

func TestIdempotency_GenerateHashDifferentEvents(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())

	e1 := makeTestEvent(1, 1)
	e2 := makeTestEvent(2, 1)

	h1, _ := svc.GenerateHash(e1)
	h2, _ := svc.GenerateHash(e2)

	if h1 == h2 {
		t.Fatal("different events should have different hashes")
	}
}

func TestIdempotency_IsDuplicateAndMarkProcessed(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()
	event := makeTestEvent(1, 1)
	hash, _ := svc.GenerateHash(event)

	isDup, err := svc.IsDuplicate(ctx, hash)
	if err != nil {
		t.Fatalf("IsDuplicate failed: %v", err)
	}
	if isDup {
		t.Fatal("new event should not be duplicate")
	}

	err = svc.MarkProcessed(ctx, hash)
	if err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	isDup, err = svc.IsDuplicate(ctx, hash)
	if err != nil {
		t.Fatalf("IsDuplicate failed: %v", err)
	}
	if !isDup {
		t.Fatal("processed event should be duplicate")
	}
}

func TestIdempotency_IsDuplicateEmptyHash(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	_, err := svc.IsDuplicate(context.Background(), "")
	if err == nil {
		t.Fatal("IsDuplicate with empty hash should fail")
	}
}

func TestIdempotency_MarkProcessedEmptyHash(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	err := svc.MarkProcessed(context.Background(), "")
	if err == nil {
		t.Fatal("MarkProcessed with empty hash should fail")
	}
}

func TestIdempotency_CounterTracking(t *testing.T) {
	metrics := newTestMetricsCollector()
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), metrics)
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		event := makeTestEvent(i, 1)
		hash, _ := svc.GenerateHash(event)
		_ = svc.MarkProcessed(ctx, hash)
	}

	if svc.GetProcessedCount() != 5 {
		t.Fatalf("processed count: want 5, got %d", svc.GetProcessedCount())
	}

	event := makeTestEvent(3, 1)
	hash, _ := svc.GenerateHash(event)
	_ = svc.MarkProcessed(ctx, hash)

	if svc.GetDuplicateCount() != 1 {
		t.Fatalf("duplicate count: want 1, got %d", svc.GetDuplicateCount())
	}
}

func TestIdempotency_Clear(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()

	for i := 1; i <= 10; i++ {
		event := makeTestEvent(i, 1)
		hash, _ := svc.GenerateHash(event)
		_ = svc.MarkProcessed(ctx, hash)
	}

	if svc.GetProcessedCount() != 10 {
		t.Fatalf("processed count before clear: want 10, got %d", svc.GetProcessedCount())
	}

	err := svc.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if svc.GetProcessedCount() != 0 {
		t.Fatalf("processed count after clear: want 0, got %d", svc.GetProcessedCount())
	}

	if svc.GetDuplicateCount() != 0 {
		t.Fatalf("duplicate count after clear: want 0, got %d", svc.GetDuplicateCount())
	}

	event := makeTestEvent(1, 1)
	hash, _ := svc.GenerateHash(event)
	isDup, _ := svc.IsDuplicate(ctx, hash)
	if isDup {
		t.Fatal("cleared event should not be duplicate")
	}
}

func TestIdempotency_ClearNotRunning(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})

	err := svc.Clear()
	if err == nil {
		t.Fatal("Clear on not-running service should fail")
	}
}

func TestIdempotency_OpsNotRunning(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})

	ctx := context.Background()

	_, err := svc.IsDuplicate(ctx, "some-hash")
	if err == nil {
		t.Fatal("IsDuplicate on not-running service should fail")
	}

	err = svc.MarkProcessed(ctx, "some-hash")
	if err == nil {
		t.Fatal("MarkProcessed on not-running service should fail")
	}
}

func TestIdempotency_MetricsRecording(t *testing.T) {
	metrics := newTestMetricsCollector()
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), metrics)
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()

	event := makeTestEvent(1, 1)
	hash, _ := svc.GenerateHash(event)

	if cnt := metrics.getCounter("idempotency_hash_generated"); cnt != 1 {
		t.Fatalf("hash_generated counter: want 1, got %d", cnt)
	}

	_ = svc.MarkProcessed(ctx, hash)
	if cnt := metrics.getCounter("idempotency_event_marked"); cnt != 1 {
		t.Fatalf("event_marked counter: want 1, got %d", cnt)
	}

	_ = svc.MarkProcessed(ctx, hash)
	if cnt := metrics.getCounter("idempotency_duplicate_marked"); cnt != 1 {
		t.Fatalf("duplicate_marked counter: want 1, got %d", cnt)
	}
}

func TestIdempotency_ConcurrentSafety(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()
	const numGoroutines = 20
	const opsPerRoutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerRoutine; i++ {
				event := makeTestEvent(gid*opsPerRoutine+i, int64(gid%3))
				hash, _ := svc.GenerateHash(event)
				_ = svc.MarkProcessed(ctx, hash)
				_, _ = svc.IsDuplicate(ctx, hash)
			}
		}(g)
	}

	wg.Wait()

	totalExpected := int64(numGoroutines * opsPerRoutine)
	if svc.GetProcessedCount() != totalExpected {
		t.Fatalf("processed count: want %d, got %d", totalExpected, svc.GetProcessedCount())
	}
}

func TestIdempotency_ConcurrentClear(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()

	for i := 1; i <= 100; i++ {
		event := makeTestEvent(i, 1)
		hash, _ := svc.GenerateHash(event)
		_ = svc.MarkProcessed(ctx, hash)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			event := makeTestEvent(1000+i, 1)
			hash, _ := svc.GenerateHash(event)
			_ = svc.MarkProcessed(ctx, hash)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = svc.Clear()
			for j := 0; j < 10; j++ {
				event := makeTestEvent(2000+j, 1)
				hash, _ := svc.GenerateHash(event)
				_ = svc.MarkProcessed(ctx, hash)
			}
		}
	}()

	wg.Wait()
	t.Logf("processed: %d, duplicates: %d", svc.GetProcessedCount(), svc.GetDuplicateCount())
}

func TestIdempotency_MultiChainIsolation(t *testing.T) {
	svc := NewDefaultIdempotencyService(core.NewTestLogger(), newTestMetricsCollector())
	_ = svc.Initialize(&core.Config{})
	_ = svc.Start()
	defer svc.Stop()

	ctx := context.Background()
	chains := []int64{1, 137, 42161}
	_ = chains
	type chainEvent struct {
		chain int64
		block uint64
		id    int
	}

	events := []chainEvent{
		{1, 1000, 1},
		{137, 1000, 1},
		{42161, 1000, 1},
		{1, 1001, 2},
		{137, 1001, 2},
	}

	hashes := make(map[int64]string)
	for _, e := range events {
		event := &core.BlockchainEvent{
			ID:              fmt.Sprintf("evt_%d_%d", e.chain, e.id),
			ChainID:         fmt.Sprintf("%d", e.chain),
			BlockNumber:     e.block,
			TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", e.id)),
			LogIndex:        0,
		}
		hash, _ := svc.GenerateHash(event)
		hashes[e.chain] = hash
		_ = svc.MarkProcessed(ctx, hash)
	}

	if svc.GetProcessedCount() != int64(len(events)) {
		t.Fatalf("all events across chains should be unique: want %d, got %d", len(events), svc.GetProcessedCount())
	}
}
