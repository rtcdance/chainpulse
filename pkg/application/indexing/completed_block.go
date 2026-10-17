package indexing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

const CompletedBlockSchema = "chainpulse.block.v1"

// CompletedBlock transports the entire result of one completed eth_getLogs
// block, including empty blocks. Kafka acknowledgement belongs to this unit.
// Header is carried in its canonical encoding so its hash survives transport.
type CompletedBlock struct {
	Schema  string                       `json:"schema"`
	ChainID string                       `json:"chain_id"`
	Header  *types.Header                `json:"header"`
	Events  []blockchain.BlockchainEvent `json:"events"`
}

type CompleteBlockCommitter interface {
	CommitChainCompleteBlock(context.Context, string, *blockchain.Block, []*blockchain.BlockchainEvent) error
}

// ProcessCompleteBlock uses the same atomic commit contract in both deployment
// modes. The transport's full header never becomes a separate side effect.
func (rt *SharedRuntime) ProcessCompleteBlock(ctx context.Context, block CompletedBlock) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := block.Normalize(); err != nil {
		return err
	}
	rt.mu.RLock()
	started, committer := rt.status.Started, rt.committer
	rt.mu.RUnlock()
	writer, ok := committer.(CompleteBlockCommitter)
	if !started || !ok {
		return fmt.Errorf("started runtime and complete block committer are required")
	}
	checkpoint, err := rt.ChainCheckpoint(ctx, block.ChainID)
	if err != nil {
		return fmt.Errorf("read complete block checkpoint: %w", err)
	}
	height := block.Header.Number.Uint64()
	if height > checkpoint.BlockNumber && height-checkpoint.BlockNumber != 1 {
		return &BlockGapError{Block: height, Checkpoint: checkpoint.BlockNumber}
	}
	events := make([]*blockchain.BlockchainEvent, len(block.Events))
	for i := range block.Events {
		events[i] = &block.Events[i]
	}
	snapshot := &blockchain.Block{Number: block.Header.Number.Uint64(), Hash: block.Header.Hash(), ParentHash: block.Header.ParentHash, Timestamp: int64(block.Header.Time)}
	if err := writer.CommitChainCompleteBlock(ctx, block.ChainID, snapshot, events); err != nil {
		return fmt.Errorf("commit completed block %d: %w", snapshot.Number, err)
	}
	rt.recordAtomicProgress(block.ChainID, len(events), snapshot.Number)
	return nil
}

// UnmarshalJSON preserves numeric lexemes in decoded projections. Default
// interface decoding through float64 would corrupt large asset values.
func (b *CompletedBlock) UnmarshalJSON(payload []byte) error {
	type envelope CompletedBlock
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode((*envelope)(b)); err != nil {
		return fmt.Errorf("decode completed block: %w", err)
	}
	return nil
}

// Normalize checks all identities before an atomic writer sees any event.
// EVM log indexes are block-wide, so two records with the same index indicate
// an incomplete or malformed RPC result, even if their transaction IDs differ.
func (b *CompletedBlock) Normalize() error {
	if b.Schema != CompletedBlockSchema || b.ChainID == "" || b.Header == nil || b.Header.Number == nil || !b.Header.Number.IsUint64() || b.Header.Time > math.MaxInt64 {
		return fmt.Errorf("invalid completed block envelope")
	}
	if len(b.Events) > 10000 {
		return fmt.Errorf("completed block exceeds the bounded log count")
	}
	hash := b.Header.Hash()
	indexes := make(map[uint64]struct{}, len(b.Events))
	for i := range b.Events {
		event := &b.Events[i]
		if event.ChainID != b.ChainID || event.BlockNumber != b.Header.Number.Uint64() || event.BlockHash != hash || event.Removed {
			return fmt.Errorf("log %d does not belong to completed canonical block", i)
		}
		if _, exists := indexes[event.LogIndex]; exists {
			return fmt.Errorf("duplicate log index %d in completed block", event.LogIndex)
		}
		indexes[event.LogIndex] = struct{}{}
		event.BlockTimestamp = int64(b.Header.Time)
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
	}
	sort.Slice(b.Events, func(i, j int) bool {
		if b.Events[i].TransactionIndex != b.Events[j].TransactionIndex {
			return b.Events[i].TransactionIndex < b.Events[j].TransactionIndex
		}
		return b.Events[i].LogIndex < b.Events[j].LogIndex
	})
	return nil
}

// BuildCompletedRange validates the complete RPC result before any transport or
// writer can advance. Both native deployment modes use this same boundary.
func BuildCompletedRange(ctx context.Context, chain string, from, to uint64, events []blockchain.BlockchainEvent, readHeader func(context.Context, uint64) (*types.Header, error)) ([]CompletedBlock, error) {
	if to < from || to-from >= 10000 || len(events) > 100000 || readHeader == nil {
		return nil, fmt.Errorf("invalid or unbounded completed RPC range")
	}
	groups := make(map[uint64][]blockchain.BlockchainEvent)
	for _, event := range events {
		if event.BlockNumber < from || event.BlockNumber > to {
			return nil, fmt.Errorf("log outside completed RPC range")
		}
		groups[event.BlockNumber] = append(groups[event.BlockNumber], event)
	}
	frames := make([]CompletedBlock, 0, to-from+1)
	for height := from; ; height++ {
		header, err := readHeader(ctx, height)
		if err != nil {
			return nil, fmt.Errorf("read completed block %d: %w", height, err)
		}
		frame := CompletedBlock{Schema: CompletedBlockSchema, ChainID: chain, Header: header, Events: groups[height]}
		if err := frame.Normalize(); err != nil {
			return nil, fmt.Errorf("validate completed block %d: %w", height, err)
		}
		if header.Number.Uint64() != height {
			return nil, fmt.Errorf("RPC header height does not match requested block")
		}
		if len(frames) > 0 && header.ParentHash != frames[len(frames)-1].Header.Hash() {
			return nil, fmt.Errorf("RPC range changed branch during collection")
		}
		frames = append(frames, frame)
		if height == to {
			break
		}
	}
	return frames, nil
}
