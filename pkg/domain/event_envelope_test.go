package domain

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

func TestNewEventEnvelope(t *testing.T) {
	before := time.Now()

	envelope := NewEventEnvelope(
		"key-001",
		"ethereum",
		"0xabc123",
		42,
		3,
		map[string]any{"event": "Transfer"},
	)

	after := time.Now()

	if envelope.EventKey != "key-001" {
		t.Errorf("expected EventKey 'key-001', got %q", envelope.EventKey)
	}
	if envelope.ChainID != "ethereum" {
		t.Errorf("expected ChainID 'ethereum', got %q", envelope.ChainID)
	}
	if envelope.TransactionHash != "0xabc123" {
		t.Errorf("expected TransactionHash '0xabc123', got %q", envelope.TransactionHash)
	}
	if envelope.BlockNumber != 42 {
		t.Errorf("expected BlockNumber 42, got %d", envelope.BlockNumber)
	}
	if envelope.LogIndex != 3 {
		t.Errorf("expected LogIndex 3, got %d", envelope.LogIndex)
	}
	if envelope.Payload == nil {
		t.Error("expected Payload to be non-nil")
	}

	if envelope.ReceivedAt.Before(before) || envelope.ReceivedAt.After(after) {
		t.Errorf("expected ReceivedAt between %v and %v, got %v", before, after, envelope.ReceivedAt)
	}
}

func TestNewEventEnvelope_ZeroValues(t *testing.T) {
	envelope := NewEventEnvelope("", "", "", 0, 0, nil)

	if envelope.EventKey != "" {
		t.Errorf("expected empty EventKey, got %q", envelope.EventKey)
	}
	if envelope.ChainID != "" {
		t.Errorf("expected empty ChainID, got %q", envelope.ChainID)
	}
	if envelope.BlockNumber != 0 {
		t.Errorf("expected BlockNumber 0, got %d", envelope.BlockNumber)
	}
	if envelope.LogIndex != 0 {
		t.Errorf("expected LogIndex 0, got %d", envelope.LogIndex)
	}
	if envelope.Payload != nil {
		t.Error("expected nil Payload")
	}
	if envelope.ReceivedAt.IsZero() {
		t.Error("expected non-zero ReceivedAt")
	}
}

func TestEventEnvelope_TypeAlias(t *testing.T) {
	var e EventEnvelope
	var _ core.EventEnvelope = e

	e = NewEventEnvelope("evt", "chain", "hash", 1, 0, nil)
	if e.CheckpointCursor != "" {
		t.Log("CheckpointCursor defaults to empty string")
	}
}

func TestBatchProcessor_Interface(t *testing.T) {
	var _ BatchProcessor = (*mockBatchProcessor)(nil)
}

func TestSharedBatchRuntime_Interface(t *testing.T) {
	var _ SharedBatchRuntime = (*mockSharedBatchRuntime)(nil)
}

type mockBatchProcessor struct{}

func (m *mockBatchProcessor) ProcessBatch(_ context.Context, _ string, _ []EventEnvelope) error {
	return nil
}

type mockSharedBatchRuntime struct{}

func (m *mockSharedBatchRuntime) ProcessBatch(_ context.Context, _ string, _ []EventEnvelope) error {
	return nil
}
