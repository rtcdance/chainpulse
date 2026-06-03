package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	pluginapi "github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
	"github.com/rtcdance/chainpulse/pkg/services/indexing"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"

	"github.com/ethereum/go-ethereum/common"
)

const monolithicEventTopic = topics.TopicBlockchainEvents

type monolithicPullerRuntime struct {
	logger      core.Logger
	eventBus    *core.DefaultEventBus
	pullers     []monolithicPollingPuller
	database    core.DatabasePlugin
	reorgChains map[string]*monolithicChainReorgRuntime
	reorgMu     sync.RWMutex
	startedAt   time.Time
	stateMu     sync.RWMutex
	lastError   string
	loopChains  map[string]*monolithicPullLoopRuntime
	backoffBase time.Duration
	backoffMax  time.Duration
}

type monolithicBlockSnapshotStore interface {
	StoreBlockSnapshot(ctx context.Context, block *core.Block) error
}

type monolithicPollingPuller interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Poll(ctx context.Context) error
	GetConfig() core.Config
	GetStats() map[string]any
}

type monolithicChainReorgRuntime struct {
	mu                sync.RWMutex
	handler           *reorg.ReorgHandler
	chainID           string
	detectedTotal     int64
	handledTotal      int64
	lastDetectedBlock uint64
	lastHandledBlock  uint64
	lastError         string
}

type monolithicPullLoopRuntime struct {
	mu              sync.RWMutex
	chainID         string
	restartTotal    int64
	failureTotal    int64
	lastError       string
	lastErrorAtUnix int64
	lastBackoffMS   int64
	lastRestartUnix int64
	state           string
}

func newMonolithicPullerRuntime(
	ctx context.Context,
	baseConfig core.Config,
	rawNodeURLs string,
	chains []string,
	logger core.Logger,
	metrics core.MetricsCollector,
	database core.DatabasePlugin,
	multiChainIndexer *indexing.MultiChainIndexer,
) (*monolithicPullerRuntime, error) {
	//nolint:funlen // Runtime initializer has many setup steps.
	nodeURLs, err := parseNodeURLs(rawNodeURLs)
	if err != nil {
		return nil, err
	}

	if len(nodeURLs) != len(chains) {
		return nil, fmt.Errorf("configured chains (%d) and blockchain node URLs (%d) must align", len(chains), len(nodeURLs))
	}
	if database == nil {
		return nil, fmt.Errorf("database is required")
	}

	eventBus := core.NewEventBus(logger)
	if err := subscribeMonolithicIndexer(ctx, eventBus, multiChainIndexer, logger); err != nil {
		return nil, err
	}

	runtime := &monolithicPullerRuntime{
		logger:      logger,
		eventBus:    eventBus,
		pullers:     make([]monolithicPollingPuller, 0, len(chains)),
		database:    database,
		reorgChains: make(map[string]*monolithicChainReorgRuntime, len(chains)),
		loopChains:  make(map[string]*monolithicPullLoopRuntime, len(chains)),
		backoffBase: 250 * time.Millisecond,
		backoffMax:  5 * time.Second,
	}

	for idx, chainID := range chains {
		pullerConfig := baseConfig
		pullerConfig.ChainID = chainID
		pullerConfig.ServiceName = chainID
		pullerConfig.BlockchainNodeURL = nodeURLs[idx]
		logger.Debug("creating puller", "chain", chainID, "nodeURL", nodeURLs[idx])
		var puller monolithicPollingPuller
		if chainid.IsCosmosChain(chainID) {
			puller = pullers.NewCosmosPuller(pullerConfig, logger, metrics)
		} else if chainid.IsSolanaChain(chainID) {
			puller = pullers.NewSolanaPuller(pullerConfig, logger, metrics, eventBus)
		} else {
			puller = pullers.NewHTTPSJSONRPCPuller(pullerConfig, logger, metrics, eventBus)
		}
		if dataPuller, ok := any(puller).(core.DataPullerPlugin); ok {
			if err := dataPuller.SubscribeToEvents(ctx, runtime.observeEvent); err != nil {
				return nil, err
			}
		}
		runtime.pullers = append(runtime.pullers, puller)
		reorgThreshold := reorgThresholdForChain(chainID)
		runtime.reorgChains[chainID] = &monolithicChainReorgRuntime{
			handler: reorg.NewReorgHandler(
				database,
				logger,
				reorgThreshold,
				reorgThreshold*10,
			).WithChainID(chainID).WithEventBus(runtime.eventBus),
			chainID: chainID,
		}
		// Inject RPC-backed block hash provider so reorg detection
		// compares local hashes against the live canonical chain.
		// Only supports EVM pullers that expose block hash RPC methods.
		if evmPuller, ok := puller.(*pullers.HTTPSJSONRPCPuller); ok {
			rpcProvider := pullers.NewRPCBlockHashProvider(evmPuller)
			runtime.reorgChains[chainID].handler.SetBlockHashProvider(rpcProvider)
		}
		// Note: SetIdempotencyInvalidator should be called here if an
		// IdempotencyService is available in the runtime, so that re-indexed
		// events after a reorg are not rejected as duplicates.
		runtime.loopChains[chainID] = &monolithicPullLoopRuntime{
			chainID: chainID,
			state:   "primed",
		}
	}

	return runtime, nil
}

