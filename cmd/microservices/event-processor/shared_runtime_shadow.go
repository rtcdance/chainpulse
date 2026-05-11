package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"chainpulse/pkg/application/bootstrap"
	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	"chainpulse/pkg/services/processor"
)

type eventProcessorSharedRuntimeShadowSnapshot struct {
	Enabled              bool
	RuntimeCount         int
	Chains               []string
	ProcessedEvents      int64
	SkippedDuplicates    int64
	RoutedFailures       int64
	LastCheckpointChain  string
	LastCheckpointCursor string
	LastCheckpointBlock  uint64
	LastError            string
	LastErrorAtUnix      int64
}

//nolint:unused
type eventProcessorSharedRuntimeShadowProvider interface {
	SharedRuntimeShadowSnapshot() eventProcessorSharedRuntimeShadowSnapshot
}

type eventProcessorShadowRuntimeProcessor struct {
	base    *processor.DefaultEventProcessor
	logger  core.Logger
	metrics core.MetricsCollector

	mu              sync.RWMutex
	runtimes        map[string]*appindexing.SharedRuntime
	lastError       string
	lastErrorAtUnix int64
}

func newEventProcessorShadowRuntimeProcessor(
	base *processor.DefaultEventProcessor,
	logger core.Logger,
	metrics core.MetricsCollector,
) *eventProcessorShadowRuntimeProcessor {
	return &eventProcessorShadowRuntimeProcessor{
		base:     base,
		logger:   logger,
		metrics:  metrics,
		runtimes: make(map[string]*appindexing.SharedRuntime),
	}
}

func (p *eventProcessorShadowRuntimeProcessor) ProcessEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if p == nil || p.base == nil {
		return fmt.Errorf("event processor runtime is not configured")
	}
	if event != nil && event.EventHash == "" {
		event.EventHash = fmt.Sprintf(
			"%s:%d:%s:%d",
			eventProcessorShadowRuntimeChainID(event),
			event.BlockNumber,
			event.TransactionHash.Hex(),
			event.LogIndex,
		)
	}

	if err := p.base.ProcessEvent(ctx, event); err != nil {
		return err
	}
	if event == nil {
		return nil
	}

	chainID := eventProcessorShadowRuntimeChainID(event)
	runtime, err := p.runtimeForChain(chainID)
	if err != nil {
		p.recordShadowError(fmt.Errorf("build shared runtime for %s: %w", chainID, err))
		return nil
	}

	envelope := appindexing.EventEnvelope{
		EventKey:         event.ID,
		ChainID:          chainID,
		BlockNumber:      event.BlockNumber,
		TransactionHash:  event.TransactionHash.Hex(),
		LogIndex:         event.LogIndex,
		Payload:          event,
		ReceivedAt:       eventProcessorShadowReceivedAt(event),
		CheckpointCursor: fmt.Sprintf("%s:%d:%d", chainID, event.BlockNumber, event.LogIndex),
	}

	if err := runtime.ProcessBatch(context.Background(), chainID, []appindexing.EventEnvelope{envelope}); err != nil {
		p.recordShadowError(fmt.Errorf("process shared runtime shadow for %s event %s: %w", chainID, event.ID, err))
		if p.metrics != nil {
			p.metrics.RecordCounter("event_processor_shared_runtime_shadow_errors_total", 1, map[string]string{
				"chain_id": chainID,
			})
		}
		if p.logger != nil {
			p.logger.Warn("event processor shared runtime shadow failed", "chain_id", chainID, "event_id", event.ID, "error", err.Error())
		}
		return nil
	}

	if p.metrics != nil {
		p.metrics.RecordCounter("event_processor_shared_runtime_shadow_events_total", 1, map[string]string{
			"chain_id": chainID,
		})
	}
	return nil
}

func (p *eventProcessorShadowRuntimeProcessor) Health() *core.HealthStatus {
	if p == nil || p.base == nil {
		return &core.HealthStatus{Status: "unhealthy", Message: "shared-runtime shadow processor not configured"}
	}
	return p.base.Health()
}

