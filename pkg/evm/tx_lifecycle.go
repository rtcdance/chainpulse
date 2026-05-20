package evm

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TxLifecyclePhase represents the stage a transaction has reached in its lifecycle.
//
// Ethereum transactions go through these phases:
//
//	Mempool (pending) → Proposed (in a block) → Confirmed (N confirmations) → Finalized (2 epochs)
//	                                                                        → Reorged (chain reorg)
//	                                                                        → Dropped (mempool eviction)
type TxLifecyclePhase string

const (
	// TxInMempool means the transaction is in the txpool but not yet in a block.
	TxInMempool TxLifecyclePhase = "mempool"

	// TxProposed means the transaction has been included in a block,
	// but the block has fewer than safe_blocks confirmations.
	TxProposed TxLifecyclePhase = "proposed"

	// TxConfirmed means the transaction is in a block with enough confirmations
	// to be considered safe (past the reorg window for this chain).
	TxConfirmed TxLifecyclePhase = "confirmed"

	// TxFinalized means the transaction is in a finalized block
	// (past the finality boundary — 2 epochs on Ethereum PoS).
	TxFinalized TxLifecyclePhase = "finalized"

	// TxReorged means the transaction was in a block that was reorged out.
	// The transaction may appear again in a different block or not at all.
	TxReorged TxLifecyclePhase = "reorged"

	// TxDropped means the transaction left the mempool without being included
	// (gas price too low, nonce gap, or replaced by a higher-gas tx).
	TxDropped TxLifecyclePhase = "dropped"
)

// TxPhaseMetadata holds additional context about a transaction's phase transition.
type TxPhaseMetadata struct {
	Phase       TxLifecyclePhase
	BlockNumber uint64
	BlockHash   common.Hash
	Timestamp   time.Time
	Reason      string
}

// TxLifecycleState tracks the current state and history of a single transaction.
type TxLifecycleState struct {
	TxHash     common.Hash
	Current    TxLifecyclePhase
	History    []TxPhaseMetadata
	FirstSeen  time.Time
	LastUpdate time.Time
}

// TxLifecycleTracker tracks the lifecycle of multiple transactions.
// It is safe for concurrent use.
type TxLifecycleTracker struct {
	mu     sync.RWMutex
	track  map[common.Hash]*TxLifecycleState
	lookup map[common.Hash]common.Hash // block_hash → first tx in that block (for reorg lookups)
}

// NewTxLifecycleTracker creates a new transaction lifecycle tracker.
func NewTxLifecycleTracker() *TxLifecycleTracker {
	return &TxLifecycleTracker{
		track:  make(map[common.Hash]*TxLifecycleState),
		lookup: make(map[common.Hash]common.Hash),
	}
}

// TrackMempool records a transaction seen in the mempool.
func (tlt *TxLifecycleTracker) TrackMempool(txHash common.Hash) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	if _, exists := tlt.track[txHash]; !exists {
		now := time.Now()
		tlt.track[txHash] = &TxLifecycleState{
			TxHash:     txHash,
			Current:    TxInMempool,
			History:    []TxPhaseMetadata{{Phase: TxInMempool, Timestamp: now, Reason: "seen in mempool"}},
			FirstSeen:  now,
			LastUpdate: now,
		}
	}
}

// TrackIncluded records a transaction included in a block.
// If the tracker already knows about it from mempool, it transitions from mempool → proposed.
// If not, it starts at proposed (common for archival indexing).
func (tlt *TxLifecycleTracker) TrackIncluded(txHash common.Hash, blockNumber uint64, blockHash common.Hash) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	now := time.Now()
	if state, exists := tlt.track[txHash]; exists {
		state.Current = TxProposed
		h := TxPhaseMetadata{Phase: TxProposed, BlockNumber: blockNumber, BlockHash: blockHash, Timestamp: now, Reason: "included in block"}
		state.History = append(state.History, h)
		state.LastUpdate = now
	} else {
		tlt.track[txHash] = &TxLifecycleState{
			TxHash:     txHash,
			Current:    TxProposed,
			History:    []TxPhaseMetadata{{Phase: TxProposed, BlockNumber: blockNumber, BlockHash: blockHash, Timestamp: now, Reason: "included in block"}},
			FirstSeen:  now,
			LastUpdate: now,
		}
	}
	tlt.lookup[blockHash] = txHash
}

