package pullers

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/evm"
)

const defaultPullerConnectionTimeout = 30 * time.Second

// BaseDataPullerPlugin provides common functionality for data puller plugins
type BaseDataPullerPlugin struct {
	mu                 sync.RWMutex
	name               string
	version            string
	config             core.Config
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	eventBus           core.EventBus
	checkpointStore    core.CheckpointStore
	isRunning          bool
	lastBlockNumber    uint64
	maxRetries         int
	retryBackoff       time.Duration
	connectionTimeout  time.Duration
	inFlight           sync.WaitGroup  // tracks in-flight operations
	shutdownTimeout    time.Duration   // max wait for in-flight ops on Stop()
	lifecycleCtx       context.Context // context for async operations like checkpoint persistence
	errorCounter       int64
	requestCounter     int64
	lastError          error
	lastErrorTime      time.Time
	lastSuccessfulPull time.Time // zero = never pulled

	// If set, Health() calls this to check RPC reachability.
	// Pullers set this in Start() once a client connection is established.
	rpcHealthCheck func(context.Context) error

	// circuit breaker protects RPC calls from cascading failures.
	// Created by NewBaseDataPullerPlugin if config.CircuitBreakerConfig is set.
	circuitBreaker *CircuitBreaker
}

// SetRPCHealthCheck registers a function that Health() calls to verify RPC
// node reachability. Pullers should call this in Start() after connecting.
func (p *BaseDataPullerPlugin) SetRPCHealthCheck(fn func(context.Context) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rpcHealthCheck = fn
}

// CircuitBreaker returns the circuit breaker for RPC protection.
// Pullers should call cb.Allow() before making RPC calls.
func (p *BaseDataPullerPlugin) CircuitBreaker() *CircuitBreaker {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.circuitBreaker
}

// NewBaseDataPullerPlugin creates a new base data puller plugin
func NewBaseDataPullerPlugin(
	name string,
	version string,
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
) *BaseDataPullerPlugin {
	shutdownTimeout := 10 * time.Second
	if config.ShutdownTimeout > 0 {
		shutdownTimeout = time.Duration(config.ShutdownTimeout) * time.Millisecond
	}
	return &BaseDataPullerPlugin{
		name:              name,
		version:           version,
		config:            config,
		logger:            logger,
		metricsCollector:  metricsCollector,
		eventBus:          eventBus,
		isRunning:         false,
		lastBlockNumber:   config.StartBlock,
		maxRetries:        config.MaxRetries,
		retryBackoff:      time.Duration(config.RetryBackoff) * time.Millisecond,
		connectionTimeout: defaultPullerConnectionTimeout,
		shutdownTimeout:   shutdownTimeout,
		circuitBreaker:    NewCircuitBreaker(DefaultCircuitBreakerConfig),
	}
}

// Name returns the plugin name
func (p *BaseDataPullerPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BaseDataPullerPlugin) Version() string {
	return p.version
}

// Initialize initializes the plugin
func (p *BaseDataPullerPlugin) Initialize(_ context.Context, config core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config
	p.lastBlockNumber = config.StartBlock
	p.maxRetries = config.MaxRetries
	p.retryBackoff = time.Duration(config.RetryBackoff) * time.Millisecond

	if p.logger != nil {
		p.logger.Info("data puller plugin initialized", "name", p.name, "start_block", config.StartBlock)
	}

	return nil
}

