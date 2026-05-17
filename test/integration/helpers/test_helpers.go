package helpers

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// ServiceInitializer provides helpers for initializing services in integration tests
type ServiceInitializer struct {
	t *testing.T
}

// NewServiceInitializer creates a new service initializer
func NewServiceInitializer(t *testing.T) *ServiceInitializer {
	return &ServiceInitializer{t: t}
}

// InitializeQueryService initializes a query service for testing
func (si *ServiceInitializer) InitializeQueryService(
	ctx context.Context,
	mongoAdapter query.MongoDBAdapter,
	postgresAdapter query.PostgreSQLAdapter,
	cacheService query.CacheService,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) (query.QueryService, error) {
	qs := query.NewQueryService(
		mongoAdapter,
		postgresAdapter,
		cacheService,
		logger,
		metricsCollector,
	)

	// Initialize the service
	if err := qs.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize query service: %w", err)
	}

	// Start the service
	if err := qs.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start query service: %w", err)
	}

	return qs, nil
}

// InitializeCacheService initializes a cache service for testing
func (si *ServiceInitializer) InitializeCacheService(
	ctx context.Context,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) (query.CacheService, error) {
	cs := query.NewCacheService(logger, metricsCollector)

	// Initialize the service
	if err := cs.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize cache service: %w", err)
	}

	// Start the service
	if err := cs.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start cache service: %w", err)
	}

	return cs, nil
}

// CleanupService stops and cleans up a service
func (si *ServiceInitializer) CleanupService(ctx context.Context, service any) error {
	// Try to stop the service if it has a Stop method
	if stoppable, ok := service.(interface{ Stop(context.Context) error }); ok {
		if err := stoppable.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}
	return nil
}

// ResourceCleaner provides helpers for cleaning up resources in integration tests
type ResourceCleaner struct {
	t          *testing.T
	cleanupFns []func() error
}

// NewResourceCleaner creates a new resource cleaner
func NewResourceCleaner(t *testing.T) *ResourceCleaner {
	return &ResourceCleaner{
		t:          t,
		cleanupFns: []func() error{},
	}
}

// Register registers a cleanup function to be called during cleanup
func (rc *ResourceCleaner) Register(fn func() error) {
	rc.cleanupFns = append(rc.cleanupFns, fn)
}

// Cleanup executes all registered cleanup functions
func (rc *ResourceCleaner) Cleanup() error {
	// Execute cleanup functions in reverse order (LIFO)
	for i := len(rc.cleanupFns) - 1; i >= 0; i-- {
		if err := rc.cleanupFns[i](); err != nil {
			rc.t.Logf("cleanup error: %v", err)
		}
	}
	return nil
}

// TestContext provides a context with timeout for integration tests
type TestContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTestContext creates a new test context with timeout
func NewTestContext(timeout time.Duration) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &TestContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Context returns the underlying context
func (tc *TestContext) Context() context.Context {
	return tc.ctx
}

// Cancel cancels the context
func (tc *TestContext) Cancel() {
	tc.cancel()
}

// Deadline returns the deadline of the context
func (tc *TestContext) Deadline() (time.Time, bool) {
	return tc.ctx.Deadline()
}

// Done returns the done channel of the context
func (tc *TestContext) Done() <-chan struct{} {
	return tc.ctx.Done()
}

// Err returns the error of the context
func (tc *TestContext) Err() error {
	return tc.ctx.Err()
}

// Value returns the value associated with a key in the context
func (tc *TestContext) Value(key any) any {
	return tc.ctx.Value(key)
}