func (m *monolithicPullerRuntime) Start(ctx context.Context, wg *sync.WaitGroup) error {
	m.stateMu.Lock()
	m.startedAt = time.Now()
	m.lastError = ""
	m.stateMu.Unlock()

	for _, puller := range m.pullers {
		if err := puller.Start(ctx); err != nil {
			_ = m.Stop()

			return fmt.Errorf("failed to start puller: %w", err)
		}

		wg.Add(1)

		runLoop := func() {
			m.runPullerLoop(ctx, wg, puller)
		}

		go runLoop()
	}

	return nil
}

func (m *monolithicPullerRuntime) runPullerLoop(ctx context.Context, wg *sync.WaitGroup, puller monolithicPollingPuller) {
	defer wg.Done()
	chainID := puller.GetConfig().ServiceName
	attempt := 0

	for {
		m.recordLoopState(chainID, "running")
		err := puller.Poll(ctx)
		if err == nil {
			continue
		}
		if err == context.Canceled || ctx.Err() != nil {
			m.recordLoopState(chainID, "stopped")
			return
		}

		backoff := m.nextLoopBackoff(attempt)
		attempt++
		m.recordLoopFailure(chainID, err, backoff)
		m.recordLoopError(err)
		m.logger.Error("monolithic puller exited", "chain_id", chainID, "error", err.Error(), "restart_backoff", backoff.String())

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			m.recordLoopState(chainID, "stopped")
			return
		case <-timer.C:
			m.recordLoopRestart(chainID)
		}
	}
}

