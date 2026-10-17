package indexing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

type subscriptionChainDelivery struct {
	gate    chan struct{}
	seen    map[string]struct{}
	keys    [1024]string
	next    int
	pending string
	index   int
}

// SubscriptionDelivery treats Kafka messages as notifications of committed
// facts. PG remains authoritative, including after a reorg or a restart.
type SubscriptionDelivery struct {
	store   ports.SubscriptionDeliveryStore
	sink    ports.SubscriptionDeliverySink
	chains  map[string]*subscriptionChainDelivery
	metrics core.MetricsCollector
}

func NewSubscriptionDelivery(store ports.SubscriptionDeliveryStore, sink ports.SubscriptionDeliverySink, chains []string, metrics core.MetricsCollector) (*SubscriptionDelivery, error) {
	if store == nil || sink == nil || len(chains) == 0 || len(chains) > 50 {
		return nil, fmt.Errorf("subscription delivery requires a store, sink and 1..50 declared chains")
	}
	d := &SubscriptionDelivery{store: store, sink: sink, chains: make(map[string]*subscriptionChainDelivery), metrics: metrics}
	for _, chain := range chains {
		if chain == "" || len(chain) > 64 || strings.TrimSpace(chain) != chain || d.chains[chain] != nil {
			return nil, fmt.Errorf("subscription delivery requires unique nonempty chain keys")
		}
		d.chains[chain] = &subscriptionChainDelivery{gate: make(chan struct{}, 1), seen: make(map[string]struct{})}
	}
	return d, nil
}

func (d *SubscriptionDelivery) Handle(ctx context.Context, message ports.SubscriptionDeliveryMessage) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(message.Payload) > 8<<20 {
		return fmt.Errorf("subscription notification exceeds the broker byte bound")
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(message.Payload, &header); err != nil {
		return d.quarantine(ctx, message, "invalid-envelope")
	}
	switch header.Schema {
	case CompletedBlockSchema:
		var frame CompletedBlock
		if err := json.Unmarshal(message.Payload, &frame); err != nil {
			return d.quarantine(ctx, message, "invalid-block")
		}
		if err := frame.Normalize(); err != nil || frame.Header.Number.Uint64() > math.MaxInt64 {
			return d.quarantine(ctx, message, "invalid-block")
		}
		return d.inChain(ctx, message, frame.ChainID, func(state *subscriptionChainDelivery) error {
			return d.block(ctx, message, frame, state)
		})
	case ReorgNoticeSchema:
		notice, err := ParseReorgNotice(message.Payload)
		if err != nil {
			return d.quarantine(ctx, message, "invalid-reorg")
		}
		return d.inChain(ctx, message, notice.ChainID, func(state *subscriptionChainDelivery) error {
			return d.reorg(ctx, message, notice, state)
		})
	default:
		return d.quarantine(ctx, message, "unsupported-schema")
	}
}

