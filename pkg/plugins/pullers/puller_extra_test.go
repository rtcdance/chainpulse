package pullers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"

	"github.com/ethereum/go-ethereum/common"
)

func TestBaseDataPullerPlugin_Start_MissingChainID(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{}, &noopLogger{}, &noopMetrics{}, nil)
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for missing ChainID")
	}
}

func TestBaseDataPullerPlugin_RecordRequest(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.RecordRequest()
	p.RecordRequest()
	p.RecordRequest()

	stats := p.BaseStats()
	if stats["request_count"].(int64) != 3 {
		t.Fatalf("expected 3 requests, got %d", stats["request_count"])
	}
}

func TestBaseDataPullerPlugin_RecordError(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.RecordError(errors.New("err1"))
	p.RecordError(errors.New("err2"))

	stats := p.BaseStats()
	if stats["error_count"].(int64) != 2 {
		t.Fatalf("expected 2 errors, got %d", stats["error_count"])
	}
}

func TestBaseDataPullerPlugin_RecordSuccessfulPull(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.RecordError(errors.New("boom"))
	p.RecordSuccessfulPull()

	stats := p.BaseStats()
	if stats["error_count"].(int64) != 0 {
		t.Fatalf("expected 0 errors after reset, got %d", stats["error_count"])
	}
	if stats["last_error"] != nil {
		t.Fatal("expected last_error to be nil after reset")
	}
}

func TestBaseDataPullerPlugin_SetRPCHealthCheck(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	cb := p.CircuitBreaker()
	if cb == nil {
		t.Fatal("expected non-nil circuit breaker")
	}

	called := false
	p.SetRPCHealthCheck(func(ctx context.Context) error {
		called = true
		return nil
	})

	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if !called {
		t.Fatal("expected RPC health check to be called")
	}
}

func TestBaseDataPullerPlugin_Health_RPCHealthCheckFails(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.SetRPCHealthCheck(func(ctx context.Context) error {
		return errors.New("rpc unreachable")
	})
	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected health check error from RPC failure")
	}
}

func TestBaseDataPullerPlugin_Health_NoSuccessfulPull(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	p.mu.Lock()
	p.lastSuccessfulPull = time.Now().Add(-10 * time.Minute)
	p.mu.Unlock()

	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected health error due to no recent pull")
	}
}

func TestBaseDataPullerPlugin_Health_RecentError(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	p.mu.Lock()
	p.lastError = errors.New("recent failure")
	p.lastErrorTime = time.Now()
	p.mu.Unlock()

	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected health error due to recent error")
	}
}

func TestBaseDataPullerPlugin_Health_CircuitBreakerOpen(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	cb := p.CircuitBreaker()
	for i := 0; i < 6; i++ {
		cb.Failure()
	}

	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected health error due to open circuit breaker")
	}
}

func TestBaseDataPullerPlugin_Health_CircuitBreakerHalfOpen(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	_ = p.Start(context.Background())
	defer p.Stop(context.Background())

	cb := p.CircuitBreaker()
	for i := 0; i < 6; i++ {
		cb.Failure()
	}
	cb.mu.Lock()
	cb.state = CircuitBreakerHalfOpen
	cb.mu.Unlock()

	err := p.Health(context.Background())
	if err == nil {
		t.Fatal("expected health error due to half-open circuit breaker")
	}
}

func TestBaseDataPullerPlugin_BuildBlockchainEvent(t *testing.T) {
	t.Parallel()
	p := newBasePuller()

	txHash := common.HexToHash("0xdead")
	contractAddr := common.HexToAddress("0xbeef")
	topics := []common.Hash{common.HexToHash("0xaaa")}

	event := p.BuildBlockchainEvent(
		"1", "mainnet",
		txHash, 100, 0,
		contractAddr, []byte{1, 2, 3},
		topics, "Transfer", topics[0],
		1234567890, false,
	)

	if event.ChainID != "1" {
		t.Errorf("ChainID = %q", event.ChainID)
	}
	if event.BlockNumber != 100 {
		t.Errorf("BlockNumber = %d", event.BlockNumber)
	}
	if event.EventName != "Transfer" {
		t.Errorf("EventName = %q", event.EventName)
	}
	if event.EventHash == "" {
		t.Error("expected non-empty EventHash")
	}
	if event.Status != blockchain.EventStatusPending {
		t.Errorf("Status = %v", blockchain.EventStatusPending)
	}
}

