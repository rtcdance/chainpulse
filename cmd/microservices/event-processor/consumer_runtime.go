package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

type eventProcessorMessageConsumer interface {
	ConsumeMessages(context.Context, string, func(core.MessageQueueMessage) error) error
}

type eventProcessorMessageProcessor interface {
	ProcessEvent(*core.BlockchainEvent) error
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
	logger    core.Logger
	metrics   core.MetricsCollector
	consumer  eventProcessorMessageConsumer
	processor eventProcessorMessageProcessor
	topics    []string

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
}

func newEventProcessorConsumeRuntime(
	logger core.Logger,
	metrics core.MetricsCollector,
	consumer eventProcessorMessageConsumer,
	processor eventProcessorMessageProcessor,
	topics []string,
) *eventProcessorConsumeRuntime {
	return &eventProcessorConsumeRuntime{
		logger:        logger,
		metrics:       metrics,
		consumer:      consumer,
		processor:     processor,
		topics:        append([]string(nil), topics...),
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
			for {
				if !r.waitUntilIntakeAllowed(ctx) {
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
					if err := r.processor.ProcessEvent(event); err != nil {
						r.recordError(fmt.Errorf("process topic %s event %s: %w", topic, event.ID, err))
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
					r.recordError(fmt.Errorf("consume topic %s: %w", topic, err))
				}
			}
		}()
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
	close(r.waitCh)
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

func decodeEventProcessorQueueMessage(message core.MessageQueueMessage) (*core.BlockchainEvent, error) {
	if len(message.Payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	var event core.BlockchainEvent
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
