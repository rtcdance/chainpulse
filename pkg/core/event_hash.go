package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ComputeEventHash produces a deterministic SHA-256 hash for a blockchain event
// using its natural uniqueness key: (chain_id, block_number, transaction_hash, log_index).
//
// This is the single canonical hash function for the entire system. All components
// (pullers, idempotency service, event processor, database dedup) MUST use this
// function to ensure consistent duplicate detection.
//
// The hash input does NOT include derived fields like EventName, Network, or
// ContractAddress — those can change across reorgs or contract upgrades but the
// natural key is immutable on-chain.
func ComputeEventHash(event *blockchain.BlockchainEvent) string {
	if event == nil {
		return ""
	}

	// Format: chain_id:block_number:tx_hash:log_index
	// Using Hex() for TransactionHash ensures consistent representation
	// regardless of whether the hash was set as a common.Hash or a string.
	hashInput := fmt.Sprintf(
		"%s:%d:%s:%d",
		event.ChainID,
		event.BlockNumber,
		event.TransactionHash.Hex(),
		event.LogIndex,
	)

	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// EventNaturalKey returns the human-readable natural key tuple for an event.
// Useful for logging and debugging.
func EventNaturalKey(event *blockchain.BlockchainEvent) string {
	if event == nil {
		return ""
	}
	return fmt.Sprintf(
		"%s:%d:%s:%d",
		event.ChainID,
		event.BlockNumber,
		event.TransactionHash.Hex(),
		event.LogIndex,
	)
}

// ErrDuplicateEvent indicates that an event with the same natural key already exists.
var ErrDuplicateEvent = fmt.Errorf("duplicate event: natural key already exists")
