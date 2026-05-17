package pullers

import (
	"context"
	"errors"
	"testing"
	"time"

	"chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

func newBasePuller() *BaseDataPullerPlugin {
	return NewBaseDataPullerPlugin("test", "1.0", core.Config{
		StartBlock:   100,
		MaxRetries:   3,
		RetryBackoff: 100,
		ChainID:      "1",
	}, &noopLogger{}, &noopMetrics{}, nil)
}

func TestBaseDataPullerPlugin_NameVersion(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if p.Name() != "test" {
		t.Errorf("Name() = %q", p.Name())
	}
	if p.Version() != "1.0" {
		t.Errorf("Version() = %q", p.Version())
	}
}

func TestBaseDataPullerPlugin_InitializeConfig(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	cfg := core.Config{StartBlock: 200, MaxRetries: 5, RetryBackoff: 500}
	if err := p.Initialize(cfg); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
}

func TestBaseDataPullerPlugin_StartStop(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if err := p.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if !p.IsRunning() {
		t.Error("IsRunning() should be true after Start")
	}
	if err := p.Start(); err == nil {
		t.Error("expected error for double Start")
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
	if p.IsRunning() {
		t.Error("IsRunning() should be false after Stop")
	}
}

func TestBaseDataPullerPlugin_StopNotRunning(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if err := p.Stop(); err == nil {
		t.Error("expected error when stopping not running")
	}
}

func TestBaseDataPullerPlugin_Health(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	_ = p.Start()
	defer p.Stop()

	if err := p.Health(); err != nil {
		t.Fatalf("Health() error: %v", err)
	}
}

func TestBaseDataPullerPlugin_HealthNotRunning(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if err := p.Health(); err == nil {
		t.Error("expected error from Health() when not running")
	}
}

func TestBaseDataPullerPlugin_BlockNumber(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if p.GetLastBlockNumber() != 100 {
		t.Errorf("GetLastBlockNumber() = %d, want 100", p.GetLastBlockNumber())
	}
	p.SetLastBlockNumber(500)
	if p.GetLastBlockNumber() != 500 {
		t.Errorf("after SetLastBlockNumber(500), GetLastBlockNumber() = %d", p.GetLastBlockNumber())
	}
}

func TestBaseDataPullerPlugin_GetConfig(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	cfg := p.GetConfig()
	if cfg.StartBlock != 100 {
		t.Errorf("StartBlock = %d", cfg.StartBlock)
	}
}

func TestBaseDataPullerPlugin_ChainID(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("", "", core.Config{ChainID: "ethereum"}, nil, nil, nil)
	if p.ChainID() != "ethereum" {
		t.Errorf("ChainID() = %q", p.ChainID())
	}
}

func TestBaseDataPullerPlugin_Network(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("", "", core.Config{Network: "mainnet"}, nil, nil, nil)
	if p.Network() != "mainnet" {
		t.Errorf("Network() = %q", p.Network())
	}
}

func TestBaseDataPullerPlugin_ValidateConfig(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	if err := p.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig() error: %v", err)
	}
}

func TestBaseDataPullerPlugin_ConnectionTimeout(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.SetConnectionTimeout(10 * time.Second)
	if p.GetConnectionTimeout() != 10*time.Second {
		t.Errorf("GetConnectionTimeout() = %v", p.GetConnectionTimeout())
	}
}

func TestBaseDataPullerPlugin_ValidateEvent(t *testing.T) {
	t.Parallel()
	p := newBasePuller()

	tests := []struct {
		name  string
		event core.BlockchainEvent
		fails bool
	}{
		{"valid", core.BlockchainEvent{
			ID:              "1",
			ChainID:         "1",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
			ContractAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			EventName:       "Transfer",
			Status:          "pending",
		}, false},
		{"empty id", core.BlockchainEvent{ChainID: "1", BlockNumber: 100}, true},
		{"empty chain", core.BlockchainEvent{ID: "1", BlockNumber: 100}, true},
		{"zero block", core.BlockchainEvent{ID: "1", ChainID: "1"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := p.ValidateEvent(tc.event)
			if tc.fails && err == nil {
				t.Error("expected validation error")
			}
			if !tc.fails && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBaseDataPullerPlugin_GenerateEventHash(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	event := core.BlockchainEvent{
		ID:              "evt1",
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: [32]byte{1, 2, 3},
	}
	h1 := p.GenerateEventHash(event)
	h2 := p.GenerateEventHash(event)
	if h1 == "" {
		t.Error("expected non-empty hash")
	}
	if h1 != h2 {
		t.Error("expected deterministic hash")
	}
}

func TestBaseDataPullerPlugin_RetryWithBackoff_Success(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.SetConnectionTimeout(100 * time.Millisecond)

	attempts := 0
	err := p.RetryWithBackoff(context.Background(), func() error {
		attempts++
		return nil
	})
	if err != nil {
		t.Fatalf("RetryWithBackoff() error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestBaseDataPullerPlugin_RetryWithBackoff_FailsThenSucceeds(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{
		StartBlock:   0,
		MaxRetries:   5,
		RetryBackoff: 1,
	}, &noopLogger{}, &noopMetrics{}, nil)

	attempts := 0
	err := p.RetryWithBackoff(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RetryWithBackoff() error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestBaseDataPullerPlugin_RetryWithBackoff_AllFail(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{
		StartBlock:   0,
		MaxRetries:   3,
		RetryBackoff: 1,
	}, &noopLogger{}, &noopMetrics{}, nil)

	attempts := 0
	err := p.RetryWithBackoff(context.Background(), func() error {
		attempts++
		return errors.New("always fails")
	})
	if err == nil {
		t.Error("expected error after all retries fail")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestBaseDataPullerPlugin_LogMethods(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.LogInfo("test info")
	p.LogError("test error")
	p.LogWarn("test warn")
}

func TestBaseDataPullerPlugin_RecordMetric(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.RecordMetric("test_metric", 42, map[string]string{"key": "val"})
}

func TestBaseDataPullerPlugin_SetCheckpointStore(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.SetCheckpointStore(nil)
	p.SetLifecycleContext(context.Background())
}

func TestBaseDataPullerPlugin_GetConfigDefaultStartBlock(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("t", "1", core.Config{}, nil, nil, nil)
	if p.GetLastBlockNumber() != 0 {
		t.Errorf("expected 0, got %d", p.GetLastBlockNumber())
	}
}

func TestBaseDataPullerPlugin_PublishEvents(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	ctx := context.Background()

	err := p.PublishEvent(ctx, core.BlockchainEvent{ID: "evt1"})
	if err != nil {
		t.Logf("PublishEvent error (expected without EventBus): %v", err)
	}

	err = p.PublishEvents(ctx, []core.BlockchainEvent{{ID: "evt1"}, {ID: "evt2"}})
	if err != nil {
		t.Logf("PublishEvents error (expected without EventBus): %v", err)
	}
}
