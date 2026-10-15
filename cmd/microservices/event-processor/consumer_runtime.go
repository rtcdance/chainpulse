package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

type eventProcessorMessageConsumer interface {
	ConsumeMessages(context.Context, string, func(core.MessageQueueMessage) error) error
}

type eventProcessorMessageProcessor interface {
	ProcessEvent(context.Context, *blockchain.BlockchainEvent) error
	Health() *core.HealthStatus
	GetProcessedCount() int64
	GetFailedCount() int64
	GetDuplicateCount() int64
}

type completeBlockMessageProcessor interface {
	ProcessCompleteBlock(context.Context, appindexing.CompletedBlock) error
}

type generationOwnedProcessor interface {
	AssignOwnership(context.Context, string) error
	RevokeOwnership(context.Context) error
}

type consumeTransportError struct {
	message string
	at      int64
}

type eventProcessorConsumeLoopSnapshot struct {
	ConfiguredTopics  int
	ActiveTopics      int
	OwnedTopics       int
	OwnershipRequired bool
	Running           bool
	Paused            bool
	State             string
	LastError         string
	LastErrorAt       int64
	LastAction        string
	Reason            string
	UpdatedUnix       int64
}

type eventProcessorConsumeRuntime struct {
	logger       portcontract.Logger
	metrics      core.MetricsCollector
	consumer     eventProcessorMessageConsumer
	processor    eventProcessorMessageProcessor
	publisher    eventProcessorMessagePublisher
	topics       []string
	outputTopics []string
	dlqDB        *sql.DB // PostgreSQL for DLQ persistence

	mu              sync.RWMutex
	running         bool
	paused          bool
	activeTopics    map[string]bool
	activeCancels   map[string]context.CancelFunc
	ownedTopics     map[string]bool
	transportErrors map[string]consumeTransportError
	lastError       string
	lastErrorAtUnix int64
	lastAction      string
	reason          string
	updatedAtUnix   int64
	waitCh          chan struct{}
}

type eventProcessorMessagePublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

func newEventProcessorConsumeRuntime(
	logger portcontract.Logger,
	metrics core.MetricsCollector,
	consumer eventProcessorMessageConsumer,
	processor eventProcessorMessageProcessor,
	publisher eventProcessorMessagePublisher,
	topics []string,
	outputTopics []string,
	dlqDB *sql.DB,
) *eventProcessorConsumeRuntime {
	return &eventProcessorConsumeRuntime{
		logger:          logger,
		metrics:         metrics,
		consumer:        consumer,
		processor:       processor,
		publisher:       publisher,
		topics:          append([]string(nil), topics...),
		outputTopics:    append([]string(nil), outputTopics...),
		dlqDB:           dlqDB,
		activeTopics:    make(map[string]bool),
		activeCancels:   make(map[string]context.CancelFunc),
		ownedTopics:     make(map[string]bool),
		transportErrors: make(map[string]consumeTransportError),
		waitCh:          make(chan struct{}),
	}
}

func (r *eventProcessorConsumeRuntime) Start(ctx context.Context, wg *sync.WaitGroup) {
	if r == nil || ctx == nil || wg == nil || r.consumer == nil || r.processor == nil {
		return
	}

	r.mu.Lock()
	r.running = true
	r.mu.Unlock()

	for _, topic := range r.topics {
		topic := topic
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				if !r.waitUntilIntakeAllowed(ctx) {
					return
				}

				consumeCtx, cancel := context.WithCancel(ctx)
				r.setTopicControl(topic, true, cancel)

				err := r.consumeTopic(consumeCtx, topic, func(message core.MessageQueueMessage) error {
					if blocks, ok := r.processor.(completeBlockMessageProcessor); ok {
						return r.processCompleteBlockMessage(consumeCtx, message, blocks)
					}
					event, err := decodeEventProcessorQueueMessage(message)
					if err != nil {
						r.recordError(fmt.Errorf("decode topic %s message %s: %w", topic, message.ID, err))
						return err
					}
					// ConsumeMessages commits the broker offset only after
					// this callback succeeds. An in-memory enqueue cannot
					// substitute for the durable processing result.
					if err := r.processor.ProcessEvent(consumeCtx, event); err != nil {
						r.recordError(fmt.Errorf("process topic %s event %s: %w", topic, event.ID, err))
						r.writeToDLQ(consumeCtx, event, err)
						return fmt.Errorf("persist event %s: %w", event.ID, err)
					}
					if err := r.publishProcessedEvent(consumeCtx, event); err != nil {
						return err
					}
					if r.metrics != nil {
						r.metrics.RecordCounter("event_processor_consume_processed", 1, map[string]string{"topic": topic})
					}
					return nil
				})

				r.setTopicControl(topic, false, nil)
				cancel()

				if ctx.Err() != nil {
					return
				}
				if errors.Is(err, context.Canceled) {
					continue
				}
				if err != nil {
					r.recordTransportError(topic, fmt.Errorf("consume topic %s: %w", topic, err))
					backoff := time.NewTimer(250 * time.Millisecond)
					select {
					case <-ctx.Done():
						backoff.Stop()
						return
					case <-backoff.C:
					}
				}
			}
		}()
	}
}

