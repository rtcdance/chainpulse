// Package digest computes canonical, order-independent digests over indexed
// EVM events and canonical chain state.
//
// Task: T02 of docs/specs/2026-10-12-web3-go-backend-delivery-plan.md (F21).
//
// Why this exists
// ---------------
// The acceptance scenarios (E01-E13) and invariants I02/I03 need one
// reproducible summary of "what was indexed". Comparing raw row dumps is
// brittle: row order is not defined, Kafka redelivery duplicates rows, and a
// reorg rewrites history. This package defines the canonical form once, so
// every digest assertion in the plan compares like with like.
//
// Invariants encoded here
// -----------------------
//
//	I02  physical event ID = (chain key, block_hash, tx_hash, log_index)
//	I03  at most one canonical block per (chain, height)
//
// Guarantees
// ----------
//   - Stable: identical logical content yields an identical digest regardless
//     of input order (stable sort) or duplicate redelivery (dedup by ID).
//   - Strict: two events sharing an ID but differing in payload is a conflict,
//     reported as an error rather than silently resolved.
//   - Chain-scoped: the chain key is part of every record, so chain A and
//     chain B never collide (I01).
package digest

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Event is the canonical identity plus payload of one indexed log.
type Event struct {
	Chain     string   // canonical chain key, e.g. "1" for Ethereum mainnet
	BlockHash string   // 0x-prefixed block hash
	TxHash    string   // 0x-prefixed transaction hash
	LogIndex  uint32   // index of the log within the block
	Topics    []string // raw topics, in order
	Data      string   // raw data (hex)
}

// ID returns the physical event identity (I02). Hashes are lower-cased so that
// checksum-cased and lower-cased forms of the same hash are one identity.
func (e Event) ID() string {
	return e.Chain + "|" +
		strings.ToLower(e.BlockHash) + "|" +
		strings.ToLower(e.TxHash) + "|" +
		strconv.FormatUint(uint64(e.LogIndex), 10)
}

// payload is the part of an event that must be identical for two events that
// share an ID. It deliberately excludes the identity fields themselves.
func (e Event) payload() string {
	topics := make([]string, len(e.Topics))
	for i, t := range e.Topics {
		topics[i] = strings.ToLower(t)
	}
	return strings.Join(topics, ",") + "|" + strings.ToLower(e.Data)
}

// Block is a canonical chain block (I03).
type Block struct {
	Chain     string // canonical chain key
	Height    uint64
	BlockHash string
}

// CanonicalEvents returns a stable SHA-256 digest over the given events.
//
// The digest is order-independent and idempotent: feeding the same events in
// any order, with any number of duplicate redeliveries, produces the same
// value. A duplicate ID with a differing payload returns an error.
func CanonicalEvents(events []Event) (string, error) {
	// Dedup by ID, detecting conflicting payloads.
	byID := make(map[string]Event, len(events))
	for _, e := range events {
		if e.Chain == "" {
			return "", fmt.Errorf("digest: event with empty chain key (I01): %+v", e)
		}
		id := e.ID()
		if prev, ok := byID[id]; ok {
			if prev.payload() != e.payload() {
				return "", fmt.Errorf("digest: conflicting payload for event %s", id)
			}
			continue // idempotent: identical redelivery collapses
		}
		byID[id] = e
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids) // stable order

	h := sha256.New()
	for _, id := range ids {
		e := byID[id]
		writeField(h, "chain", e.Chain)
		writeField(h, "block_hash", strings.ToLower(e.BlockHash))
		writeField(h, "tx_hash", strings.ToLower(e.TxHash))
		writeField(h, "log_index", strconv.FormatUint(uint64(e.LogIndex), 10))
		writeField(h, "topics", strings.Join(e.Topics, ","))
		writeField(h, "data", e.Data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CanonicalBlocks returns a stable SHA-256 digest over the canonical chain
// blocks (I03). Two blocks at the same (chain, height) with different hashes
// are a conflict and return an error.
func CanonicalBlocks(blocks []Block) (string, error) {
	type key struct {
		chain  string
		height uint64
	}
	byKey := make(map[key]string, len(blocks))
	for _, b := range blocks {
		if b.Chain == "" {
			return "", fmt.Errorf("digest: block with empty chain key (I01): %+v", b)
		}
		k := key{b.Chain, b.Height}
		hash := strings.ToLower(b.BlockHash)
		if prev, ok := byKey[k]; ok {
			if prev != hash {
				return "", fmt.Errorf(
					"digest: two canonical blocks at %s height %d (%s vs %s) — I03 violated",
					b.Chain, b.Height, prev, hash,
				)
			}
			continue
		}
		byKey[k] = hash
	}

	keys := make([]key, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].chain != keys[j].chain {
			return keys[i].chain < keys[j].chain
		}
		return keys[i].height < keys[j].height
	})

	h := sha256.New()
	for _, k := range keys {
		writeField(h, "chain", k.chain)
		writeField(h, "height", strconv.FormatUint(k.height, 10))
		writeField(h, "block_hash", byKey[k])
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MustCanonicalEvents is CanonicalEvents for tests and callers that treat a
// conflict as a programming error.
func MustCanonicalEvents(events []Event) string {
	d, err := CanonicalEvents(events)
	if err != nil {
		panic(err)
	}
	return d
}

// MustCanonicalBlocks is CanonicalBlocks for tests.
func MustCanonicalBlocks(blocks []Block) string {
	d, err := CanonicalBlocks(blocks)
	if err != nil {
		panic(err)
	}
	return d
}

// writeField writes a length-prefixed field so that concatenation is
// unambiguous (avoids "ab"+"c" colliding with "a"+"bc").
func writeField(h interface{ Write([]byte) (int, error) }, name, value string) {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(name)))
	_, _ = h.Write(lenBuf[:])
	_, _ = h.Write([]byte(name))

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(value)))
	_, _ = h.Write(lenBuf[:])
	_, _ = h.Write([]byte(value))
}
