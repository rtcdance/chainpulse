package mq

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/segmentio/kafka-go"
)

// ConsumerOwnershipHooks bind effects to one revocable broker generation.
// Assigned runs before handlers. Revoked runs after all handlers exit, with a
// fresh cleanup deadline; it cannot authorize any new processing work.
type ConsumerOwnershipHooks struct {
	Assigned func(context.Context, string) error
	Revoked  func(context.Context) error
}

const maxOwnedPartitions = 16

// ConsumeOwnedMessages propagates broker revocation into the effect context.
// One bounded reader per assigned partition preserves offset order. A handler
// failure retries the same record, so a later success cannot skip a failed one.
func (p *KafkaMQPlugin) ConsumeOwnedMessages(ctx context.Context, topic string, handler func(context.Context, core.MessageQueueMessage) error, hooks ConsumerOwnershipHooks) (result error) {
	if ctx == nil || topic == "" || handler == nil || hooks.Assigned == nil || hooks.Revoked == nil {
		return fmt.Errorf("owned consumption requires context, topic, handler and ownership hooks")
	}
	ownedCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	p.mu.Lock()
	if !p.isRunning || p.ownedConsumers[topic] != nil {
		p.mu.Unlock()
		return fmt.Errorf("owned consumer is stopped or already registered")
	}
	p.ownedConsumers[topic] = cancel
	p.inFlight.Add(1)
	maxBytes := p.consumerMaxBytes
	p.mu.Unlock()
	defer func() { p.mu.Lock(); delete(p.ownedConsumers, topic); p.mu.Unlock(); p.inFlight.Done() }()
	if maxBytes <= 0 {
		maxBytes = defaultKafkaMaxBytes
	}
	if maxBytes > defaultKafkaMaxBytes {
		return fmt.Errorf("owned fetch byte limit exceeds safety cap %d", int(defaultKafkaMaxBytes))
	}
	group, err := kafka.NewConsumerGroup(kafka.ConsumerGroupConfig{ID: p.consumerGroup, Brokers: p.brokers, Topics: []string{topic}, StartOffset: kafka.FirstOffset, HeartbeatInterval: time.Second, SessionTimeout: 10 * time.Second, RebalanceTimeout: 10 * time.Second, Timeout: 5 * time.Second, JoinGroupBackoff: 250 * time.Millisecond})
	if err != nil {
		return fmt.Errorf("create owned consumer group: %w", err)
	}
	var cleanup <-chan struct{}
	var cleanupErr error
	defer func() {
		result = errors.Join(result, group.Close())
		if cleanup != nil {
			<-cleanup
		}
		result = errors.Join(result, cleanupErr)
	}()
	for {
		generation, err := group.Next(ownedCtx)
		// Also owns the library's closed-generation Start edge case, whose
		// callback may no longer be counted by ConsumerGroup.Next.
		if cleanup != nil {
			<-cleanup
		}
		if cleanupErr != nil {
			return cleanupErr
		}
		if err != nil {
			return fmt.Errorf("next owned generation: %w", err)
		}
		assignments := generation.Assignments[topic]
		if len(assignments) > maxOwnedPartitions {
			return fmt.Errorf("assigned partition count exceeds safety cap %d", maxOwnedPartitions)
		}
		token := fmt.Sprintf("kafka-%x", sha256.Sum256(fmt.Appendf(nil, "%s/%s/%s/%d", generation.GroupID, topic, generation.MemberID, generation.ID)))
		if err := hooks.Assigned(ownedCtx, token); err != nil {
			releaseCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			return errors.Join(fmt.Errorf("assign owned generation: %w", err), hooks.Revoked(releaseCtx))
		}
		p.metricsCollector.RecordCounter("mq_generations_assigned", 1, map[string]string{"topic": topic})
		var workers sync.WaitGroup
		workers.Add(len(assignments))
		done := make(chan struct{})
		cleanup = done
		generation.Start(func(gctx context.Context) {
			defer close(done)
			select {
			case <-gctx.Done():
			case <-ownedCtx.Done():
			}
			workers.Wait()
			releaseCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			cleanupErr = hooks.Revoked(releaseCtx)
			p.metricsCollector.RecordCounter("mq_generations_revoked", 1, map[string]string{"topic": topic})
		})
		for _, assignment := range assignments {
			generation.Start(func(gctx context.Context) {
				defer workers.Done()
				workerCtx, stop := context.WithCancel(ownedCtx)
				stopRevocation := context.AfterFunc(gctx, stop)
				defer func() { stopRevocation(); stop() }()
				active := ownedGenerationContext{Context: workerCtx, generation: gctx}
				if err := p.consumeOwnedPartition(active, generation, topic, assignment, int(maxBytes), handler); err != nil && active.Err() == nil {
					p.logger.Error("Owned partition stopped", "topic", topic, "partition", assignment.ID, "error", err.Error())
				}
			})
		}
	}
}

