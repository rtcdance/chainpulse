package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestPluginInterface verifies that Plugin interface is properly defined
func TestPluginInterface(t *testing.T) {
	// This test verifies the interface exists and has the expected methods
	var _ Plugin = (*mockPlugin)(nil)
}

// TestConfigInterface verifies that Config struct has required fields
func TestConfigInterface(t *testing.T) {
	config := Config{
		DataPullerType:    "https-jsonrpc",
		BlockchainNodeURL: "http://localhost:8545",
		MQType:            "kafka",
		CacheType:         "redis",
		DatabaseType:      "postgres",
		APIType:           "rest",
		DeploymentMode:    "monolithic",
		WorkerPoolSize:    10,
		BatchSize:         100,
		MaxRetries:        3,
		RetryBackoff:      100,
	}

	if config.DataPullerType != "https-jsonrpc" {
		t.Errorf("expected DataPullerType to be 'https-jsonrpc', got %s", config.DataPullerType)
	}

	if config.WorkerPoolSize != 10 {
		t.Errorf("expected WorkerPoolSize to be 10, got %d", config.WorkerPoolSize)
	}
}

// TestBlockchainEventModel verifies BlockchainEvent struct
func TestBlockchainEventModel(t *testing.T) {
	event := BlockchainEvent{
		ID:              "event-1",
		EventHash:       "hash-1",
		BlockNumber:     12345,
		TransactionHash: common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		EventName:       "Transfer",
		Status:          "processed",
	}

	if event.ID != "event-1" {
		t.Errorf("expected ID to be 'event-1', got %s", event.ID)
	}

	if event.BlockNumber != 12345 {
		t.Errorf("expected BlockNumber to be 12345, got %d", event.BlockNumber)
	}
}

// TestCacheEntryModel verifies CacheEntry struct
func TestCacheEntryModel(t *testing.T) {
	entry := CacheEntry{
		Key:      "cache-key-1",
		Value:    []byte("cache-value"),
		HitCount: 5,
	}

	if entry.Key != "cache-key-1" {
		t.Errorf("expected Key to be 'cache-key-1', got %s", entry.Key)
	}

	if string(entry.Value) != "cache-value" {
		t.Errorf("expected Value to be 'cache-value', got %s", string(entry.Value))
	}

	if entry.HitCount != 5 {
		t.Errorf("expected HitCount to be 5, got %d", entry.HitCount)
	}
}

// TestEventFilterModel verifies EventFilter struct
func TestEventFilterModel(t *testing.T) {
	filter := EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{common.HexToAddress("0x1234567890123456789012345678901234567890")},
		FromBlock:       1000,
		ToBlock:         2000,
		Limit:           100,
		Offset:          0,
	}

	if len(filter.ContractAddress) == 0 || filter.ContractAddress[0] != common.HexToAddress("0x1234567890123456789012345678901234567890") {
		t.Errorf("expected ContractAddress to match")
	}

	if filter.FromBlock != 1000 {
		t.Errorf("expected FromBlock to be 1000, got %d", filter.FromBlock)
	}
}

// TestHealthStatus verifies HealthStatus struct
func TestHealthStatus(t *testing.T) {
	status := HealthStatus{
		Status:  "healthy",
		Message: "System is running normally",
		Details: map[string]interface{}{
			"uptime": 3600,
			"cpu":    45.5,
		},
	}

	if status.Status != "healthy" {
		t.Errorf("expected Status to be 'healthy', got %s", status.Status)
	}

	if len(status.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(status.Details))
	}
}

// TestCacheStats verifies CacheStats struct
func TestCacheStats(t *testing.T) {
	stats := CacheStats{
		HitCount:      100,
		MissCount:     50,
		EvictionCount: 10,
		HitRate:       0.667,
	}

	if stats.HitCount != 100 {
		t.Errorf("expected HitCount to be 100, got %d", stats.HitCount)
	}

	if stats.HitRate < 0.66 || stats.HitRate > 0.67 {
		t.Errorf("expected HitRate to be around 0.667, got %f", stats.HitRate)
	}
}