func (p *eventProcessorShadowRuntimeProcessor) GetProcessedCount() int64 {
	if p == nil || p.base == nil {
		return 0
	}
	return p.base.GetProcessedCount()
}

func (p *eventProcessorShadowRuntimeProcessor) GetFailedCount() int64 {
	if p == nil || p.base == nil {
		return 0
	}
	return p.base.GetFailedCount()
}

func (p *eventProcessorShadowRuntimeProcessor) GetDuplicateCount() int64 {
	if p == nil || p.base == nil {
		return 0
	}
	return p.base.GetDuplicateCount()
}

func (p *eventProcessorShadowRuntimeProcessor) SharedRuntimeShadowSnapshot() eventProcessorSharedRuntimeShadowSnapshot {
	if p == nil {
		return eventProcessorSharedRuntimeShadowSnapshot{}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	snapshot := eventProcessorSharedRuntimeShadowSnapshot{
		Enabled:         true,
		RuntimeCount:    len(p.runtimes),
		LastError:       p.lastError,
		LastErrorAtUnix: p.lastErrorAtUnix,
	}
	var lastCheckpointUpdated int64

	for chainID, runtime := range p.runtimes {
		snapshot.Chains = append(snapshot.Chains, chainID)
		status := runtime.Status()
		snapshot.ProcessedEvents += status.ProcessedEvents
		snapshot.SkippedDuplicates += status.SkippedDuplicates
		snapshot.RoutedFailures += status.RoutedFailures
		if status.LastUpdatedAt.IsZero() {
			continue
		}
		if snapshot.LastCheckpointChain == "" || status.LastUpdatedAt.Unix() >= lastCheckpointUpdated {
			if status.LastCheckpointChainID != "" {
				snapshot.LastCheckpointChain = status.LastCheckpointChainID
				snapshot.LastCheckpointCursor = status.LastCheckpointCursor
				snapshot.LastCheckpointBlock = status.LastCheckpointBlock
				lastCheckpointUpdated = status.LastUpdatedAt.Unix()
			}
		}
	}
	sort.Strings(snapshot.Chains)
	return snapshot
}

func (p *eventProcessorShadowRuntimeProcessor) Stop() error {
	if p == nil || p.base == nil {
		return nil
	}
	return p.base.Stop()
}

func (p *eventProcessorShadowRuntimeProcessor) runtimeForChain(chainID string) (*appindexing.SharedRuntime, error) {
	p.mu.RLock()
	existing := p.runtimes[chainID]
	p.mu.RUnlock()
	if existing != nil {
		return existing, nil
	}

	runtime, err := bootstrap.BuildInMemoryIndexingRuntime(
		p.logger,
		eventProcessorSharedRuntimeNoopSink{},
		[]string{chainID},
	)
	if err != nil {
		return nil, err
	}
	if err := runtime.Initialize(context.Background()); err != nil {
		return nil, err
	}
	if err := runtime.Start(context.Background()); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if current := p.runtimes[chainID]; current != nil {
		return current, nil
	}
	p.runtimes[chainID] = runtime
	return runtime, nil
}

func (p *eventProcessorShadowRuntimeProcessor) recordShadowError(err error) {
	if p == nil || err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err.Error()
	p.lastErrorAtUnix = time.Now().Unix()
}

func eventProcessorShadowRuntimeChainID(event *core.BlockchainEvent) string {
	if event == nil {
		return "unknown"
	}
	if event.ChainID != "" {
		return event.ChainID
	}
	if event.Network != "" {
		return event.Network
	}
	return "unknown"
}

func eventProcessorShadowReceivedAt(event *core.BlockchainEvent) time.Time {
	if event == nil {
		return time.Now()
	}
	if !event.CreatedAt.IsZero() {
		return event.CreatedAt
	}
	if !event.IndexedAt.IsZero() {
		return event.IndexedAt
	}
	return time.Now()
}

type eventProcessorSharedRuntimeNoopSink struct{}

func (eventProcessorSharedRuntimeNoopSink) Persist(ctx context.Context, events []appindexing.EventEnvelope) error {
	return nil
}
