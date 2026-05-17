package data

import (
	"testing"
)

func TestNewDataPullerMetrics(t *testing.T) {
	t.Parallel()
	dpm := NewDataPullerMetrics()
	if dpm == nil {
		t.Fatal("NewDataPullerMetrics() returned nil")
	}
	metrics := dpm.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}
}

func TestDataPullerMetrics_RecordEventPulled(t *testing.T) {
	t.Parallel()
	dpm := NewDataPullerMetrics()
	dpm.RecordEventPulled()
	dpm.RecordEventPulled()
	dpm.RecordEventPulled()
	metrics := dpm.GetMetrics()
	if v, ok := metrics["events_pulled"].(int64); !ok || v != 3 {
		t.Errorf("events_pulled = %v, want 3", metrics["events_pulled"])
	}
}

func TestDataPullerMetrics_RecordEventDropped(t *testing.T) {
	t.Parallel()
	dpm := NewDataPullerMetrics()
	dpm.RecordEventDropped()
	metrics := dpm.GetMetrics()
	if v, ok := metrics["events_dropped"].(int64); !ok || v != 1 {
		t.Errorf("events_dropped = %v, want 1", metrics["events_dropped"])
	}
}

func TestDataPullerMetrics_RecordError(t *testing.T) {
	t.Parallel()
	dpm := NewDataPullerMetrics()
	dpm.RecordError()
	dpm.RecordError()
	metrics := dpm.GetMetrics()
	if v, ok := metrics["errors"].(int64); !ok || v != 2 {
		t.Errorf("errors = %v, want 2", metrics["errors"])
	}
}

func TestDataPullerMetrics_AllMetrics(t *testing.T) {
	t.Parallel()
	dpm := NewDataPullerMetrics()
	dpm.RecordEventPulled()
	dpm.RecordEventDropped()
	dpm.RecordError()

	metrics := dpm.GetMetrics()
	if metrics["events_pulled"].(int64) != 1 {
		t.Error("events_pulled mismatch")
	}
	if metrics["events_dropped"].(int64) != 1 {
		t.Error("events_dropped mismatch")
	}
	if metrics["errors"].(int64) != 1 {
		t.Error("errors mismatch")
	}
}

func TestNewDataPuller_ConfigValidation(t *testing.T) {
	t.Parallel()
	cfg := DataPullerConfig{
		ChainType:      EVM,
		ChainID:        "ethereum",
		BlockchainNode: "http://localhost:8545",
		StartBlock:     100,
		BatchSize:      1000,
	}
	dp := NewDataPuller(cfg)
	if dp == nil {
		t.Fatal("NewDataPuller() returned nil")
	}
}

func TestMultiChainDataPuller_AddRemove(t *testing.T) {
	t.Parallel()
	mcdp := NewMultiChainDataPuller()
	dp := NewDataPuller(DataPullerConfig{ChainID: "eth"})

	if err := mcdp.AddPuller("eth", dp); err != nil {
		t.Fatalf("AddPuller() error: %v", err)
	}
	if err := mcdp.AddPuller("eth", dp); err == nil {
		t.Error("expected error for duplicate chain")
	}

	mcdp.RemovePuller("eth")
	mcdp.RemovePuller("nonexistent")

	if err := mcdp.AddPuller("eth", dp); err != nil {
		t.Fatalf("AddPuller() after remove error: %v", err)
	}
}

func TestBlockchainType_Values(t *testing.T) {
	t.Parallel()
	if EVM != "evm" {
		t.Errorf("EVM = %q", EVM)
	}
	if Cosmos != "cosmos" {
		t.Errorf("Cosmos = %q", Cosmos)
	}
	if Solana != "solana" {
		t.Errorf("Solana = %q", Solana)
	}
}
