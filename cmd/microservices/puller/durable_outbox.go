package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

// One bounded pass per poll also runs when the chain has no new eligible block.
// The durable id and generation travel in every record for consumer deduplication.
func (r *pullerExecutionRuntime) drainDurableNotifications(ctx context.Context, topic string) error {
	if r.durableStore == nil {
		return nil
	}
	if topic == "" {
		return fmt.Errorf("durable reorg output topic is required")
	}
	publisher, ok := r.publisher.(pullerKeyedMessagePublisher)
	if !ok {
		return fmt.Errorf("durable outbox requires chain-keyed publication")
	}
	pending, err := r.durableStore.PendingReorgNotifications(ctx, 100)
	if err != nil {
		return err
	}
	for _, notice := range pending {
		payload, err := encodeDurableReorgNotice(notice)
		if err != nil {
			return fmt.Errorf("encode durable reorg notice: %w", err)
		}
		if err := publisher.PublishMessage(ctx, mq.NewChainMessage(notice.ChainID, topic, payload)); err != nil {
			return errors.Join(fmt.Errorf("publish durable reorg notice: %w", err), r.durableStore.RecordReorgNotificationFailure(ctx, notice.ID))
		}
		if err := r.durableStore.MarkReorgNotificationPublished(ctx, notice.ID); err != nil {
			return err
		}
	}
	return nil
}

func encodeDurableReorgNotice(notice database.ReorgNotification) ([]byte, error) {
	payload, err := json.Marshal(appindexing.ReorgNotice{
		Schema: appindexing.ReorgNoticeSchema, ID: notice.ID, JobID: notice.JobID,
		ChainID: notice.ChainID, Generation: notice.Generation, Kind: notice.Topic, Payload: notice.Payload,
	})
	if err != nil {
		return nil, fmt.Errorf("encode durable reorg envelope: %w", err)
	}
	return payload, nil
}
