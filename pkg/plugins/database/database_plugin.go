package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// DatabaseStats tracks database performance metrics.
//
// Renaming would break many external uses.
type DatabaseStats struct {
	WriteCount     int64
	ReadCount      int64
	DeleteCount    int64
	ErrorCount     int64
	TotalSize      int64
	EventCount     int64
	AvgWriteTimeMs float64
	AvgReadTimeMs  float64
	LastWriteTime  time.Time
	LastReadTime   time.Time
}

// BaseDatabasePlugin provides base implementation for database plugins
type BaseDatabasePlugin struct {
	mu               sync.RWMutex
	initialized      bool
	running          bool
	config           *core.Config
	logger           core.Logger
	metricsCollector core.MetricsCollector
	lastHealthCheck  *core.HealthStatus
	writeCount       int64
	readCount        int64
	deleteCount      int64
	errorCount       int64
	totalSize        int64
	eventCount       int64
	totalWriteTime   int64 // in milliseconds
	totalReadTime    int64 // in milliseconds
	lastWriteTime    time.Time
	lastReadTime     time.Time
}

// NewBaseDatabasePlugin creates a new base database plugin
func NewBaseDatabasePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *BaseDatabasePlugin {
	return &BaseDatabasePlugin{
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Name returns the plugin name
func (p *BaseDatabasePlugin) Name() string {
	return "database"
}

// Version returns the plugin version
func (p *BaseDatabasePlugin) Version() string {
	return "1.0.0"
}

// Initialize initializes the database plugin
func (p *BaseDatabasePlugin) Initialize(_ context.Context, config core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("database plugin already initialized")
	}

	p.config = &config
	p.initialized = true

	p.logger.Info("Database plugin initialized", logkeys.LogKeyComponent, "database")

	return nil
}

// Start starts the database plugin
func (p *BaseDatabasePlugin) Start(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("database plugin not initialized")
	}

	if p.running {
		return fmt.Errorf("database plugin already running")
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Database plugin started",
	}

	p.logger.Info("Database plugin started", logkeys.LogKeyComponent, "database")

	return nil
}

// Stop stops the database plugin
func (p *BaseDatabasePlugin) Stop(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	p.running = false
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Database plugin stopped",
	}

	p.logger.Info("Database plugin stopped", logkeys.LogKeyComponent, "database")

	return nil
}

// Health returns the health status of the plugin
func (p *BaseDatabasePlugin) Health(_ context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return fmt.Errorf("database plugin not initialized")
	}

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	return nil
}

// RecordWrite records a database write operation
func (p *BaseDatabasePlugin) RecordWrite(durationMs int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.recordWriteUnlocked(durationMs)
}

// recordWriteUnlocked records a write without acquiring a lock (must be called while holding lock)
func (p *BaseDatabasePlugin) recordWriteUnlocked(durationMs int64) {
	p.writeCount++
	p.totalWriteTime += durationMs
	p.lastWriteTime = time.Now()
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("database_write", 1, map[string]string{})
		p.metricsCollector.RecordHistogram("database_write_time_ms", float64(durationMs), map[string]string{})
	}
}

// RecordRead records a database read operation
func (p *BaseDatabasePlugin) RecordRead(durationMs int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.recordReadUnlocked(durationMs)
}

// recordReadUnlocked records a read without acquiring a lock (must be called while holding lock)
func (p *BaseDatabasePlugin) recordReadUnlocked(durationMs int64) {
	p.readCount++
	p.totalReadTime += durationMs
	p.lastReadTime = time.Now()
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("database_read", 1, map[string]string{})
		p.metricsCollector.RecordHistogram("database_read_time_ms", float64(durationMs), map[string]string{})
	}
}

// RecordDelete records a database delete operation
func (p *BaseDatabasePlugin) RecordDelete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.deleteCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("database_delete", 1, map[string]string{})
	}
}

// RecordError records a database error
func (p *BaseDatabasePlugin) RecordError() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount++
	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("database_error", 1, map[string]string{})
	}
}

// UpdateEventCount updates the event count
func (p *BaseDatabasePlugin) UpdateEventCount(count int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.updateEventCountUnlocked(count)
}

// updateEventCountUnlocked updates event count without acquiring a lock (must be called while holding lock)
func (p *BaseDatabasePlugin) updateEventCountUnlocked(count int64) {
	p.eventCount = count
	if p.metricsCollector != nil {
		p.metricsCollector.RecordGauge("database_event_count", float64(count), map[string]string{})
	}
}