// Start starts the plugin
func (p *BaseDataPullerPlugin) Start(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	// Validate chain ID before starting to prevent cross-chain data contamination
	if p.config.ChainID == "" && p.config.ServiceName == "" {
		return fmt.Errorf("puller chainID is not configured: either ChainID or ServiceName must be set")
	}

	p.isRunning = true

	if p.logger != nil {
		p.logger.Info("data puller plugin started", "name", p.name)
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("plugin_starts", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// Stop stops the plugin and waits for in-flight operations to complete
func (p *BaseDataPullerPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.isRunning = false
	p.mu.Unlock()

	// Wait for in-flight operations with context
	done := make(chan struct{})
	go func() {
		defer handlePullerPanic(p.logger, "data_puller.Stop.inFlight")
		p.inFlight.Wait()
		close(done)
	}()

	timeout := p.shutdownTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	select {
	case <-done:
		if p.logger != nil {
			p.logger.Info("data puller plugin stopped, all in-flight operations completed", "name", p.name)
		}
	case <-time.After(timeout):
		if p.logger != nil {
			p.logger.Warn("data puller plugin stop timed out waiting for in-flight operations", "name", p.name, "timeout", timeout)
		}
	case <-ctx.Done():
		if p.logger != nil {
			p.logger.Warn("data puller plugin stop cancelled by context", "name", p.name)
		}
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("plugin_stops", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// Health checks the health of the plugin.
// Reports degraded if no successful pull within 5 minutes or a recent error exists.
// If an RPC health check function is set (via SetRPCHealthCheck), calls it to verify
// node reachability with the given context timeout.
// Reports circuit breaker open/half-open state as degraded.
func (p *BaseDataPullerPlugin) Health(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("health check interrupted: %w", ctx.Err())
	default:
	}

	p.mu.RLock()
	running := p.isRunning
	lastPull := p.lastSuccessfulPull
	lastErr := p.lastError
	lastErrTime := p.lastErrorTime
	checkFn := p.rpcHealthCheck
	cb := p.circuitBreaker
	p.mu.RUnlock()

	if !running {
		return fmt.Errorf("plugin not running")
	}

	if cb != nil {
		cbState := cb.State()
		if cbState == CircuitBreakerOpen {
			failures, threshold := cb.Counts()
			return fmt.Errorf("RPC circuit breaker open (%d/%d failures): %w", failures, threshold, core.ErrRPCUnreachable)
		}
		if cbState == CircuitBreakerHalfOpen {
			return fmt.Errorf("RPC circuit breaker half-open, recovery in progress: %w", core.ErrRPCUnreachable)
		}
	}

	if !lastPull.IsZero() && time.Since(lastPull) > 5*time.Minute {
		return fmt.Errorf("no successful pull in %v", time.Since(lastPull).Round(time.Second))
	}

	if lastErr != nil && time.Since(lastErrTime) < 10*time.Minute {
		return fmt.Errorf("recent error: %w, at %v", lastErr, lastErrTime.Format(time.RFC3339))
	}

	if checkFn != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := checkFn(probeCtx); err != nil {
			return fmt.Errorf("RPC health check failed: %w", err)
		}
	}

	return nil
}

// RecordSuccessfulPull records the time of a successful data pull.
// Also resets the circuit breaker to closed state.
func (p *BaseDataPullerPlugin) RecordSuccessfulPull() {
	p.mu.Lock()
	p.lastSuccessfulPull = time.Now()
	p.lastError = nil
	p.lastErrorTime = time.Time{}
	p.errorCounter = 0
	cb := p.circuitBreaker
	p.mu.Unlock()

	if cb != nil {
		cb.Reset()
	}
}

// IsRunning returns whether the plugin is running
func (p *BaseDataPullerPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// RecordError records an error for stats and logging.
// Also feeds the circuit breaker — N consecutive errors will trip it open.
func (p *BaseDataPullerPlugin) RecordError(err error) {
	p.mu.Lock()
	p.errorCounter++
	p.lastError = err
	p.lastErrorTime = time.Now()
	cb := p.circuitBreaker
	p.mu.Unlock()

	if cb != nil {
		cb.Failure()
	}
}

// RecordRequest records a successful request attempt for stats.
func (p *BaseDataPullerPlugin) RecordRequest() {
	p.mu.Lock()
	p.requestCounter++
	p.mu.Unlock()
}

// BaseStats returns common puller statistics
func (p *BaseDataPullerPlugin) BaseStats() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return map[string]any{
		"request_count":   p.requestCounter,
		"error_count":     p.errorCounter,
		"last_error":      p.lastError,
		"last_error_time": p.lastErrorTime,
		"is_running":      p.isRunning,
	}
}

// GetLastBlockNumber returns the last processed block number
func (p *BaseDataPullerPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastBlockNumber
}

// handlePullerPanic recovers from panics in puller goroutines to prevent
// process-wide crashes. All puller goroutines should defer this.
func handlePullerPanic(logger core.Logger, source string) {
	if r := recover(); r != nil {
		if logger != nil {
			logger.Error(
				"puller goroutine panicked",
				"source", source,
				"panic", fmt.Sprintf("%v", r),
			)
		}
	}
}

// SetLastBlockNumber sets the last processed block number and persists it if checkpointStore is configured.
// The block hash is not persisted when using this method; use SetLastBlockNumberWithHash
// for reorg-aware checkpointing.
func (p *BaseDataPullerPlugin) SetLastBlockNumber(blockNumber uint64) {
	p.SetLastBlockNumberWithHash(blockNumber, "")
}

// SetLastBlockNumberWithHash sets the last processed block number and block hash,
// and persists both if checkpointStore is configured.
// The stored hash enables reorg detection on startup by verifying chain continuity.
func (p *BaseDataPullerPlugin) SetLastBlockNumberWithHash(blockNumber uint64, blockHash string) {
	p.mu.Lock()
	p.lastBlockNumber = blockNumber
	store := p.checkpointStore
	chainID := p.config.ChainID
	if chainID == "" {
		chainID = p.config.ServiceName
	}
	p.mu.Unlock()

	// Persist checkpoint synchronously to avoid the race between in-memory
	// update and crash: if we used a goroutine, a crash after the in-memory
	// set but before the DB write would cause duplicate processing on restart.
	if store != nil && chainID != "" {
		ctx := p.lifecycleCtx
		if ctx == nil {
			ctx = context.Background()
		}

		const maxRetries = 3
		var lastErr error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
			}
			if err := store.SaveLastIndexedBlock(ctx, chainID, blockNumber, blockHash); err != nil {
				lastErr = err
				continue
			}
			return // success
		}
		if p.logger != nil {
			p.logger.Error(
				"checkpoint persist failed after retries",
				"chainID", chainID,
				"block", blockNumber,
				"error", lastErr,
			)
		}
	}
}

// SetCheckpointStore sets the checkpoint store for persisting indexing progress
func (p *BaseDataPullerPlugin) SetCheckpointStore(store core.CheckpointStore) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkpointStore = store
}

// SetLifecycleContext sets the lifecycle context for async operations like
// checkpoint persistence. Call this from the concrete puller's Start() method.
func (p *BaseDataPullerPlugin) SetLifecycleContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lifecycleCtx = ctx
}