func (p *KafkaMQPlugin) consumeOwnedPartition(ctx context.Context, generation *kafka.Generation, topic string, assignment kafka.PartitionAssignment, maxBytes int, handler func(context.Context, core.MessageQueueMessage) error) (result error) {
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: p.brokers, Topic: topic, Partition: assignment.ID, QueueCapacity: 1, MinBytes: 1, MaxBytes: maxBytes, MaxWait: 250 * time.Millisecond, ReadBatchTimeout: time.Second})
	defer func() { result = errors.Join(result, reader.Close()) }()
	if err := reader.SetOffset(assignment.Offset); err != nil {
		return fmt.Errorf("set assigned offset: %w", err)
	}
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch assigned partition: %w", err)
		}
		queued, err := ownedQueueMessage(topic, message)
		if err != nil {
			return err
		}
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := handler(ctx, queued)
			if err == nil {
				break
			}
			if cancellation := ctx.Err(); cancellation != nil {
				return cancellation
			}
			p.metricsCollector.RecordCounter("mq_handler_errors", 1, map[string]string{"topic": topic})
			p.logger.Warn("Owned handler retry", "topic", topic, "partition", message.Partition, "offset", message.Offset, "error", err.Error())
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := generation.CommitOffsets(map[string]map[int]int64{topic: {assignment.ID: message.Offset + 1}}); err != nil {
			return fmt.Errorf("commit active generation offset: %w", err)
		}
		p.mu.Lock()
		p.stats.messageCount++
		p.mu.Unlock()
		p.offsets.mu.Lock()
		if previous, ok := p.offsets.tracking[topic]; !ok || message.Offset > previous {
			p.offsets.tracking[topic] = message.Offset
		}
		p.offsets.mu.Unlock()
		p.metricsCollector.RecordCounter("mq_messages_consumed", 1, map[string]string{"topic": topic})
	}
}

func ownedQueueMessage(topic string, message kafka.Message) (core.MessageQueueMessage, error) {
	id, key := "", string(message.Key)
	for _, header := range message.Headers {
		if header.Key == "partition_key" && string(header.Value) != key {
			return core.MessageQueueMessage{}, fmt.Errorf("kafka partition header does not match actual record key")
		}
		if header.Key == "message_id" {
			id = string(header.Value)
		}
	}
	return core.MessageQueueMessage{ID: id, Topic: topic, Payload: message.Value, Timestamp: message.Time, Offset: message.Offset, PartitionKey: key}, nil
}

// Err observes revocation synchronously, including before the cancellation
// callback has propagated to a database driver's waiter. Values/deadlines
// remain inherited from the caller for tracing and request budgets.
type ownedGenerationContext struct {
	context.Context
	generation context.Context
}

func (c ownedGenerationContext) Err() error {
	if c.generation.Err() != nil {
		return context.Canceled
	}
	return c.Context.Err()
}