// TestQueryResult verifies QueryResult struct
func TestQueryResult(t *testing.T) {
	result := QueryResult{
		Events:       []BlockchainEvent{},
		Total:        0,
		CacheHit:     true,
		ResponseTime: 5,
	}

	if !result.CacheHit {
		t.Errorf("expected CacheHit to be true")
	}

	if result.ResponseTime != 5 {
		t.Errorf("expected ResponseTime to be 5, got %d", result.ResponseTime)
	}
}

// TestConstants verifies constants are defined correctly
func TestConstants(t *testing.T) {
	if DefaultWorkerPoolSize != 10 {
		t.Errorf("expected DefaultWorkerPoolSize to be 10, got %d", DefaultWorkerPoolSize)
	}

	if DefaultBatchSize != 100 {
		t.Errorf("expected DefaultBatchSize to be 100, got %d", DefaultBatchSize)
	}

	if DefaultMaxRetries != 3 {
		t.Errorf("expected DefaultMaxRetries to be 3, got %d", DefaultMaxRetries)
	}
}

// mockPlugin is a mock implementation of Plugin for testing
type mockPlugin struct {
	name    string
	version string
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Version() string {
	return m.version
}

func (m *mockPlugin) Initialize(config Config) error {
	return nil
}

func (m *mockPlugin) Start() error {
	return nil
}

func (m *mockPlugin) Stop() error {
	return nil
}

func (m *mockPlugin) Health() error {
	return nil
}

// contextualMockPlugin implements both ContextualStarter and ContextualStopper
type contextualMockPlugin struct {
	mockPlugin
	startCtx context.Context
	stopCtx  context.Context
	startErr error
	stopErr  error
}

func (c *contextualMockPlugin) StartWithContext(ctx context.Context) error {
	c.startCtx = ctx
	return c.startErr
}

func (c *contextualMockPlugin) StopWithContext(ctx context.Context) error {
	c.stopCtx = ctx
	return c.stopErr
}

func TestStartPluginWithContext(t *testing.T) {
	t.Run("uses StartWithContext when available", func(t *testing.T) {
		p := &contextualMockPlugin{}
		ctx := context.Background()
		err := StartPlugin(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.startCtx != ctx {
			t.Error("StartWithContext was not called with the provided context")
		}
	})

	t.Run("falls back to Start when ContextualStarter not implemented", func(t *testing.T) {
		p := &mockPlugin{}
		err := StartPlugin(context.Background(), p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("propagates StartWithContext error", func(t *testing.T) {
		expectedErr := errors.New("start failed")
		p := &contextualMockPlugin{startErr: expectedErr}
		err := StartPlugin(context.Background(), p)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestStopPluginWithContext(t *testing.T) {
	t.Run("uses StopWithContext when available", func(t *testing.T) {
		p := &contextualMockPlugin{}
		ctx := context.Background()
		err := StopPlugin(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.stopCtx != ctx {
			t.Error("StopWithContext was not called with the provided context")
		}
	})

	t.Run("falls back to Stop when ContextualStopper not implemented", func(t *testing.T) {
		p := &mockPlugin{}
		err := StopPlugin(context.Background(), p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("propagates StopWithContext error", func(t *testing.T) {
		expectedErr := errors.New("stop failed")
		p := &contextualMockPlugin{stopErr: expectedErr}
		err := StopPlugin(context.Background(), p)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestContextualStarterRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &contextualMockPlugin{}
	err := StartPlugin(ctx, p)
	// StartWithContext receives the cancelled context; whether it returns
	// an error depends on the implementation. The test verifies the context
	// is properly passed through.
	if p.startCtx != ctx {
		t.Error("context not propagated to StartWithContext")
	}
	_ = err
}