// TrackConfirmed transitions a transaction from proposed → confirmed.
func (tlt *TxLifecycleTracker) TrackConfirmed(txHash common.Hash, blockNumber uint64) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	if state, exists := tlt.track[txHash]; exists {
		state.Current = TxConfirmed
		state.History = append(state.History, TxPhaseMetadata{
			Phase: TxConfirmed, BlockNumber: blockNumber, Timestamp: time.Now(), Reason: "sufficient confirmations",
		})
		state.LastUpdate = time.Now()
	}
}

// TrackFinalized transitions a transaction from confirmed → finalized.
func (tlt *TxLifecycleTracker) TrackFinalized(txHash common.Hash, epoch uint64) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	if state, exists := tlt.track[txHash]; exists {
		state.Current = TxFinalized
		state.History = append(state.History, TxPhaseMetadata{
			Phase: TxFinalized, BlockNumber: epoch, Timestamp: time.Now(), Reason: "epoch finalized",
		})
		state.LastUpdate = time.Now()
	}
}

// TrackReorged marks all transactions in the given block as reorged.
func (tlt *TxLifecycleTracker) TrackReorged(blockHash common.Hash, reorgedBlockNumber uint64) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	now := time.Now()
	for _, state := range tlt.track {
		last := state.History[len(state.History)-1]
		if last.BlockHash == blockHash {
			state.Current = TxReorged
			state.History = append(state.History, TxPhaseMetadata{
				Phase: TxReorged, BlockNumber: reorgedBlockNumber, Timestamp: now, Reason: "block reorged",
			})
			state.LastUpdate = now
		}
	}
}

// TrackDropped marks a mempool transaction as dropped.
func (tlt *TxLifecycleTracker) TrackDropped(txHash common.Hash, reason string) {
	tlt.mu.Lock()
	defer tlt.mu.Unlock()

	if state, exists := tlt.track[txHash]; exists {
		state.Current = TxDropped
		state.History = append(state.History, TxPhaseMetadata{
			Phase: TxDropped, Timestamp: time.Now(), Reason: reason,
		})
		state.LastUpdate = time.Now()
	}
}

// GetTxState returns the current lifecycle state for a transaction.
func (tlt *TxLifecycleTracker) GetTxState(txHash common.Hash) *TxLifecycleState {
	tlt.mu.RLock()
	defer tlt.mu.RUnlock()
	return tlt.track[txHash]
}

// GetTxByBlock returns the known state for all transactions in a given block.
func (tlt *TxLifecycleTracker) GetTxByBlock(blockHash common.Hash) []*TxLifecycleState {
	tlt.mu.RLock()
	defer tlt.mu.RUnlock()

	var result []*TxLifecycleState
	for _, state := range tlt.track {
		if len(state.History) > 0 {
			last := state.History[len(state.History)-1]
			if last.BlockHash == blockHash {
				result = append(result, state)
			}
		}
	}
	return result
}

// Snapshot returns all tracked transaction states at this moment.
func (tlt *TxLifecycleTracker) Snapshot() []*TxLifecycleState {
	tlt.mu.RLock()
	defer tlt.mu.RUnlock()

	result := make([]*TxLifecycleState, 0, len(tlt.track))
	for _, state := range tlt.track {
		s := *state
		s.History = make([]TxPhaseMetadata, len(state.History))
		copy(s.History, state.History)
		result = append(result, &s)
	}
	return result
}

// CountByPhase returns the count of tracked transactions in each phase.
func (tlt *TxLifecycleTracker) CountByPhase() map[TxLifecyclePhase]int {
	tlt.mu.RLock()
	defer tlt.mu.RUnlock()

	counts := make(map[TxLifecyclePhase]int)
	for _, state := range tlt.track {
		counts[state.Current]++
	}
	return counts
}