// WaitFor waits for a condition to be true or timeout
func WaitFor(ctx context.Context, condition func() bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// WaitForWithMessage waits for a condition to be true or timeout with a custom message
func WaitForWithMessage(ctx context.Context, condition func() bool, timeout time.Duration, message string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition: %s", message)
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(ctx context.Context, fn func() error, maxRetries int, initialBackoff time.Duration) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// CreateTestEvent creates a test blockchain event
func CreateTestEvent(chainID string, blockNum uint64, eventID string) *core.BlockchainEvent {
	return &core.BlockchainEvent{
		ID:              eventID,
		ChainID:         chainID,
		BlockNumber:     blockNum,
		TransactionHash: [32]byte{},
		EventName:       "TestEvent",
	}
}

// CreateTestEvents creates multiple test blockchain events
func CreateTestEvents(chainID string, count int) []*core.BlockchainEvent {
	events := make([]*core.BlockchainEvent, count)
	for i := 0; i < count; i++ {
		blockNumber, ok := safePositiveIntToUint64(i + 1)
		if !ok {
			blockNumber = math.MaxUint64
		}
		events[i] = CreateTestEvent(chainID, blockNumber, fmt.Sprintf("event-%d", i))
	}
	return events
}

func safePositiveIntToUint64(value int) (uint64, bool) {
	if value < 0 {
		return 0, false
	}
	return uint64(value), true
}

// TestFixtureManager manages test fixtures for integration tests
type TestFixtureManager struct {
	t         *testing.T
	fixtures  map[string]any
	cleanupFn []func() error
	mu        sync.RWMutex
}

// NewTestFixtureManager creates a new test fixture manager
func NewTestFixtureManager(t *testing.T) *TestFixtureManager {
	return &TestFixtureManager{
		t:        t,
		fixtures: make(map[string]any),
	}
}

// Register registers a fixture
func (tfm *TestFixtureManager) Register(name string, fixture any) {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	tfm.fixtures[name] = fixture
}

// Get retrieves a registered fixture
func (tfm *TestFixtureManager) Get(name string) any {
	tfm.mu.RLock()
	defer tfm.mu.RUnlock()
	return tfm.fixtures[name]
}

// RegisterCleanup registers a cleanup function
func (tfm *TestFixtureManager) RegisterCleanup(fn func() error) {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	tfm.cleanupFn = append(tfm.cleanupFn, fn)
}

// CleanupAll executes all cleanup functions
func (tfm *TestFixtureManager) CleanupAll() error {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()

	for i := len(tfm.cleanupFn) - 1; i >= 0; i-- {
		if err := tfm.cleanupFn[i](); err != nil {
			tfm.t.Logf("cleanup error: %v", err)
		}
	}

	return nil
}

// TestEventBuilder builds test events with fluent API
type TestEventBuilder struct {
	event *core.BlockchainEvent
}

// NewTestEventBuilder creates a new test event builder
func NewTestEventBuilder() *TestEventBuilder {
	return &TestEventBuilder{
		event: &core.BlockchainEvent{
			ID:          "test-event",
			ChainID:     "ethereum",
			BlockNumber: 1,
			EventName:   "TestEvent",
		},
	}
}

// WithID sets the event ID
func (teb *TestEventBuilder) WithID(id string) *TestEventBuilder {
	teb.event.ID = id
	return teb
}

// WithChainID sets the chain ID
func (teb *TestEventBuilder) WithChainID(chainID string) *TestEventBuilder {
	teb.event.ChainID = chainID
	return teb
}

// WithBlockNumber sets the block number
func (teb *TestEventBuilder) WithBlockNumber(blockNum uint64) *TestEventBuilder {
	teb.event.BlockNumber = blockNum
	return teb
}

// WithEventName sets the event name
func (teb *TestEventBuilder) WithEventName(name string) *TestEventBuilder {
	teb.event.EventName = name
	return teb
}

// Build builds the event
func (teb *TestEventBuilder) Build() *core.BlockchainEvent {
	return teb.event
}

// TestServiceBuilder builds test services with dependencies
type TestServiceBuilder struct {
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	cache              any
	database           any
	eventBus           any
	additionalServices map[string]any
}

// NewTestServiceBuilder creates a new test service builder
func NewTestServiceBuilder() *TestServiceBuilder {
	return &TestServiceBuilder{
		logger:             core.NewTestLogger(),
		metricsCollector:   core.NewTestMetricsCollector(),
		additionalServices: make(map[string]any),
	}
}

// WithLogger sets the logger
func (tsb *TestServiceBuilder) WithLogger(logger core.Logger) *TestServiceBuilder {
	tsb.logger = logger
	return tsb
}

// WithMetricsCollector sets the metrics collector
func (tsb *TestServiceBuilder) WithMetricsCollector(mc core.MetricsCollector) *TestServiceBuilder {
	tsb.metricsCollector = mc
	return tsb
}

// WithCache sets the cache
func (tsb *TestServiceBuilder) WithCache(cache any) *TestServiceBuilder {
	tsb.cache = cache
	return tsb
}

// WithDatabase sets the database
func (tsb *TestServiceBuilder) WithDatabase(db any) *TestServiceBuilder {
	tsb.database = db
	return tsb
}

// WithEventBus sets the event bus
func (tsb *TestServiceBuilder) WithEventBus(eb any) *TestServiceBuilder {
	tsb.eventBus = eb
	return tsb
}

// WithService adds an additional service
func (tsb *TestServiceBuilder) WithService(name string, service any) *TestServiceBuilder {
	tsb.additionalServices[name] = service
	return tsb
}

// GetLogger returns the logger
func (tsb *TestServiceBuilder) GetLogger() core.Logger {
	return tsb.logger
}

// GetMetricsCollector returns the metrics collector
func (tsb *TestServiceBuilder) GetMetricsCollector() core.MetricsCollector {
	return tsb.metricsCollector
}

// GetCache returns the cache
func (tsb *TestServiceBuilder) GetCache() any {
	return tsb.cache
}

// GetDatabase returns the database
func (tsb *TestServiceBuilder) GetDatabase() any {
	return tsb.database
}

// GetEventBus returns the event bus
func (tsb *TestServiceBuilder) GetEventBus() any {
	return tsb.eventBus
}

// GetService returns an additional service
func (tsb *TestServiceBuilder) GetService(name string) any {
	return tsb.additionalServices[name]
}

// TestTimeoutManager manages timeouts for integration tests
type TestTimeoutManager struct {
	defaultTimeout time.Duration
	customTimeouts map[string]time.Duration
	mu             sync.RWMutex
}

// NewTestTimeoutManager creates a new test timeout manager
func NewTestTimeoutManager(defaultTimeout time.Duration) *TestTimeoutManager {
	return &TestTimeoutManager{
		defaultTimeout: defaultTimeout,
		customTimeouts: make(map[string]time.Duration),
	}
}

// SetTimeout sets a custom timeout for a test
func (ttm *TestTimeoutManager) SetTimeout(testName string, timeout time.Duration) {
	ttm.mu.Lock()
	defer ttm.mu.Unlock()
	ttm.customTimeouts[testName] = timeout
}

// GetTimeout gets the timeout for a test
func (ttm *TestTimeoutManager) GetTimeout(testName string) time.Duration {
	ttm.mu.RLock()
	defer ttm.mu.RUnlock()

	if timeout, exists := ttm.customTimeouts[testName]; exists {
		return timeout
	}

	return ttm.defaultTimeout
}

// TestMetricsCollector collects metrics during tests
type TestMetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]any
}

// NewTestMetricsCollector creates a new test metrics collector
func NewTestMetricsCollector() *TestMetricsCollector {
	return &TestMetricsCollector{
		metrics: make(map[string]any),
	}
}

// RecordMetric records a metric
func (tmc *TestMetricsCollector) RecordMetric(name string, value any) {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	tmc.metrics[name] = value
}

// GetMetric retrieves a metric
func (tmc *TestMetricsCollector) GetMetric(name string) any {
	tmc.mu.RLock()
	defer tmc.mu.RUnlock()
	return tmc.metrics[name]
}

// GetAllMetrics retrieves all metrics
func (tmc *TestMetricsCollector) GetAllMetrics() map[string]any {
	tmc.mu.RLock()
	defer tmc.mu.RUnlock()

	metrics := make(map[string]any)
	for k, v := range tmc.metrics {
		metrics[k] = v
	}

	return metrics
}

// Reset clears all metrics
func (tmc *TestMetricsCollector) Reset() {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	tmc.metrics = make(map[string]any)
}