func (m *monolithicPullerRuntime) Stop() error {
	var errs []string

	for _, puller := range m.pullers {
		if err := puller.Stop(context.Background()); err != nil && !strings.Contains(err.Error(), "plugin not running") {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to stop monolithic pullers: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (m *monolithicPullerRuntime) PullerCount() int {
	return len(m.pullers)
}

func (m *monolithicPullerRuntime) SubscriberCount() int {
	return m.eventBus.GetSubscriberCount(monolithicEventTopic)
}

// EventBus returns the shared event bus for wiring push-based subscriptions.
func (m *monolithicPullerRuntime) EventBus() core.EventBus {
	return m.eventBus
}

func (m *monolithicPullerRuntime) recordLoopError(err error) {
	if err == nil {
		return
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.lastError = err.Error()
}

func (m *monolithicPullerRuntime) nextLoopBackoff(attempt int) time.Duration {
	backoff := m.backoffBase
	for i := 0; i < attempt; i++ {
		backoff *= 2
		if backoff >= m.backoffMax {
			return m.backoffMax
		}
	}
	if backoff > m.backoffMax {
		return m.backoffMax
	}
	return backoff
}

func (m *monolithicPullerRuntime) recordLoopState(chainID, state string) {
	loop := m.loopChains[chainID]
	if loop == nil {
		return
	}
	loop.mu.Lock()
	defer loop.mu.Unlock()
	loop.state = state
}

func (m *monolithicPullerRuntime) recordLoopFailure(chainID string, err error, backoff time.Duration) {
	loop := m.loopChains[chainID]
	if loop == nil {
		return
	}
	loop.mu.Lock()
	defer loop.mu.Unlock()
	loop.failureTotal++
	loop.lastBackoffMS = backoff.Milliseconds()
	loop.lastErrorAtUnix = time.Now().Unix()
	loop.state = "backing-off"
	if err != nil {
		loop.lastError = err.Error()
	}
}

func (m *monolithicPullerRuntime) recordLoopRestart(chainID string) {
	loop := m.loopChains[chainID]
	if loop == nil {
		return
	}
	loop.mu.Lock()
	defer loop.mu.Unlock()
	loop.restartTotal++
	loop.lastRestartUnix = time.Now().Unix()
	loop.state = "restarting"
}

func (m *monolithicPullerRuntime) observeEvent(event core.BlockchainEvent) {
	m.reorgMu.RLock()
	chainRuntime, ok := m.reorgChains[event.ChainID]
	m.reorgMu.RUnlock()
	if !ok || chainRuntime == nil || chainRuntime.handler == nil {
		return
	}

	if event.BlockNumber == 0 || event.BlockHash == (common.Hash{}) {
		return
	}

	ctx := context.Background()
	reorgDetected, reorgBlock, err := chainRuntime.handler.DetectReorg(ctx, event.BlockNumber, event.BlockHash)
	if err != nil {
		chainRuntime.recordError(err)
		m.logger.Warn("monolithic reorg detection failed", "chain_id", event.ChainID, "block_number", event.BlockNumber, "error", err.Error())
		return
	}

	if reorgDetected {
		chainRuntime.recordDetection(reorgBlock)
		if err := chainRuntime.handler.HandleReorg(ctx, reorgBlock); err != nil {
			chainRuntime.recordError(err)
			m.logger.Warn("monolithic reorg rollback failed", "chain_id", event.ChainID, "reorg_block", reorgBlock, "error", err.Error())
			return
		}
		chainRuntime.recordHandled(reorgBlock)
	}

	if err := m.storeBlockSnapshot(ctx, &event); err != nil {
		chainRuntime.recordError(err)
		m.logger.Warn("monolithic block snapshot persistence failed", "chain_id", event.ChainID, "block_number", event.BlockNumber, "error", err.Error())
		return
	}

	chainRuntime.handler.UpdateBlockHash(event.BlockNumber, event.BlockHash)
	chainRuntime.clearError()
}

func (m *monolithicPullerRuntime) storeBlockSnapshot(ctx context.Context, event *core.BlockchainEvent) error {
	store, ok := m.database.(monolithicBlockSnapshotStore)
	if !ok {
		return fmt.Errorf("monolithic database does not support block snapshots")
	}

	return store.StoreBlockSnapshot(ctx, &core.Block{
		Number: event.BlockNumber,
		Hash:   event.BlockHash,
	})
}

func (m *monolithicPullerRuntime) ReorgStatus() monolithicReorgSummary {
	summary := monolithicReorgSummary{
		Wired:      len(m.reorgChains) > 0,
		ChainCount: len(m.reorgChains),
		Posture:    "monolithic-reorg-unwired",
		Hint:       "monolithic reorg rollback is not wired",
	}
	if len(m.reorgChains) == 0 {
		return summary
	}

	summary.Posture = "monolithic-reorg-armed"
	summary.Hint = "monolithic reorg rollback is wired with in-memory block snapshots"

	for chainID, runtime := range m.reorgChains {
		snapshot := runtime.snapshot()
		summary.DetectedTotal += snapshot.DetectedTotal
		summary.HandledTotal += snapshot.HandledTotal
		if snapshot.LastDetectedBlock > 0 {
			summary.LastDetectedChainID = chainID
			summary.LastDetectedBlock = snapshot.LastDetectedBlock
		}
		if snapshot.LastHandledBlock > 0 {
			summary.LastHandledChainID = chainID
			summary.LastHandledBlock = snapshot.LastHandledBlock
		}
		if snapshot.LastError != "" {
			summary.LastError = snapshot.LastError
			summary.Posture = "monolithic-reorg-degraded"
			summary.Hint = "monolithic reorg rollback is wired but the latest detection or rollback attempt failed"
		}
	}

	if summary.HandledTotal > 0 {
		summary.Posture = "monolithic-reorg-active"
		summary.Hint = "monolithic reorg rollback is wired and has already handled at least one in-memory rollback"
	}
	if summary.LastError != "" {
		summary.Posture = "monolithic-reorg-degraded"
		summary.Hint = "monolithic reorg rollback is wired but the latest detection or rollback attempt failed"
	}

	return summary
}

func (m *monolithicPullerRuntime) PullerStatus() monolithicPullerSummary {
	//nolint:funlen // Status builder has many field assignments.
	summary := monolithicPullerSummary{
		PullerCount:    len(m.pullers),
		ControlTarget:  pluginapi.RuntimeControlTargetPollingLoop,
		ControlPosture: "monolithic-puller-read-only-control",
		ControlHint:    "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
		Posture:        "monolithic-puller-uninitialized",
		Hint:           "monolithic puller runtime is not yet initialized",
	}

	if len(m.pullers) == 0 {
		return summary
	}

	m.stateMu.RLock()
	startedAt := m.startedAt
	lastError := m.lastError
	m.stateMu.RUnlock()

	summary.Control = pluginapi.RuntimeControlCore{
		Paused: false,
		State:  "running",
		Reason: "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
	}

	if startedAt.IsZero() {
		summary.Control.State = "idle"
		summary.Posture = "monolithic-puller-primed"
		summary.Hint = "monolithic puller runtime is wired but has not started polling yet"
		return summary
	}

	for _, puller := range m.pullers {
		stats := puller.GetStats()
		chainID := puller.GetConfig().ServiceName
		if running, _ := stats["is_running"].(bool); running {
			summary.ActivePullers++
		}
		summary.RequestCount += int64Value(stats["request_count"])
		summary.ErrorCount += int64Value(stats["error_count"])
		if errValue := stats["last_error"]; errValue != nil && summary.LastError == "" {
			summary.LastError = fmt.Sprint(errValue)
			summary.LastErrorChainID = chainID
		}
		if loop := m.loopChains[chainID]; loop != nil {
			snapshot := loop.snapshot()
			summary.LoopRestartTotal += snapshot.RestartTotal
			summary.LoopFailureTotal += snapshot.FailureTotal
			if snapshot.LastBackoffMS > summary.LastBackoffMS {
				summary.LastBackoffMS = snapshot.LastBackoffMS
			}
			if snapshot.LastError != "" && summary.LastError == "" {
				summary.LastError = snapshot.LastError
				summary.LastErrorChainID = chainID
			}
			if snapshot.State == "backing-off" {
				summary.BackingOffChains++
			}
		}
	}

	if summary.LastError == "" && lastError != "" {
		summary.LastError = lastError
	}

	switch {
	case summary.ActivePullers == summary.PullerCount && summary.ErrorCount == 0:
		summary.Posture = "monolithic-puller-healthy"
		summary.Hint = "monolithic puller runtime is polling all configured chains without recorded errors"
	case summary.BackingOffChains > 0:
		summary.Posture = "monolithic-puller-recovering"
		summary.Hint = "monolithic puller runtime is restarting failed poll loops with bounded backoff"
	case summary.ActivePullers > 0 && summary.ErrorCount == 0:
		summary.Posture = "monolithic-puller-partial"
		summary.Hint = "monolithic puller runtime is started but not all configured chains are actively polling yet"
	case summary.ActivePullers > 0:
		summary.Posture = "monolithic-puller-degraded"
		summary.Hint = "monolithic puller runtime is polling but has recorded errors; inspect runtime summary and logs"
	default:
		summary.Posture = "monolithic-puller-idle"
		summary.Hint = "monolithic puller runtime is wired but currently has no active polling loops"
	}

	return summary
}

func (m *monolithicPullerRuntime) RuntimeControl() pluginapi.RuntimeControlCore {
	return m.PullerStatus().Control
}

func (m *monolithicPullerRuntime) HandleRuntimeControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	_ = pluginapi.WriteRuntimeControlEnvelopeWithTarget(
		w,
		"monolithic",
		pluginapi.RuntimeControlTargetPollingLoop,
		m.RuntimeControl(),
	)
}

type monolithicPullerSummary struct {
	PullerCount      int
	ActivePullers    int
	BackingOffChains int
	RequestCount     int64
	ErrorCount       int64
	LoopRestartTotal int64
	LoopFailureTotal int64
	LastBackoffMS    int64
	LastErrorChainID string
	LastError        string
	Posture          string
	Hint             string
	ControlTarget    string
	ControlPosture   string
	ControlHint      string
	Control          pluginapi.RuntimeControlCore
}

type monolithicPullLoopSnapshot struct {
	RestartTotal  int64
	FailureTotal  int64
	LastError     string
	LastBackoffMS int64
	State         string
}

func (m *monolithicPullLoopRuntime) snapshot() monolithicPullLoopSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return monolithicPullLoopSnapshot{
		RestartTotal:  m.restartTotal,
		FailureTotal:  m.failureTotal,
		LastError:     m.lastError,
		LastBackoffMS: m.lastBackoffMS,
		State:         m.state,
	}
}

type monolithicReorgSummary struct {
	Wired               bool
	ChainCount          int
	DetectedTotal       int64
	HandledTotal        int64
	LastDetectedChainID string
	LastDetectedBlock   uint64
	LastHandledChainID  string
	LastHandledBlock    uint64
	LastError           string
	Posture             string
	Hint                string
}

type monolithicChainReorgSnapshot struct {
	DetectedTotal     int64
	HandledTotal      int64
	LastDetectedBlock uint64
	LastHandledBlock  uint64
	LastError         string
}

func (m *monolithicChainReorgRuntime) snapshot() monolithicChainReorgSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return monolithicChainReorgSnapshot{
		DetectedTotal:     m.detectedTotal,
		HandledTotal:      m.handledTotal,
		LastDetectedBlock: m.lastDetectedBlock,
		LastHandledBlock:  m.lastHandledBlock,
		LastError:         m.lastError,
	}
}

func (m *monolithicChainReorgRuntime) recordDetection(block uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detectedTotal++
	m.lastDetectedBlock = block
}

func (m *monolithicChainReorgRuntime) recordHandled(block uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handledTotal++
	m.lastHandledBlock = block
}

func (m *monolithicChainReorgRuntime) recordError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err.Error()
}

