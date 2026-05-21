package runtimetypes

import (
	"testing"
	"time"
)

func TestRuntimeTypesSmoke(t *testing.T) {
	env := EventEnvelope{
		EventKey:         "evt-1",
		ChainID:          "ethereum",
		BlockNumber:      100,
		TransactionHash:  "0xabc",
		LogIndex:         1,
		ReceivedAt:       time.Now(),
		CheckpointCursor: "100:0",
	}
	if env.EventKey != "evt-1" {
		t.Error("bad event key")
	}

	cp := Checkpoint{
		ChainID:     "ethereum",
		BlockNumber: 100,
		Cursor:      "100:0",
		UpdatedAt:   time.Now(),
	}
	if cp.BlockNumber != 100 {
		t.Error("bad block number")
	}

	pf := ProcessingFailure{
		EventKey:   "evt-1",
		ChainID:    "ethereum",
		Retryable:  false,
		Reason:     "test error",
		OccurredAt: time.Now(),
	}
	if pf.Reason != "test error" {
		t.Error("bad reason")
	}

	rs := RuntimeStatus{
		State:           "running",
		Initialized:     true,
		Started:         true,
		ProcessedEvents: 500,
		LastUpdatedAt:   time.Now(),
	}
	if !rs.Initialized {
		t.Error("expected initialized")
	}
}