func (d *SubscriptionDelivery) inChain(ctx context.Context, message ports.SubscriptionDeliveryMessage, chain string, apply func(*subscriptionChainDelivery) error) error {
	state := d.chains[chain]
	if state == nil {
		return d.quarantine(ctx, message, "unconfigured-chain")
	}
	if message.PartitionKey != chain {
		return d.quarantine(ctx, message, "partition-chain-mismatch")
	}
	select {
	case state.gate <- struct{}{}:
		defer func() { <-state.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return apply(state)
}

func (d *SubscriptionDelivery) block(ctx context.Context, message ports.SubscriptionDeliveryMessage, frame CompletedBlock, state *subscriptionChainDelivery) error {
	record, err := d.store.ReadSubscriptionBlock(ctx, frame.ChainID, frame.Header.Number.Uint64(), frame.Header.Hash().Hex())
	if err != nil {
		return fmt.Errorf("read committed subscription block: %w", err)
	}
	if !record.Canonical {
		d.record(frame.ChainID, "orphaned")
		return nil
	}
	if len(record.Events) != len(frame.Events) {
		return d.quarantine(ctx, message, "canonical-payload-mismatch")
	}
	for i, event := range record.Events {
		input := frame.Events[i]
		if event == nil || event.ID != input.ID || event.ContractAddress != input.ContractAddress || !bytes.Equal(event.EventData, input.EventData) || !slices.Equal(event.EventTopic, input.EventTopic) {
			return d.quarantine(ctx, message, "canonical-payload-mismatch")
		}
	}
	key := "block/" + strconv.FormatInt(record.Generation, 10) + "/" + frame.Header.Hash().Hex()
	if _, ok := state.seen[key]; ok {
		d.record(frame.ChainID, "duplicate")
		return nil
	}
	if state.pending != key {
		state.pending, state.index = key, 0
	}
	for state.index < len(record.Events) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.sink.BroadcastEvent(ctx, record.Events[state.index]); err != nil {
			return fmt.Errorf("broadcast committed subscription event: %w", err)
		}
		state.index++
	}
	state.remember(key)
	state.pending, state.index = "", 0
	d.record(frame.ChainID, "delivered")
	return nil
}

func (d *SubscriptionDelivery) reorg(ctx context.Context, message ports.SubscriptionDeliveryMessage, notice ReorgNotice, state *subscriptionChainDelivery) error {
	record, err := d.store.ReadSubscriptionReorg(ctx, notice.ChainID, notice.ID)
	if err != nil {
		return fmt.Errorf("read durable subscription reorg: %w", err)
	}
	if !record.Found || record.ChainID != notice.ChainID || record.JobID != notice.JobID || record.Kind != notice.Kind || record.Generation != notice.Generation {
		return d.quarantine(ctx, message, "outbox-identity-mismatch")
	}
	if record.Generation < record.CurrentGeneration {
		d.record(notice.ChainID, "stale-reorg")
		return nil
	}
	if record.Generation > record.CurrentGeneration {
		return fmt.Errorf("durable reorg generation exceeds canonical authority")
	}
	stored := notice
	stored.Payload = record.Payload
	rollback, err := stored.Rollback()
	if err != nil {
		return fmt.Errorf("invalid durable reorg payload: %w", err)
	}
	input, err := notice.Rollback()
	if err != nil {
		return d.quarantine(ctx, message, "invalid-reorg")
	}
	if rollback == nil {
		d.record(notice.ChainID, "detected")
		return nil
	}
	if input.ChainID != rollback.ChainID || input.FromBlock != rollback.FromBlock || input.ToBlock != rollback.ToBlock || !input.DetectedAt.Equal(rollback.DetectedAt) {
		return d.quarantine(ctx, message, "outbox-payload-mismatch")
	}
	key := "reorg/" + strconv.FormatInt(notice.Generation, 10) + "/" + strconv.FormatInt(notice.ID, 10)
	if _, ok := state.seen[key]; ok {
		d.record(notice.ChainID, "duplicate")
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.sink.BroadcastReorg(ctx, rollback); err != nil {
		return fmt.Errorf("broadcast durable reorg correction: %w", err)
	}
	state.remember(key)
	d.record(notice.ChainID, "delivered")
	return nil
}

func (s *subscriptionChainDelivery) remember(key string) {
	delete(s.seen, s.keys[s.next])
	s.keys[s.next] = key
	s.next = (s.next + 1) % len(s.keys)
	s.seen[key] = struct{}{}
}

func (d *SubscriptionDelivery) quarantine(ctx context.Context, message ports.SubscriptionDeliveryMessage, reason string) error {
	if err := d.store.QuarantineSubscription(ctx, message, reason); err != nil {
		return fmt.Errorf("persist subscription quarantine: %w", err)
	}
	d.record("unknown", "quarantined")
	return nil
}

func (d *SubscriptionDelivery) record(chain, result string) {
	if d.metrics != nil {
		d.metrics.RecordCounter("subscription_delivery", 1, map[string]string{"chain_id": chain, "result": result})
	}
}