func (m *monolithicChainReorgRuntime) clearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = ""
}

func subscribeMonolithicIndexer(
	ctx context.Context,
	eventBus core.EventBus,
	multiChainIndexer *indexing.MultiChainIndexer,
	logger core.Logger,
) error {
	_, err := eventBus.SubscribeNamed(ctx, monolithicEventTopic, "monolithic-indexer", func(_ context.Context, payload any) error {
		event, ok := payload.(core.BlockchainEvent)
		if !ok {
			logger.Warn("ignored unexpected monolithic event payload", "topic", monolithicEventTopic)

			return nil
		}

		logger.Info("monolithic event received", "event_name", event.EventName, "block", event.BlockNumber)
		err := multiChainIndexer.IndexEventsFromChain(ctx, event.ChainID, []*core.BlockchainEvent{&event})
		if err != nil {
			logger.Error("failed to process monolithic event", "error", err)
		}
		return nil
	})
	return err
}

func parseNodeURLs(raw string) ([]string, error) {
	nodeURLs := make([]string, 0)

	for _, nodeURL := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(nodeURL)
		if trimmed == "" {
			continue
		}

		nodeURLs = append(nodeURLs, trimmed)
	}

	if len(nodeURLs) == 0 {
		return nil, fmt.Errorf("at least one blockchain node URL is required")
	}

	return nodeURLs, nil
}

func reorgThresholdForChain(chainID string) uint64 {
	switch strings.ToLower(strings.TrimSpace(chainID)) {
	case "polygon":
		return 128
	case "bsc":
		return 15
	default:
		return 12
	}
}