// LoadCheckpoint loads the last indexed block from the checkpoint store.
// Returns the stored block number, or 0 if no checkpoint exists or loading fails.
// The block hash is discarded; use LoadCheckpointWithHash for reorg-aware startup.
func (p *BaseDataPullerPlugin) LoadCheckpoint(ctx context.Context) uint64 {
	blockNum, _ := p.LoadCheckpointWithHash(ctx)
	return blockNum
}

// LoadCheckpointWithHash loads the last indexed block number and its block hash.
// Returns (0, "") if no checkpoint exists or loading fails.
// The stored hash can be compared against the chain to detect reorgs on startup.
func (p *BaseDataPullerPlugin) LoadCheckpointWithHash(ctx context.Context) (uint64, string) {
	p.mu.RLock()
	store := p.checkpointStore
	chainID := p.config.ChainID
	if chainID == "" {
		chainID = p.config.ServiceName
	}
	if chainID == "" {
		if p.logger != nil {
			p.logger.Warn("chainID is empty; checkpoint cannot be loaded reliably")
		}
	}
	p.mu.RUnlock()

	if store == nil {
		return 0, ""
	}

	blockNum, blockHash, err := store.GetLastIndexedBlock(ctx, chainID)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("failed to load checkpoint", "error", err.Error())
		}
		return 0, ""
	}
	return blockNum, blockHash
}