func TestBaseDataPullerPlugin_ValidateEvent_ContractAddressFilter(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{
		ChainID:           "1",
		ContractAddresses: []string{"0x0000000000000000000000000000000000000aaa"},
	}, &noopLogger{}, &noopMetrics{}, nil)

	txHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	allowedAddr := common.HexToAddress("0xaaa")
	disallowedAddr := common.HexToAddress("0xbbb")

	tests := []struct {
		name    string
		address common.Address
		fails   bool
	}{
		{"allowed address", allowedAddr, false},
		{"disallowed address", disallowedAddr, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := blockchain.BlockchainEvent{
				BlockNumber:     100,
				TransactionHash: txHash,
				ContractAddress: tt.address,
				EventName:       "Transfer",
			}
			err := p.ValidateEvent(ev)
			if tt.fails && err == nil {
				t.Error("expected validation error")
			}
			if !tt.fails && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBaseDataPullerPlugin_ChainID_Fallback(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("", "", core.Config{ServiceName: "my-service"}, nil, nil, nil)
	if p.ChainID() != "my-service" {
		t.Errorf("ChainID() = %q, want 'my-service'", p.ChainID())
	}
}

func TestBaseDataPullerPlugin_NetworDefaultk(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("", "", core.Config{}, nil, nil, nil)
	if p.Network() != "mainnet" {
		t.Errorf("Network() = %q, want 'mainnet'", p.Network())
	}
}

func TestBaseDataPullerPlugin_ValidateConfig_EmptyChainID(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("", "", core.Config{}, nil, nil, nil)
	err := p.ValidateConfig()
	if err == nil {
		t.Fatal("expected validation error for empty ChainID")
	}
}

func TestBaseDataPullerPlugin_LoadCheckpoint_NoStore(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	blockNum := p.LoadCheckpoint(context.Background())
	if blockNum != 0 {
		t.Fatalf("expected 0, got %d", blockNum)
	}
}

func TestBaseDataPullerPlugin_LoadCheckpointWithHashNoStore(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	blockNum, hash := p.LoadCheckpointWithHash(context.Background())
	if blockNum != 0 || hash != "" {
		t.Fatalf("expected (0, ''), got (%d, %q)", blockNum, hash)
	}
}

func TestBaseDataPullerPlugin_SetLastBlockNumberWithHash(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	p.SetLastBlockNumberWithHash(500, "0xabc")
	if p.GetLastBlockNumber() != 500 {
		t.Fatalf("expected 500, got %d", p.GetLastBlockNumber())
	}
}

func TestBaseDataPullerPlugin_CircuitBreaker(t *testing.T) {
	t.Parallel()
	p := newBasePuller()
	cb := p.CircuitBreaker()
	if cb == nil {
		t.Fatal("expected non-nil circuit breaker")
	}
	cb2 := p.CircuitBreaker()
	if cb != cb2 {
		t.Fatal("expected same circuit breaker instance")
	}
}

func TestBaseDataPullerPlugin_LogMethods_NilLogger(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{}, nil, nil, nil)
	p.LogInfo("test")
	p.LogError("test")
	p.LogWarn("test")
}

func TestBaseDataPullerPlugin_RecordMetric_NilCollector(t *testing.T) {
	t.Parallel()
	p := NewBaseDataPullerPlugin("test", "1.0", core.Config{}, &noopLogger{}, nil, nil)
	p.RecordMetric("test", int64(1), nil)
	p.RecordMetric("test", float64(1.0), nil)
}
