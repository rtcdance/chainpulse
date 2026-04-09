package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/application/bootstrap"
	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/pullers"
)

type pullerMessagePublisher interface {
	Publish(context.Context, string, []byte) error
}

type pullerExecutionRuntimeSnapshot struct {
	Enabled              bool
	ConfiguredPullers    int
	AttachedPullers      int
	PublishedEvents      int64
	PublishedMessages    int64
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

type pullerExecutionRuntimeStatusProvider interface {
	ExecutionSnapshot() pullerExecutionRuntimeSnapshot
}

type pullerExecutionRuntime struct {
	logger       core.Logger
	metrics      core.MetricsCollector
	publisher    pullerMessagePublisher
	outputTopics []string

	mu                sync.RWMutex
	configuredPullers int
	runtimes          map[string]*appindexing.SharedRuntime
	publishedEvents   int64
	publishedMessages int64
	lastError         string
	lastErrorAtUnix   int64
}

func newPullerExecutionRuntime(
	logger core.Logger,
	metrics core.MetricsCollector,
	publisher pullerMessagePublisher,
	outputTopics []string,
) *pullerExecutionRuntime {
	return &pullerExecutionRuntime{
		logger:       logger,
		metrics:      metrics,
		publisher:    publisher,
		outputTopics: append([]string(nil), outputTopics...),
		runtimes:     make(map[string]*appindexing.SharedRuntime),
	}
}

func (r *pullerExecutionRuntime) SetConfiguredPullers(count int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configuredPullers = count
}

func (r *pullerExecutionRuntime) ExecutionSnapshot() pullerExecutionRuntimeSnapshot {
	if r == nil {
		return pullerExecutionRuntimeSnapshot{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := pullerExecutionRuntimeSnapshot{
		Enabled:           true,
		ConfiguredPullers: r.configuredPullers,
		AttachedPullers:   len(r.runtimes),
		PublishedEvents:   r.publishedEvents,
		PublishedMessages: r.publishedMessages,
		RuntimeCount:      len(r.runtimes),
		LastError:         r.lastError,
		LastErrorAtUnix:   r.lastErrorAtUnix,
	}

	var lastCheckpointUpdated int64
	for chainID, runtime := range r.runtimes {
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

func (r *pullerExecutionRuntime) Poll(ctx context.Context, puller *pullers.MultiChainDataPuller, config PullerConfig) error {
	if r == nil || puller == nil {
		return nil
	}

	var firstErr error
	processedBlocks := puller.GetProcessedBlocksFromAllChains()

	for _, chainID := range puller.RegisteredChains() {
		err := r.pollChain(ctx, puller, config, chainID, processedBlocks)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (r *pullerExecutionRuntime) pollChain(
	ctx context.Context,
	puller *pullers.MultiChainDataPuller,
	config PullerConfig,
	chainID string,
	processedBlocks map[string]uint64,
) error {
	latestBlock, err := puller.GetLatestBlockFromChain(ctx, chainID)
	if err != nil {
		return fmt.Errorf("get latest block for %s: %w", chainID, err)
	}

	targetBlock := latestBlock
	if confirmation, ok := safePositiveIntToUint64(config.BlockConfirmation); ok && confirmation > 0 {
		if latestBlock <= confirmation {
			return nil
		}
		targetBlock = latestBlock - confirmation
	}

	fromBlock := processedBlocks[chainID] + 1
	if targetBlock == 0 || fromBlock > targetBlock {
		return nil
	}

	events, err := puller.PullEventsFromChain(ctx, chainID, fromBlock, targetBlock)
	if err != nil {
		return fmt.Errorf("pull events for %s [%d,%d]: %w", chainID, fromBlock, targetBlock, err)
	}

	if err := r.publishEvents(ctx, config.InstanceID, events); err != nil {
		return fmt.Errorf("publish events for %s: %w", chainID, err)
	}

	if err := r.shadowEvents(ctx, chainID, events); err != nil {
		return fmt.Errorf("shadow events for %s: %w", chainID, err)
	}

	return puller.SetLastProcessedBlock(chainID, targetBlock)
}

func safePositiveIntToUint64(value int) (uint64, bool) {
	if value < 0 {
		return 0, false
	}

	if value > math.MaxInt {
		return 0, false
	}

	return uint64(value), true
}

func (r *pullerExecutionRuntime) publishEvents(ctx context.Context, instanceID string, events []core.BlockchainEvent) error {
	if r == nil || r.publisher == nil || len(r.outputTopics) == 0 || len(events) == 0 {
		return nil
	}

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		for _, topic := range r.outputTopics {
			if err := r.publisher.Publish(ctx, topic, payload); err != nil {
				return err
			}
			if r.metrics != nil {
				r.metrics.RecordCounter("puller_published_messages_total", 1, map[string]string{"topic": topic})
			}
			r.mu.Lock()
			r.publishedMessages++
			r.mu.Unlock()
		}
		if r.metrics != nil {
			r.metrics.RecordCounter("puller_published_events_total", 1, map[string]string{"instance": instanceID})
		}
		r.mu.Lock()
		r.publishedEvents++
		r.mu.Unlock()
	}

	return nil
}

func (r *pullerExecutionRuntime) shadowEvents(ctx context.Context, chainID string, events []core.BlockchainEvent) error {
	if r == nil || len(events) == 0 {
		return nil
	}

	runtime, err := r.runtimeForChain(chainID)
	if err != nil {
		return err
	}

	envelopes := make([]appindexing.EventEnvelope, 0, len(events))
	for _, event := range events {
		eventCopy := event
		if eventCopy.EventHash == "" {
			eventCopy.EventHash = fmt.Sprintf("%s:%d:%s:%d", chainID, eventCopy.BlockNumber, eventCopy.TransactionHash.Hex(), eventCopy.LogIndex)
		}
		eventKey := eventCopy.ID
		if eventKey == "" {
			eventKey = eventCopy.EventHash
		}
		envelopes = append(envelopes, appindexing.EventEnvelope{
			EventKey:         eventKey,
			ChainID:          chainID,
			BlockNumber:      eventCopy.BlockNumber,
			TransactionHash:  eventCopy.TransactionHash.Hex(),
			LogIndex:         eventCopy.LogIndex,
			Payload:          &eventCopy,
			ReceivedAt:       pullerShadowReceivedAt(eventCopy),
			CheckpointCursor: fmt.Sprintf("%s:%d:%d", chainID, eventCopy.BlockNumber, eventCopy.LogIndex),
		})
	}

	return runtime.ProcessBatch(ctx, chainID, envelopes)
}

func (r *pullerExecutionRuntime) runtimeForChain(chainID string) (*appindexing.SharedRuntime, error) {
	r.mu.RLock()
	existing := r.runtimes[chainID]
	r.mu.RUnlock()
	if existing != nil {
		return existing, nil
	}

	runtime, err := bootstrap.BuildInMemoryIndexingRuntime(r.logger, pullerSharedRuntimeNoopSink{}, []string{chainID})
	if err != nil {
		return nil, err
	}
	if err := runtime.Initialize(context.Background()); err != nil {
		return nil, err
	}
	if err := runtime.Start(context.Background()); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if current := r.runtimes[chainID]; current != nil {
		return current, nil
	}
	r.runtimes[chainID] = runtime
	return runtime, nil
}

func (r *pullerExecutionRuntime) recordError(err error) {
	if r == nil || err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = err.Error()
	r.lastErrorAtUnix = time.Now().Unix()
}

func pullerShadowReceivedAt(event core.BlockchainEvent) time.Time {
	if !event.CreatedAt.IsZero() {
		return event.CreatedAt
	}
	if !event.IndexedAt.IsZero() {
		return event.IndexedAt
	}
	return time.Now()
}

type pullerSharedRuntimeNoopSink struct{}

func (pullerSharedRuntimeNoopSink) Persist(ctx context.Context, events []appindexing.EventEnvelope) error {
	return nil
}

func registerConfiguredPullers(
	multi *pullers.MultiChainDataPuller,
	config PullerConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
) (int, error) {
	if multi == nil {
		return 0, fmt.Errorf("multi-chain puller is required")
	}

	registered := 0
	for idx, entry := range config.BlockchainRPCs {
		chainID, rpcURL, err := parsePullerRPCEntry(entry, idx)
		if err != nil {
			return registered, err
		}

		puller := pullers.NewHTTPSJSONRPCPuller(core.Config{
			DataPullerType:    "https-jsonrpc",
			BlockchainNodeURL: rpcURL,
			StartBlock:        0,
			BatchSize:         config.BatchSize,
			MaxRetries:        config.MaxRetries,
			RetryBackoff:      1000,
			LogLevel:          config.LogLevel,
		}, logger, metrics, nil)

		if err := multi.RegisterPuller(chainID, puller); err != nil {
			return registered, err
		}
		registered++
	}

	return registered, nil
}

func parsePullerRPCEntry(entry string, index int) (string, string, error) {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" {
		return "", "", fmt.Errorf("blockchain RPC entry %d is empty", index)
	}

	if key, value, ok := parsePullerRPCKeyValue(trimmed); ok {
		if _, err := url.ParseRequestURI(value); err != nil {
			return "", "", fmt.Errorf("invalid blockchain RPC URL %q: %w", value, err)
		}
		return sanitizePullerChainID(key, index), value, nil
	}

	if _, err := url.ParseRequestURI(trimmed); err != nil {
		return "", "", fmt.Errorf("invalid blockchain RPC URL %q: %w", trimmed, err)
	}
	return inferPullerChainID(trimmed, index), trimmed, nil
}

func parsePullerRPCKeyValue(entry string) (string, string, bool) {
	idx := strings.Index(entry, "=")
	if idx <= 0 || idx >= len(entry)-1 {
		return "", "", false
	}
	return strings.TrimSpace(entry[:idx]), strings.TrimSpace(entry[idx+1:]), true
}

func inferPullerChainID(rawURL string, index int) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Sprintf("chain-%d", index+1)
	}

	host := strings.ToLower(parsed.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimSuffix(host, "-rpc")
	host = strings.TrimSuffix(host, "_rpc")
	host = strings.TrimSuffix(host, "-node")
	if host == "localhost" || host == "127.0.0.1" {
		return fmt.Sprintf("chain-%d", index+1)
	}

	parts := strings.FieldsFunc(host, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return fmt.Sprintf("chain-%d", index+1)
	}
	return sanitizePullerChainID(parts[0], index)
}

func sanitizePullerChainID(chainID string, index int) string {
	normalized := strings.ToLower(strings.TrimSpace(chainID))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return fmt.Sprintf("chain-%d", index+1)
	}
	return normalized
}
