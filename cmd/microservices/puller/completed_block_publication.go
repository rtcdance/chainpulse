package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

func (r *pullerExecutionRuntime) publishCompleteRange(ctx context.Context, source *pullers.MultiChainDataPuller, chain string, from, to uint64, events []blockchain.BlockchainEvent) error {
	publisher, ok := r.publisher.(pullerKeyedMessagePublisher)
	if !ok || len(r.outputTopics) == 0 {
		return fmt.Errorf("complete block publication requires keyed publisher and topics")
	}
	frames, err := appindexing.BuildCompletedRange(ctx, chain, from, to, events, func(ctx context.Context, height uint64) (*types.Header, error) {
		return source.BlockHeaderFromChain(ctx, chain, height)
	})
	if err != nil {
		return err
	}
	for _, frame := range frames {
		payload, err := json.Marshal(frame)
		if err != nil {
			return fmt.Errorf("encode complete block: %w", err)
		}
		for _, topic := range r.outputTopics {
			if err := publisher.PublishMessage(ctx, mq.NewChainMessage(chain, topic, payload)); err != nil {
				return fmt.Errorf("publish complete block %d: %w", frame.Header.Number.Uint64(), err)
			}
			r.mu.Lock()
			r.publishedMessages++
			r.mu.Unlock()
		}
		r.mu.Lock()
		r.publishedEvents += int64(len(frame.Events))
		r.mu.Unlock()
	}
	r.mu.Lock()
	r.publishedTips[chain] = frames[len(frames)-1].Header
	r.mu.Unlock()

	return nil
}