// PublishEvent publishes an event to the event bus
func (p *BaseDataPullerPlugin) PublishEvent(ctx context.Context, event core.BlockchainEvent) error {
	if p.eventBus == nil {
		return fmt.Errorf("event bus not available")
	}

	event.ProcessedAt = time.Now().UTC()
	event.Status = "published"

	if err := p.eventBus.Publish(ctx, topics.TopicBlockchainEvents, event); err != nil {
		if p.logger != nil {
			p.logger.Error("failed to publish event", "error", err.Error(), "block", event.BlockNumber)
		}
		return err
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("events_published", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// PublishEvents publishes multiple events to the event bus
func (p *BaseDataPullerPlugin) PublishEvents(ctx context.Context, events []core.BlockchainEvent) error {
	for _, event := range events {
		if err := p.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// RecordMetric records a metric
func (p *BaseDataPullerPlugin) RecordMetric(name string, value any, tags map[string]string) {
	if p.metricsCollector == nil {
		return
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	tags["plugin"] = p.name

	switch v := value.(type) {
	case int64:
		p.metricsCollector.RecordCounter(name, v, tags)
	case float64:
		p.metricsCollector.RecordGauge(name, v, tags)
	}
}

// LogInfo logs an info message
func (p *BaseDataPullerPlugin) LogInfo(msg string, fields ...any) {
	if p.logger != nil {
		p.logger.Info(msg, fields...)
	}
}

// LogError logs an error message
func (p *BaseDataPullerPlugin) LogError(msg string, fields ...any) {
	if p.logger != nil {
		p.logger.Error(msg, fields...)
	}
}

// LogWarn logs a warning message
func (p *BaseDataPullerPlugin) LogWarn(msg string, fields ...any) {
	if p.logger != nil {
		p.logger.Warn(msg, fields...)
	}
}

// RetryWithBackoff retries an operation with exponential backoff and error classification.
// Different error types receive different retry strategies:
//   - Rate limit (429): longer backoff, more retries
//   - Timeout: moderate backoff, standard retries
//   - Network/Connection: standard backoff, standard retries
//   - Non-retryable (auth, invalid params): no retry, immediate return
func (p *BaseDataPullerPlugin) RetryWithBackoff(ctx context.Context, operation func() error) error {
	backoff := p.retryBackoff
	var lastErr error

	if p.maxRetries <= 0 {
		p.maxRetries = 3
		p.retryBackoff = 1 * time.Second
		backoff = p.retryBackoff
	}

	p.logger.Debug("retry config", "maxRetries", p.maxRetries, "retryBackoff", backoff)

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		// Classify error to determine retry strategy
		errType := classifyError(err)
		if !errType.retryable {
			p.logger.Error("non-retryable error, aborting", "error", err.Error(), "type", errType.category)
			return err
		}

		// Adjust backoff based on error type
		effectiveBackoff := backoff
		if errType.category == "rate_limit" {
			// Rate limits: use longer backoff (at least 5s)
			effectiveBackoff = max(backoff, 5*time.Second)
		}

		p.logger.Warn("retry attempt failed", "attempt", attempt+1, "maxRetries", p.maxRetries, "error", err, "error_type", errType.category)

		if attempt < p.maxRetries-1 {
			// Add jitter: randomize between 50%-100% of backoff to avoid thundering herd
			jitteredBackoff := time.Duration(float64(effectiveBackoff) * (0.5 + rand.Float64()*0.5))
			select {
			case <-time.After(jitteredBackoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	p.logger.Error("all retries exhausted", "maxRetries", p.maxRetries, "lastError", lastErr)
	if lastErr != nil {
		return fmt.Errorf("max retries exceeded: last error: %w", lastErr)
	}
	return fmt.Errorf("max retries exceeded: unknown error")
}

// errorClassification describes retry characteristics of an error.
type errorClassification struct {
	retryable bool
	category  string // "rate_limit", "timeout", "network", "non_retryable"
}

// classifyError examines an error and determines its retry strategy.
func classifyError(err error) errorClassification {
	if err == nil {
		return errorClassification{retryable: true, category: "none"}
	}

	errStr := err.Error()

	// Rate limit errors (HTTP 429, "rate limit", "too many requests")
	if strings.Contains(errStr, "429") ||
		strings.Contains(strings.ToLower(errStr), "rate limit") ||
		strings.Contains(strings.ToLower(errStr), "too many requests") ||
		strings.Contains(strings.ToLower(errStr), "throttl") {
		return errorClassification{retryable: true, category: "rate_limit"}
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "i/o timeout") {
		return errorClassification{retryable: true, category: "timeout"}
	}

	// Network/Connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") {
		return errorClassification{retryable: true, category: "network"}
	}

	// Non-retryable errors
	if strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(strings.ToLower(errStr), "unauthorized") ||
		strings.Contains(strings.ToLower(errStr), "forbidden") ||
		strings.Contains(strings.ToLower(errStr), "invalid api key") ||
		strings.Contains(strings.ToLower(errStr), "invalid param") ||
		strings.Contains(strings.ToLower(errStr), "unknown method") {
		return errorClassification{retryable: false, category: "non_retryable"}
	}

	// Default: treat as retryable network-type error
	return errorClassification{retryable: true, category: "unknown"}
}

// GetConfig returns the plugin configuration
func (p *BaseDataPullerPlugin) GetConfig() core.Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// ChainID returns the per-chain identifier for this puller instance.
// It checks Config.ChainID first, falls back to ServiceName for backward
// compatibility. Returns empty string if neither is configured — the caller
// must validate before use.
func (p *BaseDataPullerPlugin) ChainID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.ChainID != "" {
		return p.config.ChainID
	}
	if p.config.ServiceName != "" {
		return p.config.ServiceName
	}
	return ""
}

// ValidateConfig validates that the puller has a valid chain ID configured.
// Returns an error if ChainID would be empty, which would cause data contamination.
func (p *BaseDataPullerPlugin) ValidateConfig() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.ChainID == "" && p.config.ServiceName == "" {
		return fmt.Errorf("puller chainID is not configured: either ChainID or ServiceName must be set to prevent cross-chain data contamination")
	}
	return nil
}

// Network returns the network label for this puller's chain.
// Falls back to "mainnet" if not explicitly configured.
func (p *BaseDataPullerPlugin) Network() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.Network != "" {
		return p.config.Network
	}
	return "mainnet"
}

// SetConnectionTimeout sets the connection timeout
func (p *BaseDataPullerPlugin) SetConnectionTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connectionTimeout = timeout
}

// GetConnectionTimeout returns the connection timeout
func (p *BaseDataPullerPlugin) GetConnectionTimeout() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connectionTimeout
}

// ValidateEvent validates a blockchain event
func (p *BaseDataPullerPlugin) ValidateEvent(event core.BlockchainEvent) error {
	if event.BlockNumber == 0 {
		return fmt.Errorf("invalid block number")
	}
	if event.TransactionHash == (common.Hash{}) {
		return fmt.Errorf("invalid transaction hash")
	}
	if event.ContractAddress == (common.Address{}) {
		return fmt.Errorf("invalid contract address")
	}
	if event.EventName == "" {
		return fmt.Errorf("invalid event name")
	}

	// Client-side contract address allowlist validation.
	// The server-side eth_getLogs address filter is the primary filter,
	// but a compromised RPC could return events from unmonitored contracts.
	if addresses := p.GetConfig().ContractAddresses; len(addresses) > 0 {
		eventAddr := event.ContractAddress.Hex()
		found := false
		for _, addr := range addresses {
			if strings.EqualFold(eventAddr, addr) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("contract address %s not in monitored list", eventAddr)
		}
	}

	return nil
}

// GenerateEventHash generates a deterministic hash for an event using the
// canonical ComputeEventHash function from pkg/core.
func (p *BaseDataPullerPlugin) GenerateEventHash(event core.BlockchainEvent) string {
	return core.ComputeEventHash(&event)
}

// BuildBlockchainEvent constructs a core.BlockchainEvent from parsed log fields.
// Both grpc and websocket pullers use this after converting their hex-encoded
// Log struct fields into native Go types.
func (p *BaseDataPullerPlugin) BuildBlockchainEvent(
	chainID, network string,
	txHash common.Hash,
	blockNumber uint64,
	logIndex uint64,
	contractAddr common.Address,
	eventData []byte,
	eventTopics []common.Hash,
	eventName string,
	eventSig common.Hash,
	blockTimestamp int64,
	removed bool,
) core.BlockchainEvent {
	event := core.BlockchainEvent{
		ID:              chainID + "-" + txHash.Hex() + "-" + strconv.FormatUint(logIndex, 10),
		BlockNumber:     blockNumber,
		TransactionHash: txHash,
		LogIndex:        logIndex,
		ContractAddress: contractAddr,
		EventName:       eventName,
		EventSignature:  eventSig,
		EventData:       eventData,
		DecodedData:     evm.DecodeEvent(eventName, eventTopics, eventData),
		ChainID:         chainID,
		Network:         network,
		BlockTimestamp:  blockTimestamp,
		Status:          core.EventStatusPending,
		Removed:         removed,
	}
	event.EventHash = p.GenerateEventHash(event)
	return event
}
