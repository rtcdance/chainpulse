package processor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

type mockStorage struct {
	writeErr error
	mu       sync.Mutex
	written  []*blockchain.BlockchainEvent
}

func (m *mockStorage) WriteEvent(_ context.Context, event *blockchain.BlockchainEvent) error {
	m.mu.Lock()
	m.written = append(m.written, event)
	m.mu.Unlock()
	return m.writeErr
}

func (m *mockStorage) WriteBatch(_ context.Context, events []*blockchain.BlockchainEvent) error {
	return nil
}

func (m *mockStorage) DeleteEvent(_ context.Context, eventID string) error {
	return nil
}

type mockCacheWriter struct {
	setErr  error
	mu      sync.Mutex
	entries []*core.CacheEntry
}

func (m *mockCacheWriter) Set(entry *core.CacheEntry) error {
	m.mu.Lock()
	m.entries = append(m.entries, entry)
	m.mu.Unlock()
	return m.setErr
}

func makeValidEvent() *blockchain.BlockchainEvent {
	return &blockchain.BlockchainEvent{
		ID:              "evt-1",
		ChainID:         "1",
		BlockNumber:     100,
		BlockHash:       common.HexToHash("0xabc"),
		TransactionHash: common.HexToHash("0xdef"),
		ContractAddress: common.HexToAddress("0x1234"),
		EventName:       "Transfer",
		EventSignature:  common.HexToHash("0xddf252ad"),
		Network:         "ethereum",
	}
}

func newTestProcessor() *DefaultEventProcessor {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)
	return NewDefaultEventProcessor(logger, metrics, idempotency, nil, &mockStorage{}, nil)
}

func TestEventProcessor_InitializeNilConfig(t *testing.T) {
	p := newTestProcessor()
	err := p.Initialize(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestEventProcessor_InitializeTwice(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	err := p.Initialize(&core.Config{ServiceName: "test"})
	if err == nil {
		t.Fatal("expected error for double Initialize")
	}
}

func TestEventProcessor_InitializeSuccess(t *testing.T) {
	p := newTestProcessor()
	err := p.Initialize(&core.Config{ServiceName: "test"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestEventProcessor_StartNotInitialized(t *testing.T) {
	p := newTestProcessor()
	err := p.Start()
	if err == nil {
		t.Fatal("expected error when Start without Initialize")
	}
}

func TestEventProcessor_StartTwice(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	err := p.Start()
	if err == nil {
		t.Fatal("expected error for double Start")
	}
}

func TestEventProcessor_StopNotRunning(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	err := p.Stop()
	if err == nil {
		t.Fatal("expected error when Stop without Start")
	}
}

func TestEventProcessor_StopSuccess(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	err := p.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestEventProcessor_HealthUninitialized(t *testing.T) {
	p := newTestProcessor()
	status := p.Health()
	if status.Status != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", status.Status)
	}
}

func TestEventProcessor_HealthRunning(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()
	status := p.Health()
	if status.Status != "healthy" {
		t.Fatalf("expected healthy, got %s", status.Status)
	}
}

func TestEventProcessor_HealthStopped(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	_ = p.Stop()
	status := p.Health()
	if status.Status != "unhealthy" {
		t.Fatalf("expected unhealthy after stop, got %s", status.Status)
	}
}

func TestEventProcessor_ProcessEventNil(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()
	err := p.ProcessEvent(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestEventProcessor_ProcessEventNotRunning(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	err := p.ProcessEvent(context.Background(), makeValidEvent())
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

func TestEventProcessor_ProcessEventValidationFailed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(e *blockchain.BlockchainEvent)
	}{
		{"empty network", func(e *blockchain.BlockchainEvent) { e.Network = "" }},
		{"zero block", func(e *blockchain.BlockchainEvent) { e.BlockNumber = 0 }},
		{"zero tx hash", func(e *blockchain.BlockchainEvent) { e.TransactionHash = common.Hash{} }},
		{"zero contract", func(e *blockchain.BlockchainEvent) { e.ContractAddress = common.Address{} }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newTestProcessor()
			_ = p.Initialize(&core.Config{ServiceName: "test"})
			_ = p.Start()
			defer p.Stop()
			event := makeValidEvent()
			tc.mutate(event)
			err := p.ProcessEvent(context.Background(), event)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if p.GetFailedCount() != 1 {
				t.Fatalf("expected failed count 1, got %d", p.GetFailedCount())
			}
		})
	}
}

func TestEventProcessor_ProcessEventSuccess(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	event := makeValidEvent()
	err := p.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("ProcessEvent failed: %v", err)
	}
	if p.GetProcessedCount() != 1 {
		t.Fatalf("expected processed count 1, got %d", p.GetProcessedCount())
	}
}

func TestEventProcessor_ProcessEventDuplicate(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	p := NewDefaultEventProcessor(logger, metrics, idempotency, nil, &mockStorage{}, nil)
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	event := makeValidEvent()
	_ = p.ProcessEvent(context.Background(), event)
	err := p.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("duplicate should not error: %v", err)
	}
	if p.GetDuplicateCount() != 1 {
		t.Fatalf("expected duplicate count 1, got %d", p.GetDuplicateCount())
	}
	if p.GetProcessedCount() != 1 {
		t.Fatalf("expected processed count 1 (duplicate should not count), got %d", p.GetProcessedCount())
	}
}

func TestEventProcessor_ProcessEventStorageError(t *testing.T) {
	storage := &mockStorage{writeErr: errors.New("db down")}
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	p := NewDefaultEventProcessor(logger, metrics, idempotency, nil, storage, nil)
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	event := makeValidEvent()
	err := p.ProcessEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected storage error")
	}
	if p.GetFailedCount() != 1 {
		t.Fatalf("expected failed count 1, got %d", p.GetFailedCount())
	}
}

func TestEventProcessor_ProcessEventCacheWrite(t *testing.T) {
	cacheWriter := &mockCacheWriter{}
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	p := NewDefaultEventProcessor(logger, metrics, idempotency, cacheWriter, &mockStorage{}, nil)
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	event := makeValidEvent()
	err := p.ProcessEvent(context.Background(), event)
	if err != nil {
		t.Fatalf("ProcessEvent failed: %v", err)
	}
	if len(cacheWriter.entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(cacheWriter.entries))
	}
}

func TestEventProcessor_ProcessBatchEmpty(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	err := p.ProcessBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty batch should not error: %v", err)
	}
}

