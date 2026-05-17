package main

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
)

func TestEventProcessorShadowRuntimeProcessorTracksSharedRuntimeStatus(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	runtime, err := newEventProcessorProcessingRuntime(EventProcessorConfig{
		Port:      8082,
		BatchSize: 100,
		LogLevel:  "info",
	}, logger, metrics)
	if err != nil {
		t.Fatalf("create processor runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Stop(); err != nil {
			t.Fatalf("stop processor runtime: %v", err)
		}
	})

	processorRuntime := runtime.MessageProcessor()
	event, err := newValidEventProcessorShadowEvent()
	if err != nil {
		t.Fatalf("build shadow event: %v", err)
	}

	if err := processorRuntime.ProcessEvent(context.Background(), event); err != nil {
		t.Fatalf("process event: %v", err)
	}

	shadowProvider, ok := processorRuntime.(eventProcessorSharedRuntimeShadowProvider)
	if !ok {
		t.Fatal("expected shared runtime shadow provider")
	}

	snapshot := shadowProvider.SharedRuntimeShadowSnapshot()
	if !snapshot.Enabled {
		t.Fatal("expected shared runtime shadow enabled")
	}
	if snapshot.RuntimeCount != 1 {
		t.Fatalf("expected runtime count 1, got %d", snapshot.RuntimeCount)
	}
	if snapshot.ProcessedEvents != 1 {
		t.Fatalf("expected processed events 1, got %d", snapshot.ProcessedEvents)
	}
	if snapshot.LastCheckpointChain != "ethereum" {
		t.Fatalf("expected last checkpoint chain ethereum, got %q", snapshot.LastCheckpointChain)
	}
}

func newValidEventProcessorShadowEvent() (*core.BlockchainEvent, error) {
	return &core.BlockchainEvent{
		ID:              "evt-shadow-1",
		ChainID:         "ethereum",
		Network:         "ethereum",
		BlockNumber:     123,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1234"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		EventName:       "Transfer",
	}, nil
}
