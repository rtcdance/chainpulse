package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
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

type eventProcessorConsumeLoopSnapshot struct {
	ConfiguredTopics int
	ActiveTopics     int
	Running          bool
	Paused           bool
	State            string
	LastError        string
	LastErrorAt      int64
	LastAction       string
	Reason           string
	UpdatedUnix      int64
}

type eventProcessorConsumeRuntime struct {
	logger       core.Logger
	metrics      core.MetricsCollector
	consumer     eventProcessorMessageConsumer
	processor    eventProcessorMessageProcessor
	publisher    eventProcessorMessagePublisher
	topics       []string
	outputTopics []string
	dlqDB        *sql.DB // PostgreSQL for DLQ persistence
	batchSize    int
	batchFlushMs int

	mu              sync.RWMutex
	running         bool
	paused          bool
	activeTopics    map[string]bool
	activeCancels   map[string]context.CancelFunc
	lastError       string
	lastErrorAtUnix int64
	lastAction      string
	reason          string
	updatedAtUnix   int64
	waitCh          chan struct{}
	closeOnce       sync.Once
}

type eventProcessorMessagePublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

func newEventProcessorConsumeRuntime(
	logger core.Logger,
	metrics core.MetricsCollector,
	consumer eventProcessorMessageConsumer,
	processor eventProcessorMessageProcessor,
	publisher eventProcessorMessagePublisher,
	topics []string,
	outputTopics []string,
	dlqDB *sql.DB,
) *eventProcessorConsumeRuntime {
	return &eventProcessorConsumeRuntime{
		logger:        logger,
		metrics:       metrics,
		consumer:      consumer,
		processor:     processor,
		publisher:     publisher,
		topics:        append([]string(nil), topics...),
		outputTopics:  append([]string(nil), outputTopics...),
		dlqDB:         dlqDB,
		batchSize:     50,
		batchFlushMs:  500,
		activeTopics:  make(map[string]bool),
		activeCancels: make(map[string]context.CancelFunc),
		waitCh:        make(chan struct{}),
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

			// Batch accumulator: events are collected and flushed periodically.
			eventCh := make(chan *blockchain.BlockchainEvent, r.batchSize*2)

			// Start batch flusher goroutine
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.batchFlushLoop(ctx, topic, eventCh)
			}()

			for {
				if !r.waitUntilIntakeAllowed(ctx) {
					// Drain remaining events before exiting
					close(eventCh)
					return
				}

				consumeCtx, cancel := context.WithCancel(ctx)
				r.setTopicControl(topic, true, cancel)

				err := r.consumer.ConsumeMessages(consumeCtx, topic, func(message core.MessageQueueMessage) error {
					event, err := decodeEventProcessorQueueMessage(message)
					if err != nil {
						r.recordError(fmt.Errorf("decode topic %s message %s: %w", topic, message.ID, err))
						return err
					}
					select {
					case eventCh <- event:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})

				r.setTopicControl(topic, false, nil)
				cancel()

				if ctx.Err() != nil {
					close(eventCh)
					return
				}
				if errors.Is(err, context.Canceled) {
					continue
				}
				if err != nil {
					r.recordError(fmt.Errorf("consume topic %s: %w", topic, err))
				}
			}
		}()
	}
}

// batchFlushLoop reads events from eventCh and processes them in batches.
// This amortizes MongoDB write costs by using InsertMany instead of InsertOne.
func (r *eventProcessorConsumeRuntime) batchFlushLoop(ctx context.Context, topic string, eventCh <-chan *blockchain.BlockchainEvent) {
	batch := make([]*blockchain.BlockchainEvent, 0, r.batchSize)
	flushTimer := time.NewTimer(time.Duration(r.batchFlushMs) * time.Millisecond)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		r.flushBatch(ctx, topic, batch)
		batch = batch[:0]
	}

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed, flush remaining
				flush()
				return
			}
			batch = append(batch, event)
			if len(batch) >= r.batchSize {
				flush()
				if !flushTimer.Stop() {
					select {
					case <-flushTimer.C:
					default:
					}
				}
				flushTimer.Reset(time.Duration(r.batchFlushMs) * time.Millisecond)
			}
		case <-flushTimer.C:
			flush()
			flushTimer.Reset(time.Duration(r.batchFlushMs) * time.Millisecond)
		case <-ctx.Done():
			// Drain remaining events from channel
			for len(eventCh) > 0 {
				if event := <-eventCh; event != nil {
					batch = append(batch, event)
				}
			}
			flush()
			return
		}
	}
}

// flushBatch processes a batch of events. It first filters duplicates via
// idempotency, then writes non-duplicate events in a single MongoDB batch,
// and finally publishes each processed event to output topics.
func (r *eventProcessorConsumeRuntime) flushBatch(ctx context.Context, topic string, batch []*blockchain.BlockchainEvent) {
	r.logger.Info("flushBatch called", "topic", topic, "batch_size", len(batch))
	for i, event := range batch {
		r.logger.Info("flushBatch processing event", "index", i, "event_id", event.ID, "network", event.Network, "event_name", event.EventName)
		if err := r.processor.ProcessEvent(ctx, event); err != nil {
			r.recordError(fmt.Errorf("process topic %s event %s: %w", topic, event.ID, err))
			r.writeToDLQ(ctx, event, err)
			continue
		}
		r.logger.Info("flushBatch event processed, publishing", "event_id", event.ID)
		r.publishProcessedEvent(ctx, event)
		if r.metrics != nil {
			r.metrics.RecordCounter("event_processor_consume_processed", 1, map[string]string{"topic": topic})
		}
	}
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
	r.paused = false
	r.reason = reason
	r.lastAction = "resume-intake"
	r.updatedAtUnix = time.Now().Unix()
	r.closeOnce.Do(func() {
		close(r.waitCh)
	})
	r.waitCh = make(chan struct{})
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

	return eventProcessorConsumeLoopSnapshot{
		ConfiguredTopics: len(r.topics),
		ActiveTopics:     active,
		Running:          r.running,
		Paused:           r.paused,
		State:            state,
		LastError:        r.lastError,
		LastErrorAt:      r.lastErrorAtUnix,
		LastAction:       r.lastAction,
		Reason:           r.reason,
		UpdatedUnix:      r.updatedAtUnix,
	}
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

func (r *eventProcessorConsumeRuntime) publishProcessedEvent(ctx context.Context, event *blockchain.BlockchainEvent) {
	if r.publisher == nil || len(r.outputTopics) == 0 {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		r.logger.Warn("Failed to marshal event for output topic", "eventId", event.ID, "error", err.Error())
		return
	}
	for _, topic := range r.outputTopics {
		if err := r.publisher.Publish(ctx, topic, payload); err != nil {
			r.logger.Warn("Failed to publish event to output topic", "topic", topic, "eventId", event.ID, "error", err.Error())
		}
	}
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