func (r *eventProcessorConsumeRuntime) processCompleteBlockMessage(ctx context.Context, message core.MessageQueueMessage, processor completeBlockMessageProcessor) error {
	var frame appindexing.CompletedBlock
	if err := json.Unmarshal(message.Payload, &frame); err != nil {
		return fmt.Errorf("decode complete block: %w", err)
	}
	if err := frame.Normalize(); err != nil {
		return err
	}
	if message.PartitionKey != frame.ChainID {
		return fmt.Errorf("complete block partition key does not match chain")
	}
	if err := processor.ProcessCompleteBlock(ctx, frame); err != nil {
		var orphan *appindexing.OrphanedBlockError
		if errors.As(err, &orphan) {
			r.clearCompleteBlockError()
			if r.metrics != nil {
				r.metrics.RecordCounter("event_processor_orphaned_blocks", 1, map[string]string{"chain_id": frame.ChainID})
			}
			return nil
		}
		return fmt.Errorf("persist complete block: %w", err)
	}
	if r.publisher != nil && len(r.outputTopics) > 0 {
		publisher, ok := r.publisher.(interface {
			PublishMessage(context.Context, core.MessageQueueMessage) error
		})
		if !ok {
			return fmt.Errorf("committed block publication requires chain-keyed publisher")
		}
		payload, err := json.Marshal(frame)
		if err != nil {
			return fmt.Errorf("encode committed block: %w", err)
		}
		for _, topic := range r.outputTopics {
			if err := publisher.PublishMessage(ctx, mq.NewChainMessage(frame.ChainID, topic, payload)); err != nil {
				return fmt.Errorf("publish committed block: %w", err)
			}
		}
	}
	r.clearCompleteBlockError()
	return nil
}

func (r *eventProcessorConsumeRuntime) clearCompleteBlockError() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = ""
	r.lastErrorAtUnix = 0
}

func (r *eventProcessorConsumeRuntime) PauseIntake(reason string) eventProcessorConsumeLoopSnapshot {
	if r == nil {
		return eventProcessorConsumeLoopSnapshot{}
	}

	r.mu.Lock()
	r.paused = true
	r.reason = reason
	r.lastAction = "pause-intake"
	r.updatedAtUnix = time.Now().Unix()
	cancels := make([]context.CancelFunc, 0, len(r.activeCancels))
	for _, cancel := range r.activeCancels {
		cancels = append(cancels, cancel)
	}
	r.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	return r.Snapshot()
}

func (r *eventProcessorConsumeRuntime) ResumeIntake(reason string) eventProcessorConsumeLoopSnapshot {
	if r == nil {
		return eventProcessorConsumeLoopSnapshot{}
	}

	r.mu.Lock()
	wasPaused := r.paused
	r.paused = false
	r.reason = reason
	r.lastAction = "resume-intake"
	r.updatedAtUnix = time.Now().Unix()
	if wasPaused {
		close(r.waitCh)
		r.waitCh = make(chan struct{})
	}
	r.mu.Unlock()

	return r.Snapshot()
}

