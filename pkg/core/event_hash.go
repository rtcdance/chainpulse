package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ComputeEventHash produces a deterministic SHA-256 hash for a blockchain event
// using its physical identity (invariant I02):
//
//	(chain_id, block_hash, transaction_hash, log_index)
//
// block_hash is part of the identity, not a derived field. The same
// (chain, tx_hash, log_index) can exist on two competing branches at the same
// height; without the block hash a reorg that replaces a block would collide
// with the block it replaced (finding F06). Block *number* is deliberately not
// part of the input: it is used for ordering and range scans, and is implied by
// the block hash.
//
// This is the single canonical hash function for the entire system. All
// components (pullers, idempotency service, event processor, database dedup)
// MUST use this function to ensure consistent duplicate detection.
func ComputeEventHash(event *blockchain.BlockchainEvent) string {
	if event == nil {
		return ""
	}

	// Format: chain_id:block_hash:tx_hash:log_index
	// Using Hex() for the hashes ensures a consistent representation regardless
	// of whether they were set as common.Hash or a string.
	hashInput := fmt.Sprintf(
		"%s:%s:%s:%d",
		event.ChainID,
		event.BlockHash.Hex(),
		event.TransactionHash.Hex(),
		event.LogIndex,
	)

	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// EventNaturalKey returns the human-readable physical identity tuple for an
// event, matching ComputeEventHash's input. Useful for logging and debugging.
func EventNaturalKey(event *blockchain.BlockchainEvent) string {
	if event == nil {
		return ""
	}
	return fmt.Sprintf(
		"%s:%s:%s:%d",
		event.ChainID,
		event.BlockHash.Hex(),
		event.TransactionHash.Hex(),
		event.LogIndex,
	)
}

// ErrDuplicateEvent indicates that an event with the same natural key already exists.
var ErrDuplicateEvent = fmt.Errorf("duplicate event: natural key already exists")
