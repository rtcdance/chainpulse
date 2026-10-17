package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type completeBlockWriter interface {
	ProcessCompleteBlock(context.Context, appindexing.CompletedBlock) error
}

// ingestCompleteRange validates the complete RPC result before writing any
// block. A block's entire log group shares one commit and one checkpoint.
func ingestCompleteRange(ctx context.Context, chainID string, from, to uint64, events []blockchain.BlockchainEvent, header func(context.Context, uint64) (*types.Header, error), writer completeBlockWriter, afterBlock func(*types.Header) error) error {
	frames, err := appindexing.BuildCompletedRange(ctx, chainID, from, to, events, header)
	if err != nil {
		return err
	}
	for _, frame := range frames {
		if err := writer.ProcessCompleteBlock(ctx, frame); err != nil {
			return fmt.Errorf("commit completed block %d: %w", frame.Header.Number.Uint64(), err)
		}
		if afterBlock != nil {
			if err := afterBlock(frame.Header); err != nil {
				return fmt.Errorf("record completed block: %w", err)
			}
		}
	}
	return nil
}