func (r *eventProcessorConsumeRuntime) Snapshot() eventProcessorConsumeLoopSnapshot {
	if r == nil {
		return eventProcessorConsumeLoopSnapshot{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	active := 0
	for _, running := range r.activeTopics {
		if running {
			active++
		}
	}

	state := "running"
	switch {
	case r.paused:
		state = "paused"
	case active == 0 && len(r.topics) > 0:
		state = "idle"
	}
	lastError, lastErrorAt := r.lastError, r.lastErrorAtUnix
	if lastError == "" {
		for _, topic := range r.topics {
			if transport, exists := r.transportErrors[topic]; exists {
				lastError, lastErrorAt = transport.message, transport.at
				break
			}
		}
	}
	_, ownershipRequired := r.processor.(generationOwnedProcessor)

	return eventProcessorConsumeLoopSnapshot{
		ConfiguredTopics:  len(r.topics),
		ActiveTopics:      active,
		OwnedTopics:       len(r.ownedTopics),
		OwnershipRequired: ownershipRequired,
		Running:           r.running,
		Paused:            r.paused,
		State:             state,
		LastError:         lastError,
		LastErrorAt:       lastErrorAt,
		LastAction:        r.lastAction,
		Reason:            r.reason,
		UpdatedUnix:       r.updatedAtUnix,
	}
}

func (r *eventProcessorConsumeRuntime) recordTransportError(topic string, err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.transportErrors == nil {
		r.transportErrors = make(map[string]consumeTransportError)
	}
	r.transportErrors[topic] = consumeTransportError{message: err.Error(), at: time.Now().Unix()}
}

func (r *eventProcessorConsumeRuntime) setOwnership(topic string, owned bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !owned {
		delete(r.ownedTopics, topic)
		return
	}
	if r.ownedTopics == nil {
		r.ownedTopics = make(map[string]bool)
	}
	r.ownedTopics[topic] = true
	delete(r.transportErrors, topic)
}

func (r *eventProcessorConsumeRuntime) setTopicControl(topic string, active bool, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if active {
		r.activeTopics[topic] = true
		if cancel != nil {
			r.activeCancels[topic] = cancel
		}
		return
	}
	delete(r.activeTopics, topic)
	delete(r.activeCancels, topic)
}

func (r *eventProcessorConsumeRuntime) recordError(err error) {
	if r == nil || err == nil {
		return
	}

	r.mu.Lock()
	r.lastError = err.Error()
	r.lastErrorAtUnix = time.Now().Unix()
	r.mu.Unlock()
}

func (r *eventProcessorConsumeRuntime) waitUntilIntakeAllowed(ctx context.Context) bool {
	for {
		r.mu.RLock()
		paused := r.paused
		waitCh := r.waitCh
		r.mu.RUnlock()
		if !paused {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-waitCh:
		}
	}
}

func decodeEventProcessorQueueMessage(message core.MessageQueueMessage) (*blockchain.BlockchainEvent, error) {
	if len(message.Payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	var event blockchain.BlockchainEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventProcessorConsumeRuntime) publishProcessedEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if r.publisher == nil || len(r.outputTopics) == 0 {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode processed event %s: %w", event.ID, err)
	}
	for _, topic := range r.outputTopics {
		if err := r.publisher.Publish(ctx, topic, payload); err != nil {
			return fmt.Errorf("publish processed event %s to %s: %w", event.ID, topic, err)
		}
	}
	return nil
}

func (r *eventProcessorConsumeRuntime) writeToDLQ(ctx context.Context, event *blockchain.BlockchainEvent, processErr error) {
	if r.dlqDB == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		r.logger.Warn("Failed to marshal event for DLQ", "eventId", event.ID, "error", err.Error())
		return
	}
	_, err = r.dlqDB.ExecContext(ctx,
		`INSERT INTO dlq_events (id, chain_id, original_event_id, error_message, retry_count, status, payload)
		 VALUES ($1, $2, $3, $4, 0, 'pending', $5)
		 ON CONFLICT (id) DO UPDATE SET retry_count = dlq_events.retry_count + 1, error_message = $4, status = 'pending', payload = $5, updated_at = NOW()`,
		event.ID, event.ChainID, event.ID, processErr.Error(), string(payload))
	if err != nil {
		r.logger.Warn("Failed to write event to DLQ", "eventId", event.ID, "error", err.Error())
	}
}

// consumeTopic preserves the legacy adapter contract while making broker-owned
// contexts mandatory for the PostgreSQL processor's durable effect path.
func (r *eventProcessorConsumeRuntime) consumeTopic(ctx context.Context, topic string, legacy func(core.MessageQueueMessage) error) error {
	owner, ok := r.processor.(generationOwnedProcessor)
	if !ok {
		return r.consumer.ConsumeMessages(ctx, topic, legacy)
	}
	consumer, ok := r.consumer.(interface {
		ConsumeOwnedMessages(context.Context, string, func(context.Context, core.MessageQueueMessage) error, mq.ConsumerOwnershipHooks) error
	})
	if !ok {
		return fmt.Errorf("durable processor requires generation-owned Kafka consumption")
	}
	blocks, ok := r.processor.(completeBlockMessageProcessor)
	if !ok {
		return fmt.Errorf("owned processor requires complete block effects")
	}
	return consumer.ConsumeOwnedMessages(ctx, topic, func(active context.Context, message core.MessageQueueMessage) error {
		err := r.processCompleteBlockMessage(active, message, blocks)
		if err != nil && active.Err() == nil {
			r.recordError(fmt.Errorf("process owned topic %s: %w", topic, err))
		}
		return err
	}, mq.ConsumerOwnershipHooks{
		Assigned: func(active context.Context, token string) error {
			if err := owner.AssignOwnership(active, token); err != nil {
				return err
			}
			r.setOwnership(topic, true)
			return nil
		},
		Revoked: func(cleanup context.Context) error {
			r.setOwnership(topic, false)
			return owner.RevokeOwnership(cleanup)
		},
	})
}
