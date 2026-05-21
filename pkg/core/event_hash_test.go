package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestComputeEventHash_Deterministic(t *testing.T) {
	t.Parallel()
	event := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     12345,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        3,
	}

	h1 := ComputeEventHash(event)
	h2 := ComputeEventHash(event)
	if h1 != h2 {
		t.Errorf("same event produced different hashes: %s vs %s", h1, h2)
	}
}

func TestComputeEventHash_DifferentEventsDifferentHashes(t *testing.T) {
	t.Parallel()
	base := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        0,
	}

	cases := []struct {
		name   string
		modify func(*BlockchainEvent)
	}{
		{"different chain", func(e *BlockchainEvent) { e.ChainID = "137" }},
		{"different block", func(e *BlockchainEvent) { e.BlockNumber = 200 }},
		{"different tx", func(e *BlockchainEvent) { e.TransactionHash = common.HexToHash("0xdef456") }},
		{"different log", func(e *BlockchainEvent) { e.LogIndex = 5 }},
	}

	baseHash := ComputeEventHash(base)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			modified := *base
			tc.modify(&modified)
			modifiedHash := ComputeEventHash(&modified)
			if modifiedHash == baseHash {
				t.Errorf("hash collision: %s modified event produced same hash as base", tc.name)
			}
		})
	}
}

func TestComputeEventHash_IgnoresDerivedFields(t *testing.T) {
	t.Parallel()
	// Two events with same natural key but different EventName/Network/ContractAddress
	// should produce the SAME hash (they are the same on-chain event)
	e1 := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        0,
		Network:         "ethereum",
		EventName:       "Transfer",
		ContractAddress: common.HexToAddress("0x1234"),
	}

	e2 := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        0,
		Network:         "mainnet",
		EventName:       "Approval",
		ContractAddress: common.HexToAddress("0x5678"),
	}

	if ComputeEventHash(e1) != ComputeEventHash(e2) {
		t.Error("events with same natural key but different derived fields should have same hash")
	}
}

func TestComputeEventHash_NilEvent(t *testing.T) {
	t.Parallel()
	if ComputeEventHash(nil) != "" {
		t.Error("nil event should return empty string")
	}
}

func TestComputeEventHash_ZeroValues(t *testing.T) {
	t.Parallel()
	event := &BlockchainEvent{
		ChainID:         "",
		BlockNumber:     0,
		TransactionHash: common.Hash{},
		LogIndex:        0,
	}
	hash := ComputeEventHash(event)
	if hash == "" {
		t.Error("zero-value event should still produce a hash")
	}
}

func TestComputeEventHash_OutputLength(t *testing.T) {
	t.Parallel()
	event := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     1,
		TransactionHash: common.HexToHash("0x01"),
		LogIndex:        1,
	}
	hash := ComputeEventHash(event)
	// SHA-256 = 32 bytes = 64 hex chars
	if len(hash) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(hash))
	}
}

func TestEventNaturalKey(t *testing.T) {
	t.Parallel()
	event := &BlockchainEvent{
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        3,
	}
	key := EventNaturalKey(event)
	if key == "" {
		t.Error("natural key should not be empty")
	}
	// Should contain all components
	if key == EventNaturalKey(nil) {
		t.Error("nil event should produce different key")
	}
}
