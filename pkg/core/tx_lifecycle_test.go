package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestTxLifecycleFullPath(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	txHash := common.HexToHash("0xabc")
	blockHash := common.HexToHash("0xdef")

	// mempool → proposed → confirmed → finalized
	tracker.TrackMempool(txHash)
	state := tracker.GetTxState(txHash)
	if state.Current != TxInMempool {
		t.Errorf("expected mempool, got %s", state.Current)
	}

	tracker.TrackIncluded(txHash, 100, blockHash)
	state = tracker.GetTxState(txHash)
	if state.Current != TxProposed {
		t.Errorf("expected proposed, got %s", state.Current)
	}

	tracker.TrackConfirmed(txHash, 105)
	state = tracker.GetTxState(txHash)
	if state.Current != TxConfirmed {
		t.Errorf("expected confirmed, got %s", state.Current)
	}

	tracker.TrackFinalized(txHash, 64)
	state = tracker.GetTxState(txHash)
	if state.Current != TxFinalized {
		t.Errorf("expected finalized, got %s", state.Current)
	}

	// Verify full history
	if len(state.History) != 4 {
		t.Errorf("expected 4 history entries, got %d", len(state.History))
	}
}

func TestTxLifecycleReorg(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	txHash := common.HexToHash("0xabc")
	blockHash := common.HexToHash("0xdef")

	tracker.TrackIncluded(txHash, 100, blockHash)
	if tracker.GetTxState(txHash).Current != TxProposed {
		t.Fatal("expected proposed")
	}

	tracker.TrackReorged(blockHash, 99)
	if tracker.GetTxState(txHash).Current != TxReorged {
		t.Errorf("expected reorged, got %s", tracker.GetTxState(txHash).Current)
	}
}

func TestTxLifecycleDropped(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	txHash := common.HexToHash("0xabc")

	tracker.TrackMempool(txHash)
	tracker.TrackDropped(txHash, "gas price too low")

	state := tracker.GetTxState(txHash)
	if state.Current != TxDropped {
		t.Errorf("expected dropped, got %s", state.Current)
	}
}

func TestTxLifecycleSnapshot(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	tx1 := common.HexToHash("0x1")
	tx2 := common.HexToHash("0x2")

	tracker.TrackMempool(tx1)
	tracker.TrackIncluded(tx2, 100, common.HexToHash("0xblock"))

	snapshot := tracker.Snapshot()
	if len(snapshot) != 2 {
		t.Errorf("expected 2 in snapshot, got %d", len(snapshot))
	}
}

func TestTxLifecycleCountByPhase(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	tracker.TrackMempool(common.HexToHash("0x1"))
	tracker.TrackMempool(common.HexToHash("0x2"))
	tracker.TrackIncluded(common.HexToHash("0x3"), 100, common.HexToHash("0xb1"))

	counts := tracker.CountByPhase()
	if counts[TxInMempool] != 2 {
		t.Errorf("expected 2 in mempool, got %d", counts[TxInMempool])
	}
	if counts[TxProposed] != 1 {
		t.Errorf("expected 1 proposed, got %d", counts[TxProposed])
	}
}

func TestTxLifecycleTxByBlock(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	blockHash := common.HexToHash("0xblock1")

	tracker.TrackIncluded(common.HexToHash("0x1"), 100, blockHash)
	tracker.TrackIncluded(common.HexToHash("0x2"), 100, blockHash)

	txs := tracker.GetTxByBlock(blockHash)
	if len(txs) != 2 {
		t.Errorf("expected 2 txs in block, got %d", len(txs))
	}
}

func TestTxLifecycleUnknownTx(t *testing.T) {
	t.Parallel()

	tracker := NewTxLifecycleTracker()
	state := tracker.GetTxState(common.HexToHash("0xunknown"))
	if state != nil {
		t.Error("expected nil for unknown tx")
	}
}