// UpdateTotalSize updates the total size
func (p *BaseDatabasePlugin) UpdateTotalSize(size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalSize = size
	p.metricsCollector.RecordGauge("database_total_size", float64(size), map[string]string{})
}

// GetWriteCount returns the write count
func (p *BaseDatabasePlugin) GetWriteCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.writeCount
}

// GetReadCount returns the read count
func (p *BaseDatabasePlugin) GetReadCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.readCount
}

// GetDeleteCount returns the delete count
func (p *BaseDatabasePlugin) GetDeleteCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.deleteCount
}

// GetErrorCount returns the error count
func (p *BaseDatabasePlugin) GetErrorCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.errorCount
}

// GetEventCount returns the event count
func (p *BaseDatabasePlugin) GetEventCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.eventCount
}

// GetTotalSize returns the total size
func (p *BaseDatabasePlugin) GetTotalSize() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.totalSize
}

// DefaultInMemoryDatabasePlugin provides in-memory database implementation
type DefaultInMemoryDatabasePlugin struct {
	*BaseDatabasePlugin
	events map[string]*blockchain.BlockchainEvent
}

// NewDefaultInMemoryDatabasePlugin creates a new in-memory database plugin
func NewDefaultInMemoryDatabasePlugin(logger core.Logger, metricsCollector core.MetricsCollector) *DefaultInMemoryDatabasePlugin {
	return &DefaultInMemoryDatabasePlugin{
		BaseDatabasePlugin: NewBaseDatabasePlugin(logger, metricsCollector),
		events:             make(map[string]*blockchain.BlockchainEvent),
	}
}

// WriteEvent writes a blockchain event to the database
func (p *DefaultInMemoryDatabasePlugin) WriteEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	if event.EventHash == "" {
		return fmt.Errorf("event hash is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	start := time.Now()
	p.events[event.EventHash] = event
	duration := time.Since(start).Milliseconds()

	p.recordWriteUnlocked(duration)
	p.updateEventCountUnlocked(int64(len(p.events)))

	return nil
}

// WriteEvents writes multiple blockchain events to the database (batch)
func (p *DefaultInMemoryDatabasePlugin) WriteEvents(ctx context.Context, events []blockchain.BlockchainEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("events list is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	start := time.Now()

	for i := range events {
		if events[i].EventHash == "" {
			p.errorCount++
			if p.metricsCollector != nil {
				p.metricsCollector.RecordCounter("database_error", 1, map[string]string{})
			}
			return fmt.Errorf("event hash is required for event %d", i)
		}
		p.events[events[i].EventHash] = &events[i]
	}

	duration := time.Since(start).Milliseconds()
	p.recordWriteUnlocked(duration)
	p.updateEventCountUnlocked(int64(len(p.events)))

	return nil
}

// WriteBatch writes multiple blockchain events atomically (in-memory batch).
func (p *DefaultInMemoryDatabasePlugin) WriteBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	start := time.Now()

	for _, event := range events {
		if event == nil {
			continue
		}
		if event.EventHash == "" {
			p.errorCount++
			return fmt.Errorf("event hash is required")
		}
		p.events[event.EventHash] = event
	}

	duration := time.Since(start).Milliseconds()
	p.recordWriteUnlocked(duration)
	p.updateEventCountUnlocked(int64(len(p.events)))

	return nil
}

// QueryEvents queries events from the database
func (p *DefaultInMemoryDatabasePlugin) QueryEvents(filter *core.EventFilter) (*core.QueryResult, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}

	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	start := time.Now()

	// Filter events
	results := make([]blockchain.BlockchainEvent, 0, len(p.events))
	for _, event := range p.events {
		// Check contract address filter
		if len(filter.ContractAddress) > 0 {
			found := false
			for _, addr := range filter.ContractAddress {
				if event.ContractAddress == addr {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Check block range filter
		if filter.FromBlock > 0 && event.BlockNumber < filter.FromBlock {
			continue
		}
		if filter.ToBlock > 0 && event.BlockNumber > filter.ToBlock {
			continue
		}
		results = append(results, *event)
	}

	// Apply limit and offset
	total := int64(len(results))
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	duration := time.Since(start).Milliseconds()
	p.mu.RUnlock()
	p.RecordRead(duration)

	return &core.QueryResult{
		Events:       results,
		Total:        total,
		CacheHit:     false,
		ResponseTime: duration,
	}, nil
}

// GetEventByHash retrieves an event by its hash
func (p *DefaultInMemoryDatabasePlugin) GetEventByHash(hash string) (*blockchain.BlockchainEvent, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	start := time.Now()

	event, exists := p.events[hash]
	if !exists {
		p.mu.RUnlock()
		return nil, nil
	}

	duration := time.Since(start).Milliseconds()
	p.mu.RUnlock()
	p.RecordRead(duration)

	return event, nil
}

// DeleteEvent deletes an event from the database by ID
func (p *DefaultInMemoryDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("database plugin not running")
	}

	// Find and delete event by EventHash (eventID is actually the EventHash)
	if _, exists := p.events[eventID]; exists {
		delete(p.events, eventID)
		p.deleteCount++
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("database_delete", 1, map[string]string{})
		}
		p.updateEventCountUnlocked(int64(len(p.events)))
		return nil
	}

	return nil
}

// GetStats returns database statistics
func (p *DefaultInMemoryDatabasePlugin) GetStats() *DatabaseStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgWriteTime := 0.0
	if p.writeCount > 0 {
		avgWriteTime = float64(p.totalWriteTime) / float64(p.writeCount)
	}

	avgReadTime := 0.0
	if p.readCount > 0 {
		avgReadTime = float64(p.totalReadTime) / float64(p.readCount)
	}

	return &DatabaseStats{
		WriteCount:     p.writeCount,
		ReadCount:      p.readCount,
		DeleteCount:    p.deleteCount,
		ErrorCount:     p.errorCount,
		TotalSize:      p.totalSize,
		EventCount:     p.eventCount,
		AvgWriteTimeMs: avgWriteTime,
		AvgReadTimeMs:  avgReadTime,
		LastWriteTime:  p.lastWriteTime,
		LastReadTime:   p.lastReadTime,
	}
}