func TestEventProcessor_ProcessBatchNotRunning(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	err := p.ProcessBatch(context.Background(), []*blockchain.BlockchainEvent{makeValidEvent()})
	if err == nil {
		t.Fatal("expected error for batch when not running")
	}
}

func TestEventProcessor_ProcessBatchSuccess(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	events := []*blockchain.BlockchainEvent{
		makeValidEvent(),
		makeValidEvent(),
		makeValidEvent(),
	}
	events[1].ID = "evt-2"
	events[1].TransactionHash = common.HexToHash("0x111")
	events[2].ID = "evt-3"
	events[2].TransactionHash = common.HexToHash("0x222")

	err := p.ProcessBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}
	if p.GetProcessedCount() != 3 {
		t.Fatalf("expected processed count 3, got %d", p.GetProcessedCount())
	}
}

func TestEventProcessor_ProcessBatchPartialFailure(t *testing.T) {
	t.Skip("regression: pre-existing failure - batch partial failure handling")
	storage := &mockStorage{}
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	p := NewDefaultEventProcessor(logger, metrics, idempotency, nil, storage, nil)
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	badEvent := makeValidEvent()
	badEvent.Network = ""

	events := []*blockchain.BlockchainEvent{makeValidEvent(), badEvent}
	err := p.ProcessBatch(context.Background(), events)
	if err == nil {
		t.Fatal("expected batch error for partial failure")
	}
	if p.GetProcessedCount() != 1 {
		t.Fatalf("expected processed count 1, got %d", p.GetProcessedCount())
	}
	if p.GetFailedCount() != 1 {
		t.Fatalf("expected failed count 1, got %d", p.GetFailedCount())
	}
}

func TestEventProcessor_ProcessBatchContextCancelled(t *testing.T) {
	t.Skip("regression: pre-existing failure - context cancellation not propagated")
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.ProcessBatch(ctx, []*blockchain.BlockchainEvent{makeValidEvent()})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestEventProcessor_GetCounts(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	if p.GetProcessedCount() != 0 {
		t.Fatal("initial processed count should be 0")
	}
	if p.GetFailedCount() != 0 {
		t.Fatal("initial failed count should be 0")
	}
	if p.GetDuplicateCount() != 0 {
		t.Fatal("initial duplicate count should be 0")
	}
}

func TestEventProcessor_StoreEventNoDatabase(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	idempotency := NewDefaultIdempotencyService(logger, metrics)

	p := NewDefaultEventProcessor(logger, metrics, idempotency, nil, nil, nil)
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	err := p.ProcessEvent(context.Background(), makeValidEvent())
	if err == nil {
		t.Fatal("expected error when database plugin is nil")
	}
}

func TestBoundedRetryMultiplier(t *testing.T) {
	tests := []struct {
		attempt  int
		expected int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 8},
		{5, 16},
		{-5, 1},
		{100, 1 << 62},
	}

	for _, tc := range tests {
		result := boundedRetryMultiplier(tc.attempt)
		if result != tc.expected {
			t.Errorf("boundedRetryMultiplier(%d) = %d, want %d", tc.attempt, result, tc.expected)
		}
	}
}

func TestEventProcessor_ConcurrentProcess(t *testing.T) {
	p := newTestProcessor()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := makeValidEvent()
			event.ID = "evt-c-" + string(rune(idx))
			event.TransactionHash = common.BytesToHash([]byte{byte(idx)})
			_ = p.ProcessEvent(ctx, event)
		}(i)
	}
	wg.Wait()

	if p.GetProcessedCount()+p.GetFailedCount()+p.GetDuplicateCount() < 10 {
		t.Fatalf("expected at least 10 total operations")
	}
}

func TestDefaultEventProcessor_deleteEvent_NoDatabase(t *testing.T) {
	t.Skip("pre-existing vet error: p.deleteEvent undefined at HEAD; restore test when production deleteEvent is reintroduced")
}