// GetAllEvents retrieves all events from the database
func (p *DefaultInMemoryDatabasePlugin) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	events := make([]*blockchain.BlockchainEvent, 0, len(p.events))
	for _, event := range p.events {
		events = append(events, event)
	}

	p.mu.RUnlock()
	p.RecordRead(0)
	return events, nil
}

// GetAllBlocks retrieves all blocks from the database
func (p *DefaultInMemoryDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	// In-memory implementation doesn't store blocks
	p.mu.RUnlock()
	p.RecordRead(0)
	return []*blockchain.Block{}, nil
}

// GetEventsByBlockRange retrieves events within a block range
func (p *DefaultInMemoryDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	var events []*blockchain.BlockchainEvent
	for _, event := range p.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			events = append(events, event)
		}
	}

	p.mu.RUnlock()
	p.RecordRead(0)
	return events, nil
}

// GetBlock retrieves a specific block by number
func (p *DefaultInMemoryDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	// In-memory implementation doesn't store blocks
	p.mu.RUnlock()
	p.RecordRead(0)
	return nil, nil
}

// GetLatestBlock retrieves the latest block number
func (p *DefaultInMemoryDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return 0, fmt.Errorf("database plugin not running")
	}

	// In-memory implementation doesn't store blocks
	p.mu.RUnlock()
	p.RecordRead(0)
	return 0, nil
}

// DeleteEventsByBlockRange deletes events within a block range
func (p *DefaultInMemoryDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return 0, fmt.Errorf("database plugin not running")
	}

	count := int64(0)
	for key, event := range p.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			delete(p.events, key)
			count++
		}
	}

	if count > 0 {
		p.deleteCount++
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("database_delete", 1, map[string]string{})
		}
		p.updateEventCountUnlocked(int64(len(p.events)))
	}

	return count, nil
}

// MarkEventsAsReorged marks events within a block range as reorged (soft delete)
func (p *DefaultInMemoryDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return 0, fmt.Errorf("database plugin not running")
	}

	count := int64(0)
	for key, event := range p.events {
		if event.BlockNumber >= fromBlock && event.BlockNumber <= toBlock {
			event.Status = blockchain.EventStatusReorged
			p.events[key] = event
			count++
		}
	}

	return count, nil
}

// GetReorgStats retrieves reorg statistics
func (p *DefaultInMemoryDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	p.mu.RLock()

	if !p.running {
		p.mu.RUnlock()
		return nil, fmt.Errorf("database plugin not running")
	}

	// In-memory implementation returns empty stats
	p.mu.RUnlock()
	p.RecordRead(0)
	return &core.ReorgStats{}, nil
}
